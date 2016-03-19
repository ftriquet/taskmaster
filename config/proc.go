package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	//"sync"
	"syscall"
	"time"
)

const (
	Stopped    = "STOPPED"
	Running    = "RUNNING"
	Starting   = "STARTING"
	Fatal      = "FATAL"
	Never      = "Never"
	Always     = "Always"
	Unexpected = "Unexpected"
)

var defaultUmask uint32 = 0022

type Proc struct {
	Name         string
	Command      string
	NumProcs     uint
	Umask        uint32
	Outfile      string
	Errfile      string
	Stdout       *os.File
	Stderr       *os.File
	WorkingDir   string
	Cmd          *exec.Cmd
	Env          []string
	Status       string
	Autostart    bool
	Autorestart  string
	ExitCodes    []int
	StartTime    uint
	StartRetries int
	StopSignal   syscall.Signal
	StopTime     uint
	Killed       bool
	Finish       chan bool
	KillChan     chan bool
	Time         time.Time
}

func NewProc() Proc {
	p := Proc{}
	p.Status = Stopped
	p.Name = ""
	p.Command = ""
	p.NumProcs = 1
	p.Umask = defaultUmask
	p.StartTime = 10
	p.StopTime = 10
	p.SetStopSignal("INT")
	p.Autorestart = Never
	p.Autostart = false
	p.StartRetries = 0
	p.Killed = false
	return p
}

func (p *Proc) SetStopSignal(sig string) {
	switch sig {
	case "ABRT":
		p.StopSignal = syscall.SIGABRT
	case "ALRM":
		p.StopSignal = syscall.SIGALRM
	case "BUS":
		p.StopSignal = syscall.SIGBUS
	case "CHLD":
		p.StopSignal = syscall.SIGCHLD
	case "CONT":
		p.StopSignal = syscall.SIGCONT
	case "FPE":
		p.StopSignal = syscall.SIGFPE
	case "HUP":
		p.StopSignal = syscall.SIGHUP
	case "ILL":
		p.StopSignal = syscall.SIGILL
	case "INT":
		p.StopSignal = syscall.SIGINT
	case "IO":
		p.StopSignal = syscall.SIGIO
	case "IOT":
		p.StopSignal = syscall.SIGIOT
	case "KILL":
		p.StopSignal = syscall.SIGKILL
	case "PIPE":
		p.StopSignal = syscall.SIGPIPE
	case "PROF":
		p.StopSignal = syscall.SIGPROF
	case "QUIT":
		p.StopSignal = syscall.SIGQUIT
	case "SEGV":
		p.StopSignal = syscall.SIGSEGV
	case "STOP":
		p.StopSignal = syscall.SIGSTOP
	case "SYS":
		p.StopSignal = syscall.SIGSYS
	case "TERM":
		p.StopSignal = syscall.SIGTERM
	case "TRAP":
		p.StopSignal = syscall.SIGTRAP
	case "TSTP":
		p.StopSignal = syscall.SIGTSTP
	case "TTIN":
		p.StopSignal = syscall.SIGTTIN
	case "TTOU":
		p.StopSignal = syscall.SIGTTOU
	case "URG":
		p.StopSignal = syscall.SIGURG
	case "USR1":
		p.StopSignal = syscall.SIGUSR1
	case "USR2":
		p.StopSignal = syscall.SIGUSR2
	case "WINCH":
		p.StopSignal = syscall.SIGWINCH
	case "XCPU":
		p.StopSignal = syscall.SIGXCPU
	case "XFSZ":
		p.StopSignal = syscall.SIGXFSZ
	}
}

func (p *Proc) SetUmask(u uint32) {
	p.Umask = u
}

func (p *Proc) SetStderr() error {
	var file *os.File
	var err error

	file, err = os.Create(p.Errfile)
	if err == nil {
		p.Stderr = file
		p.Cmd.Stderr = file
	} else {
		return errors.New("Unable to open stderr log file")
	}
	return nil
}

func (p *Proc) SetStdout() error {
	var file *os.File
	var err error

	file, err = os.Create(p.Outfile)
	if err == nil {
		p.Stdout = file
		p.Cmd.Stdout = file
	} else {
		return errors.New("Unable to open stdout log file")
	}
	return nil
}

