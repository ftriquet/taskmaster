package main

import (
	"bytes"
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
	"strings"
	"sync"
	"syscall"
	"taskmaster/common"
	"taskmaster/log"

	"golang.org/x/crypto/ssh/terminal"
)

type MethodFunc func(string, *[]common.ProcStatus) error

var (
	g_procs    map[string]*common.Process
	lock       = new(sync.RWMutex)
	passlock   = new(sync.RWMutex)
	isauthlock = new(sync.RWMutex)
	password   string
	isUserAuth bool
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

func getProc(k string) (res *common.Process, exists bool) {
	lock.RLock()
	defer lock.RUnlock()
	res, exists = g_procs[k]
	return
}

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

func sliceContains(s []string, name string) bool {
	for _, n := range s {
		if n == name {
			return true
		}
	}
	return false
}

func (h *Handler) GetStatus(params []string, result *[]common.ProcStatus) error {
	if !h.isUserAuth() {
		return errors.New("You are not authenticated. Restart your client")
	}
	res := []common.ProcStatus{}
	var procList []string
	lock.RLock()
	for k, proc := range g_procs {
		procList = append(procList, k)
		if params[0] == "" || sliceContains(params, k) {
			res = append(res, proc.GetProcStatus())
		}
	}
	lock.RUnlock()
	res = append(res, common.ProcStatus{Name: strings.Join(procList, " ")})
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
	return ret
}

func (h *Handler) init(config, log string) {
	h.methodMap = map[string]MethodFunc{
		"StartProc":   h.StartProc,
		"StopProc":    h.StopProc,
		"RestartProc": h.RestartProc,
		"Reload":      h.ReloadConfig,
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
	fmt.Printf("Password: ")
	bytepass, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nUnable to generate hash\n")
		return
	}
	fmt.Printf("\nConfirm password: ")
	bytepass2, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nUnable to generate hash\n")
		return
	}
	if !bytes.Equal(bytepass, bytepass2) {
		fmt.Fprintf(os.Stderr, "\nPassword incorrect\n")
		return
	}
	hash := sha256.New()
	fmt.Printf("\n%x\n", hash.Sum(bytepass))
}

func (h *Handler) HasPassword(i bool, ret *bool) error {
	*ret = ((getPassword() != "") && (getIsUserAuth() == false))
	return nil
}

func (h *Handler) isUserAuth() bool {
	var has bool
	h.HasPassword(false, &has)
	if !has {
		return true
	} else {
		return getIsUserAuth()
	}
}

func setIsUserAuth(is bool) {
	isauthlock.Lock()
	defer isauthlock.Unlock()
	isUserAuth = is
}

func getIsUserAuth() bool {
	isauthlock.RLock()
	defer isauthlock.RUnlock()
	return isUserAuth
}

func setPassword(pass string) {
	passlock.Lock()
	password = pass
	passlock.Unlock()
}
func getPassword() string {
	passlock.RLock()
	defer passlock.RUnlock()
	return password
}

func (h *Handler) Authenticate(pass string, ret *bool) error {
	hash := sha256.New()
	hashedPass := fmt.Sprintf("%x", hash.Sum([]byte(pass)))
	if hashedPass == password {
		*ret = true
		setIsUserAuth(true)
	} else {
		*ret = false
		setIsUserAuth(false)
	}
	return nil
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
	port := flag.Uint("p", 4242, "Server port")
	configFile := flag.String("c", "./config.json", "Config-file name")
	logfile := flag.String("l", "./taskmaster_logs", "Taskmaster's log file")
	logsize := flag.Uint("s", 65535, "Max size of a log file")
	lognb := flag.Uint("n", 8, "Max number of log files")
	genPassword := flag.Bool("h", false, "Generate password hash")
	httpFlag := flag.Bool("b", true, "Active http server")
	flag.Parse()

	if *genPassword {
		generateHash()
		return
	}
	h := new(Handler)
	h.init(*configFile, *logfile)

	logw.InitSilent()
	err := logw.InitRotatingLog(h.logfile, int(*logsize), int(*lognb))
	if err != nil {
		log.Fatal("Failed to open log file")
	}
	g_procs, err = LoadFile(h.configFile)
	if err != nil {
		log.Fatal("Unable to load config file")
	}
	err = rpc.Register(h)
	if err != nil {
		log.Fatal(err)
	}
	rpc.HandleHTTP()
	listener, err := net.Listen("tcp", ":"+strconv.FormatUint(uint64(*port), 10))
	if err != nil {
		log.Fatal(err)
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
	listenSIGHUP(*configFile, h)
	if *httpFlag {
		http.HandleFunc("/", generateRenderer(h))
	}
	log.Fatal(http.Serve(listener, nil))
}
