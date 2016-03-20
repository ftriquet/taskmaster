package main

import (
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"taskmaster/log"
	"time"
)

const (
	Stopped                = "STOPPED"
	Running                = "RUNNING"
	Starting               = "STARTING"
	Fatal                  = "FATAL"
	Never                  = "Never"
	Always                 = "Always"
	Unexpected             = "Unexpected"
	DflUmask        uint32 = 022
	DflStopSignal          = syscall.SIGTERM
	DflAutoRestart         = Unexpected
	DflAutoStart           = false
	DflStartRetries        = 3
	DflStopTime     uint   = 10
	DflStartTime    uint   = 10
	DflNumProcs     uint   = 1
)

var (
	g_procs map[string]*Process
	lock    = new(sync.RWMutex)
)

type Handler struct {
}

type Process struct {
	ProcStatus
	Name         string
	NumProcs     uint
	Command      string
	Umask        uint32
	Outfile      string
	Errfile      string
	Stdout       *os.File
	Stderr       *os.File
	WorkingDir   string
	Cmd          *exec.Cmd
	Env          []string
	AutoStart    bool
	AutoRestart  string
	ExitCodes    []int
	StartTime    uint
	StartRetries int
	StopSignal   syscall.Signal
	StopTime     uint
	Time         time.Time
	Killed       bool
}

//ProcStatus s
type ProcStatus struct {
	Name    string
	Pid     int
	State   string
	Runtime time.Duration
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
