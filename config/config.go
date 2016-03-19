package config

import (
	"os"
	"os/exec"
	"syscall"
	"time"
)

//Process a
type Process struct {
	ProcStatus
	NumProcs     uint
	umask        uint32
	Outfile      string
	Errfile      string
	Stdout       *os.File
	Stderr       *os.File
	WorkingDir   string
	Cmd          *exec.Cmd
	Env          []string
	Autostart    bool
	Autorestart  string
	ExitCodes    []int
	StartTime    uint
	StartRetries int
	StopSignal   syscall.Signal
	StopTime     uint
	Time         time.Time
}

//ProcStatus s
type ProcStatus struct {
	Name    string
	Pid     int
	State   string
	Runtime time.Duration
}
