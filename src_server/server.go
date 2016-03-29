package main

import (
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"sync"
	"taskmaster/common"
	"taskmaster/log"
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

func getProc(k string) (res *common.Process, exists bool) {
	lock.RLock()
	defer lock.RUnlock()
	res, exists = g_procs[k]
	return
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
