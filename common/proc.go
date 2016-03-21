package common

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"taskmaster/log"
	"time"
)

func NewProc() Process {
	p := Process{}
	p.ProcStatus = ProcStatus{State: Stopped}
	p.AutoRestart = DflAutoRestart
	p.AutoStart = DflAutoStart
	p.StartTime = DflStartTime
	p.StartRetries = DflStartRetries
	p.StopTime = DflStopTime
	p.Umask = DflUmask
	p.NumProcs = DflNumProcs
	p.ExitCodes = []int{0, 2}
	return p
}

func (p *Process) IsValid() bool {
	return p.Name != "" && p.Command != ""
}

func (p *Process) SetStderr() error {
	file, err := os.Create(p.Errfile)
	if err != nil {
		return err
	}
	p.Stderr = file
	p.Cmd.Stderr = file
	return nil
}

func (p *Process) SetStdout() error {
	file, err := os.Create(p.Outfile)
	if err != nil {
		return err
	}
	p.Stdout = file
	p.Cmd.Stdout = file
	return nil
}

func (p *Process) Init() error {
	spl := strings.Fields(p.Command)
	p.Cmd = exec.Command(spl[0], spl[1:]...)
	if p.WorkingDir != "" {
		p.Cmd.Dir = p.WorkingDir
	}
	if p.Env != nil {
		p.Cmd.Env = p.Env
	}
	if p.Stderr == nil && p.Errfile != "" {
		p.CloseLogs()
		if err := p.SetStderr(); err != nil {
			return err
		}
	}
	if p.Stdout == nil && p.Outfile != "" {
		p.CloseLogs()
		if err := p.SetStdout(); err != nil {
			return err
		}
	}
	return nil
}

func (p *Process) CloseLogs() {
	if p.Stderr != nil {
		p.Stderr.Close()
		p.Stderr = nil
	}
	if p.Stdout != nil {
		p.Stdout.Close()
		p.Stdout = nil
	}
}

func (p *Process) HasCorrectlyExit() bool {
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

func (p *Process) GetExitCode() int {
	exitCode := p.Cmd.ProcessState.Sys().(syscall.WaitStatus)
	if exitCode > 128 {
		exitCode = (exitCode >> 8) & 255
	} else {
		exitCode += 128
	}
	return int(exitCode)
}

func (p *Process) HasExit() bool {
	if p.ProcStatus.State == Running || p.ProcStatus.State == Starting {
		return false
	}
	return p.Cmd.ProcessState.Exited()
}

func (p *Process) StrStatus() string {
	var st string
	var status string = p.ProcStatus.State
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

func (p *Process) Status() ProcStatus {
	return p.ProcStatus
}

func (p *ProcStatus) String() string {
	return fmt.Sprintf("%s: %s [%d] %s\n", p.Name, p.State, p.Pid, p.Runtime.String())
}

func (p *Process) Start(started, processEnd chan bool) {
	err := p.Init()
	if err != nil {
		logw.Error(err.Error())
		return
	}
	if p.State != Starting {
		logw.Error("Process %s already started", p.Name)
		return
	}
	syscall.Umask(int(p.Umask))
	err = p.Cmd.Start()
	if err != nil {
		started <- false
		return
	}
	started <- true
	p.Cmd.Wait()
	processEnd <- true
}
