package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"sync"
	"syscall"
	"taskmaster/common"
	"taskmaster/log"

	"golang.org/x/crypto/ssh/terminal"
)

type MethodFunc func(string, *[]common.ProcStatus) error

var (
	g_procs  map[string]*common.Process
	lock     = new(sync.RWMutex)
	password string
)

type Handler struct {
	Actions             chan common.ServerMethod
	Response            chan error
	methodMap           map[string]MethodFunc
	configFile, logfile string
	Pause, Continue     chan bool
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
	copy(oldEnv, old)
	copy(newEnv, new)
	if oldEnv == nil || newEnv == nil {
		return oldEnv == nil && newEnv == nil
	}
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

func (h *Handler) updateWhatMustBeUpdated(newConf map[string]*common.Process) {
	var toRestart []string
	var useless []common.ProcStatus
	for k := range newConf {
		if proc, exists := g_procs[k]; exists {
			if proc.State == common.Running || proc.State == common.Starting {
				if mustBeRestarted(g_procs[k], newConf[k]) {
					toRestart = append(toRestart, k)
					h.StopProc(k, &useless)
					g_procs[k] = newConf[k]
				} else {
					updateProc(g_procs[k], newConf[k])
				}
			} else if proc.State == common.Backoff {
				toRestart = append(toRestart, k)
				h.StopProc(k, &useless)
				g_procs[k] = newConf[k]
			} else {
				newConf[k].State = g_procs[k].GetProcStatus().State
				g_procs[k] = newConf[k]
			}
		} else {
			g_procs[k] = newConf[k]
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
			//newConf, err := LoadFile(filename)
			//if err != nil {
			//	logw.Error("Unable to laod config file: %s", filename)
			//	continue
			//}
			//h.Pause <- true
			//h.removeProcs(newConf)
			//h.updateWhatMustBeUpdated(newConf)
			//h.handleAutoStart()
			//h.Continue <- true
			h.ReloadConfig("lol", &[]common.ProcStatus{})
		}
	}()

}

func (h *Handler) GetProcList(p *[]string, res *[]string) error {
	var list []string

	for k, _ := range g_procs {
		list = append(list, k)
	}
	*res = list
	return nil
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
	password = wrapper.Password
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
		if programs[i].IsValid() {
			resPtr = append(resPtr, &programs[i])
		} else {
			//logw.Warning("Process %d has an empty name or empty command")
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

func (h *Handler) GetStatus(param string, result *[]common.ProcStatus) error {
	res := []common.ProcStatus{}
	lock.RLock()
	p, exists := g_procs[param]
	lock.RUnlock()
	if exists {
		res = append(res, p.GetProcStatus())
	} else {
		logw.Warning("%s: Process not found", param)
		return fmt.Errorf("Process not found: %s", param)
	}
	*result = res
	return nil
}

func (h *Handler) AddMethod(action common.ServerMethod, res *[]common.ProcStatus) error {
	action.Method = h.methodMap[action.MethodName]
	if action.Method == nil {
		return errors.New("No such method")
	}
	action.Result = res
	h.Actions <- action
	if action.MethodName == "Shutdown" {
		defer close(h.Actions)
	}
	ret := <-h.Response
	fmt.Printf("Returning form AddMethod %s\n", action.MethodName)
	return ret
}

func (h *Handler) init(config, log string) {
	h.methodMap = map[string]MethodFunc{
		"StartProc":   h.StartProc,
		"StopProc":    h.StopProc,
		"RestartProc": h.RestartProc,
		"Shutdown":    h.Shutdown,
	}
	h.logfile = log
	h.configFile = config
	h.Actions = make(chan common.ServerMethod)
	h.Response = make(chan error)
	h.Continue = make(chan bool)
	h.Pause = make(chan bool)
}

func generateHash() {
	fmt.Println("Password:")
	bytepass, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to generate hash\n")
		return
	}
	hash := sha256.New()
	fmt.Printf("%x\n", hash.Sum(bytepass))
}

func checkPassword(testing bool) bool {
	fmt.Println("Password:")
	var bytepass []byte
	var err error
	if testing {
		bytepass, err = terminal.ReadPassword(int(syscall.Stderr))
	} else {
		bytepass, err = terminal.ReadPassword(int(syscall.Stdin))
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read password\n")
		return false
	}
	hash := sha256.New()
	hashedPass := fmt.Sprintf("%x", hash.Sum(bytepass))
	if hashedPass == password {
		return true
	}
	return false
}

func main() {
	configFile := flag.String("c", "./config.json", "Config-file name")
	logfile := flag.String("l", "./taskmaster_logs", "Taskmaster's log file")
	genPassword := flag.Bool("p", false, "Generate password hash")
	flag.Parse()

	if *genPassword {
		generateHash()
		return
	}
	h := new(Handler)
	h.init(*configFile, *logfile)

	logw.InitSilent()
	err := logw.InitRotatingLog(h.logfile, 65535, 8)
	if err != nil {
		panic(err)
	}
	g_procs, err = LoadFile(h.configFile)
	if err != nil {
		fmt.Println("Unable to load config file")
		os.Exit(1)
	}
	listenSIGHUP(*configFile, h)
	err = rpc.Register(h)
	if err != nil {
		panic(err)
	}
	rpc.HandleHTTP()
	listener, err := net.Listen("tcp", ":4242")
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			select {
			case action, open := <-h.Actions: //(Servermethod)
				if open {
					err := action.Method(action.Param, action.Result)
					h.Response <- err
				} else {
					listener.Close()
					os.Exit(0)
				}
			case <-h.Pause:
				<-h.Continue
			}
		}
	}()
	h.handleAutoStart()
	http.HandleFunc("/", generateRenderer(h))
	log.Fatal(http.Serve(listener, nil))
}
