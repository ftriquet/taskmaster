package common

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

const (
	Stopped                = "STOPPED"
	Running                = "RUNNING"
	Starting               = "STARTING"
	Fatal                  = "FATAL"
	Exited                 = "EXITED"
	Stopping               = "STOPPING"
	Backoff                = "BACKOFF"
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

type ReadRequest struct {
	Field    string
	Response chan interface{}
}

type WriteRequest struct {
	FieldName string
	Field     interface{}
	Response  chan bool
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
	Killed       bool
	Lock         *sync.RWMutex
	Die          chan chan bool
	ReadChan     chan ReadRequest
	WriteChan    chan WriteRequest
	StopDealer   chan bool
}

func (p *Process) StartDataDealer() {
	p.StopDealer = make(chan bool)
	go func() {
		for {
			select {
			case <-p.StopDealer:
				close(p.ReadChan)
				close(p.WriteChan)
				close(p.StopDealer)
				return
			case readReq := <-p.ReadChan:
				resp := p.getFieldByName(readReq.Field)
				readReq.Response <- resp
			case writeReq := <-p.WriteChan:
				if err := p.setFieldByName(writeReq.FieldName, writeReq.Field); err == nil {
					writeReq.Response <- true
				} else {
					writeReq.Response <- false
				}
			}
		}
	}()
}

//ProcStatus s
type ProcStatus struct {
	Name    string
	Pid     int
	State   string
	Runtime time.Time
}

//Wrapper for a server method call
type ServerMethod struct {
	MethodName string
	Param      string
	Method     func(string, *[]ProcStatus) error
	Result     *[]ProcStatus
}

func (p *Process) setFieldByName(name string, value interface{}) error {

	var ok bool
	switch name {
	case "ProcStatus":
		if p.ProcStatus, ok = value.(ProcStatus); !ok {
			return errors.New("Invalid type in write request")
		}
	case "Name":
		if p.Name, ok = value.(string); !ok {
			return errors.New("Invalid type in write request")
		}
	case "NumProcs":
		if p.NumProcs, ok = value.(uint); !ok {
			return errors.New("Invalid type in write request")
		}
	case "Command":
		if p.Command, ok = value.(string); !ok {
			return errors.New("Invalid type in write request")
		}
	case "Umask":
		if p.Umask, ok = value.(uint32); !ok {
			return errors.New("Invalid type in write request")
		}
	case "Outfile":
		if p.Outfile, ok = value.(string); !ok {
			return errors.New("Invalid type in write request")
		}
	case "Errfile":
		if p.Errfile, ok = value.(string); !ok {
			return errors.New("Invalid type in write request")
		}
	case "Stdout":
		if p.Stdout, ok = value.(*os.File); !ok {
			return errors.New("Invalid type in write request")
		}
	case "Stderr":
		if p.Stderr, ok = value.(*os.File); !ok {
			return errors.New("Invalid type in write request")
		}
	case "WorkingDir":
		if p.WorkingDir, ok = value.(string); !ok {
			return errors.New("Invalid type in write request")
		}
	case "Cmd":
		if p.Cmd, ok = value.(*exec.Cmd); !ok {
			return errors.New("Invalid type in write request")
		}
	case "Env":
		if p.Env, ok = value.([]string); !ok {
			return errors.New("Invalid type in write request")
		}
	case "AutoStart":
		if p.AutoStart, ok = value.(bool); !ok {
			return errors.New("Invalid type in write request")
		}
	case "AutoRestart":
		if p.AutoRestart, ok = value.(string); !ok {
			return errors.New("Invalid type in write request")
		}
	case "ExitCodes":
		if p.ExitCodes, ok = value.([]int); !ok {
			return errors.New("Invalid type in write request")
		}
	case "StartTime":
		if p.StartTime, ok = value.(uint); !ok {
			return errors.New("Invalid type in write request")
		}
	case "StartRetries":
		if p.StartRetries, ok = value.(int); !ok {
			return errors.New("Invalid type in write request")
		}
	case "StopSignal":
		if p.StopSignal, ok = value.(syscall.Signal); !ok {
			return errors.New("Invalid type in write request")
		}
	case "StopTime":
		if p.StopTime, ok = value.(uint); !ok {
			return errors.New("Invalid type in write request")
		}
	case "Killed":
		if p.Killed, ok = value.(bool); !ok {
			return errors.New("Invalid type in write request")
		}
	case "Lock":
		if p.Lock, ok = value.(*sync.RWMutex); !ok {
			return errors.New("Invalid type in write request")
		}
	case "Die":
		if p.Die, ok = value.(chan chan bool); !ok {
			return errors.New("Invalid type in write request")
		}
	}
	return errors.New("Invalid write request")
}

func (p *Process) getFieldByName(name string) interface{} {
	switch name {
	case "ProcStatus":
		return p.ProcStatus
	case "Name":
		return p.Name
	case "NumProcs":
		return p.NumProcs
	case "Command":
		return p.Command
	case "Umask":
		return p.Umask
	case "Outfile":
		return p.Outfile
	case "Errfile":
		return p.Errfile
	case "Stdout":
		return p.Stdout
	case "Stderr":
		return p.Stderr
	case "WorkingDir":
		return p.WorkingDir
	case "Cmd":
		return p.Cmd
	case "Env":
		return p.Env
	case "AutoStart":
		return p.AutoStart
	case "AutoRestart":
		return p.AutoRestart
	case "ExitCodes":
		return p.ExitCodes
	case "StartTime":
		return p.StartTime
	case "StartRetries":
		return p.StartRetries
	case "StopSignal":
		return p.StopSignal
	case "StopTime":
		return p.StopTime
	case "Killed":
		return p.Killed
	case "Lock":
		return p.Lock
	}
	return nil
}
