package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"taskmaster/common"
	"taskmaster/log"

	"golang.org/x/crypto/ssh/terminal"
)

//type MethodFunc func([]string, *interface{}) error
type MethodFunc func([]string, *[]common.ProcStatus) error

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
}

func listenSIGHUP(filename string) {
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGHUP)
	var err error
	go func() {
		<-sig
		lock.Lock()
		tmp := g_procs
		tmp = tmp
		lock.Unlock()
		if err = LoadFile(filename); err != nil {
			logw.Error(err.Error())
		}
		//update process avec tmp et g_trucs
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
func LoadFile(filename string) error {
	progs, err := loadFileSlice(filename)
	if err != nil {
		return err
	}
	m := make(map[string]*common.Process, len(progs))
	for _, ptr := range progs {
		m[ptr.Name] = ptr
	}
	lock.Lock()
	g_procs = m
	lock.Unlock()
	return nil
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

func (h *Handler) GetStatus(commands []string, result *[]common.ProcStatus) error {
	res := []common.ProcStatus{}
	if commands == nil || (len(commands) == 1 && commands[0] == "all") {
		for _, proc := range g_procs {
			res = append(res, proc.ProcStatus)
		}
	} else {
		for _, proc := range commands {
			p, exists := g_procs[proc]
			if exists {
				res = append(res, p.ProcStatus)
			} else {
				logw.Warning("%s: Process not found", proc)
			}
		}
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
	fmt.Println("Action sent, waiting for response")
	return <-h.Response
}

func (h *Handler) init(config, log string) {
	h.methodMap = map[string]MethodFunc{
		"StartProc": h.StartProc,
		//"StopProc":  h.StopProc,
	}
	h.logfile = log
	h.configFile = config
	h.Actions = make(chan common.ServerMethod)
	h.Response = make(chan error)
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

	go func() {
		for {
			//fmt.Println("Waiting for action")
			action := <-h.Actions //(Servermethod)
			//fmt.Printf("Action received: %s\n", action.MethodName)
			err := action.Method(action.Params, action.Result)
			//fmt.Println("DONE")
			h.Response <- err
			//fmt.Println("Response sent")
		}
	}()
	logw.InitSilent()
	err := logw.InitRotatingLog(h.logfile, 65535, 8)
	if err != nil {
		panic(err)
	}
	err = LoadFile(h.configFile)
	if err != nil {
		panic(err)
	}

	err = rpc.Register(h)
	if err != nil {
		panic(err)
	}
	rpc.HandleHTTP()
	listener, err := net.Listen("tcp", ":4242")
	if err != nil {
		panic(err)
	}
	err = http.Serve(listener, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println("LOOP")
}