func (p *Proc) Init() error {
	spl := strings.Fields(p.Command)
	p.Cmd = exec.Command(spl[0], spl[1:]...)
	if p.WorkingDir != "" {
		p.Cmd.Dir = p.WorkingDir
	}
	if p.Env != nil {
		p.Cmd.Env = p.Env
	}
	if p.ExitCodes == nil {
		p.ExitCodes = []int{0}
	}
	p.CloseLogs()
	if p.Stderr == nil && p.Errfile != "" {
		p.SetStderr()
	}
	if p.Stdout == nil && p.Outfile != "" {
		p.SetStdout()
	}
	p.Finish = make(chan bool)
	p.KillChan = make(chan bool, 1)
	return nil
}

func (p *Proc) Launch(started chan bool) error {
	var err error
	st := p.Status
	if st == Running {
		return errors.New("Program already started")
	}
	syscall.Umask(int(p.Umask))
	err = p.Cmd.Start()
	if err != nil {
		started <- false
		return err
	}
	p.Time = time.Now()
	p.Status = Running
	started <- true
	p.Cmd.Wait()
	p.Status = Stopped
	p.CloseLogs()
	p.Finish <- true
	return nil
}

func (p *Proc) CloseLogs() {
	if p.Stderr != nil {
		p.Stderr.Close()
		p.Stderr = nil
	}
	if p.Stdout != nil {
		p.Stdout.Close()
		p.Stdout = nil
	}
}

func (p *Proc) HasCorrectlyExit() bool {
	if p.Killed {
		return true
	}
	exitCode := p.GetExitCode()
	for _, e := range p.ExitCodes {
		if e == exitCode {
			return true
		}
	}
	return false
}

func (p *Proc) GetExitCode() int {
	exitCode := p.Cmd.ProcessState.Sys().(syscall.WaitStatus)
	if exitCode > 128 {
		exitCode = (exitCode >> 8) & 255
	}
	return int(exitCode)
}

func (p *Proc) HasExit() bool {
	if p.Status == Running || p.Status == Starting {
		return false
	}
	return p.Cmd.ProcessState.Exited()
}

func (p *Proc) Stop() {
	st := p.Status
	if st == Stopped || p.Cmd == nil {
		return
	}
	timeout := make(chan bool, 1)
	stopped := make(chan bool, 1)
	syscall.Kill(p.Cmd.Process.Pid, p.StopSignal)
	if p.Cmd.ProcessState == nil || !p.Cmd.ProcessState.Exited() {
		go func() {
			time.Sleep(time.Second * time.Duration(p.StopTime))
			timeout <- true
		}()
		go func() {
			for p.Cmd.ProcessState == nil {
			}
			stopped <- true
		}()
		select {
		case <-timeout:
			if p.Cmd.ProcessState == nil || p.Cmd.ProcessState.Exited() == false {
				syscall.Kill(p.Cmd.Process.Pid, syscall.SIGKILL)
			}
		case <-stopped:
			return
		}
	}
}

func (p *Proc) String() string {
	var res string

	res = `
	Name: %s
	Command: %s
	Numprocs: %d
	Umask: %d
	Outfile: %s
	Errfile: %s
	WorkingDir: %s
	Env: %v
	Status: %s
	Autostart: %t
	Autorestart: %s
	ExitCodes: %v
	StartTime: %d
	StartRetries: %d
	StopSignal: %d
	StopTime: %d
	Killed: %t
	`
	return fmt.Sprintf(res, p.Name, p.Command, p.NumProcs, p.Umask, p.Outfile,
		p.Errfile, p.WorkingDir, p.Env, p.Status, p.Autostart, p.Autorestart,
		p.ExitCodes, p.StartTime, p.StartRetries, p.StopSignal, p.StopTime, p.Killed)
}

func (p *Proc) StrStatus() string {
	var st string
	var status string = p.Status
	if status == Running || status == Starting {
		st = "%s: %s [%d] (%s) %s\n"
		return fmt.Sprintf(st, p.Name, status, p.Cmd.Process.Pid,
			time.Since(p.Time).String(), p.Command)
	} else {
		st = "%s : %s: %s\n"
		return fmt.Sprintf(st, p.Name, status, p.Command)
	}
	return ""
}
