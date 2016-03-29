package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"taskmaster/common"
	"taskmaster/log"
)

func (h *Handler) updateWhatMustBeUpdated(newConf map[string]*common.Process) {
	var toRestart []string
	var useless []common.ProcStatus
	for k := range newConf {
		if proc, exists := getProc(k); exists {
			procState := proc.GetProcStatus()
			if procState.State == common.Running || procState.State == common.Starting {
				if mustBeRestarted(g_procs[k], newConf[k]) {
					toRestart = append(toRestart, k)
					h.StopProc(k, &useless)
					replaceProcess(k, newConf)
				} else {
					updateProc(g_procs[k], newConf[k])
				}
			} else if procState.State == common.Backoff {
				toRestart = append(toRestart, k)
				h.StopProc(k, &useless)
				replaceProcess(k, newConf)
			} else {
				newConf[k].State = procState.State
				replaceProcess(k, newConf)
			}
		} else {
			replaceProcess(k, newConf)
		}
	}
	for _, name := range toRestart {
		h.StartProc(name, &useless)
	}
}

func listenSIGHUP(filename string, h *Handler) {
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGHUP)
	go func() {
		for {
			<-sig
			h.ReloadConfig("lol", &[]common.ProcStatus{})
		}
	}()
}

func loadFileSlice(filename string) ([]*common.Process, error) {
	configFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	wrapper := struct {
		Password string
		ProgList []common.Process
	}{}
	var programs []common.Process
	var resPtr []*common.Process
	err = json.Unmarshal(configFile, &wrapper)
	if err != nil {
		return nil, err
	}
	programs = wrapper.ProgList
	if wrapper.Password != getPassword() {
		setPassword(wrapper.Password)
		if wrapper.Password != "" {
			setIsUserAuth(false)
		}
	}
	size := len(programs)
	programs = make([]common.Process, size)
	for i := 0; i < size; i++ {
		programs[i] = common.NewProc()
	}
	wrapper.ProgList = programs
	err = json.Unmarshal(configFile, &wrapper)
	if err != nil {
		return nil, err
	}
	programs = wrapper.ProgList
	programs = CreateMultiProcess(programs)
	for i := range programs {
		programs[i].Name = strings.TrimSpace(programs[i].Name)
		if err := programs[i].IsValid(); err == nil {
			resPtr = append(resPtr, &programs[i])
		} else {
			logw.Warning(err.Error())
			fmt.Fprintf(os.Stderr, err.Error())
		}
	}
	return resPtr, nil
}

//LoadFile reads the config file
func LoadFile(filename string) (map[string]*common.Process, error) {
	progs, err := loadFileSlice(filename)
	if err != nil {
		return nil, err
	}
	m := make(map[string]*common.Process, len(progs))
	for _, ptr := range progs {
		m[ptr.Name] = ptr
	}
	return m, nil
}

func CreateMultiProcess(progs []common.Process) []common.Process {
	var newSlice []common.Process
	for _, p := range progs {
		if p.NumProcs > 1 {
			for i := uint(0); i < p.NumProcs; i++ {
				tmp := p
				nb := strconv.Itoa(int(i))
				tmp.Name = p.Name + nb
				tmp.ProcStatus.Name = tmp.Name
				newSlice = append(newSlice, tmp)
			}
		} else {
			p.ProcStatus.Name = p.Name
			newSlice = append(newSlice, p)
		}
	}
	return newSlice
}

func (h *Handler) removeProcs(new map[string]*common.Process) {
	lock.Lock()
	for k := range g_procs {
		if _, exists := new[k]; !exists {
			var useless []common.ProcStatus
			h.StopProc(k, &useless)
			delete(g_procs, k)
		}
	}
	lock.Unlock()
}

func isEnvEqual(old, new []string) bool {
	var oldEnv, newEnv []string
	oldEnv = make([]string, len(old))
	newEnv = make([]string, len(new))
	copy(oldEnv, old)
	copy(newEnv, new)
	if len(oldEnv) != len(newEnv) {
		return false
	}
	sort.Strings(oldEnv)
	sort.Strings(newEnv)
	for i := range oldEnv {
		if oldEnv[i] != newEnv[i] {
			return false
		}
	}
	return true
}

func mustBeRestarted(old, new *common.Process) bool {
	switch {
	case old.Command != new.Command:
		return true
	case old.Outfile != new.Outfile:
		return true
	case old.Errfile != new.Errfile:
		return true
	case old.WorkingDir != new.WorkingDir:
		return true
	case old.Umask != new.Umask:
		return true
	case !isEnvEqual(old.Env, new.Env):
		return true
	default:
		return false
	}
}

func updateProc(old, new *common.Process) {
	old.Lock.Lock()
	defer old.Lock.Unlock()
	old.AutoStart = new.AutoStart
	old.AutoRestart = new.AutoRestart
	old.ExitCodes = new.ExitCodes
	old.StartTime = new.StartTime
	old.StartRetries = new.StartRetries
	old.StopSignal = new.StopSignal
	old.StopTime = new.StopTime
}

func replaceProcess(k string, newConf map[string]*common.Process) {
	lock.Lock()
	defer lock.Unlock()
	g_procs[k] = newConf[k]
}
