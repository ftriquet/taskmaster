package common

import (
	"os"
	"os/exec"
	"syscall"
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

//Wrapper for a server method call
type ServerMethod struct {
	MethodName string
	Params     []string
	Method     func([]string, *interface{}) error
	Result     *interface{}
}