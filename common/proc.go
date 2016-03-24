package common

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
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
	p.StopSignal = syscall.SIGINT
	p.ExitCodes = []int{0, 2}
	p.Lock = &sync.RWMutex{}
	p.Die = make(chan chan bool)
	return p
}

func (p *Process) GetProcStatus() ProcStatus {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.ProcStatus
}

func (p *Process) SetPid(pid int) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.Pid = pid
}

func (p *Process) GetPid() int {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.Pid
}

func (p *Process) SetRuntime(start time.Time) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.Runtime = start
}

func (p *Process) GetName() string {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.Name
}

func (p *Process) SetName(param string) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.Name = param
	p.ProcStatus.Name = param
}

func (p *Process) GetNumProcs() uint {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.NumProcs
}

func (p *Process) SetNumProcs(param uint) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.NumProcs = param
}

func (p *Process) GetCommand() string {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.Command
}

func (p *Process) SetCommand(param string) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.Command = param
}

func (p *Process) GetUmask() uint32 {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.Umask
}

func (p *Process) SetUmask(param uint32) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.Umask = param
}

func (p *Process) GetOutfile() string {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.Outfile
}

func (p *Process) SetOutfile(param string) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.Outfile = param
}
func (p *Process) GetErrfile() string {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.Errfile
}
func (p *Process) SetErrfile(param string) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.Errfile = param
}
func (p *Process) GetStdout() *os.File {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.Stdout
}
func (p *Process) SetStdout(param *os.File) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.Stdout = param
}
func (p *Process) GetStderr() *os.File {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.Stderr
}
func (p *Process) SetStderr(param *os.File) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.Stderr = param
}
func (p *Process) GetWorkingDir() string {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.WorkingDir
}
func (p *Process) SetWorkingDir(param string) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.WorkingDir = param
}
func (p *Process) GetCmd() *exec.Cmd {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.Cmd
}
func (p *Process) SetCmd(param *exec.Cmd) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.Cmd = param
}
func (p *Process) GetEnv() []string {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.Env
}
func (p *Process) SetEnv(param []string) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.Env = param
}
func (p *Process) GetAutoStart() bool {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.AutoStart
}
func (p *Process) SetAutoStart(param bool) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.AutoStart = param
}
func (p *Process) GetAutoRestart() string {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.AutoRestart
}
func (p *Process) SetAutoRestart(param string) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.AutoRestart = param
}
func (p *Process) GetExitCodes() []int {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.ExitCodes
}
func (p *Process) SetExitCodes(param []int) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.ExitCodes = param
}
func (p *Process) GetStartTime() uint {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.StartTime
}
func (p *Process) SetStartTime(param uint) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.StartTime = param
}
func (p *Process) GetStartRetries() int {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.StartRetries
}
func (p *Process) SetStartRetries(param int) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.StartRetries = param
}
func (p *Process) GetStopSignal() syscall.Signal {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.StopSignal
}
func (p *Process) SetStopSignal(param syscall.Signal) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.StopSignal = param
}
func (p *Process) GetStopTime() uint {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.StopTime
}
func (p *Process) SetStopTime(param uint) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.StopTime = param
}
func (p *Process) GetKilled() bool {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.Killed
}
func (p *Process) SetKilled(param bool) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.Killed = param
}

func (p *Process) SetStatus(state string) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.State = state
	logw.Info("Process %s entered status %s", p.Name, state)
}

func (p *Process) IsValid() bool {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	return p.Name != "" && p.Command != ""
}

func (p *Process) InitStderr() error {
	file, err := os.Create(p.Errfile)
	if err != nil {
		return err
	}
	p.Stderr = file
	p.Cmd.Stderr = file
	return nil
}

func (p *Process) InitStdout() error {
	file, err := os.Create(p.Outfile)
	if err != nil {
		return err
	}
	p.Stdout = file
	p.Cmd.Stdout = file
	return nil
}

func (p *Process) Init() error {
	spl := strings.Fields(p.GetCommand())
	p.Cmd = exec.Command(spl[0], spl[1:]...)
	wd := p.GetWorkingDir()
	if wd != "" {
		p.Cmd.Dir = wd
	}
	env := p.GetEnv()
	if env != nil {
		p.Cmd.Env = env
	}
	if p.Stderr == nil && p.GetErrfile() != "" {
		p.CloseLogs()
		if err := p.InitStderr(); err != nil {
			return err
		}
	}
	if p.Stdout == nil && p.GetOutfile() != "" {
		p.CloseLogs()
		if err := p.InitStdout(); err != nil {
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
	codes := p.GetExitCodes()
	for _, e := range codes {
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

func (p *ProcStatus) String() string {
	if p.State == Running || p.State == Starting {
		return fmt.Sprintf("%s: %s [%d] %.5s\n", p.Name, p.State, p.Pid, time.Since(p.Runtime).String())
	} else {
		return fmt.Sprintf("%s: %s\n", p.Name, p.State)
	}
}

func (p *Process) Start(started, processEnd chan bool) {
	err := p.Init()
	if err != nil {
		logw.Error(err.Error())
		return
	}
	if p.State == Starting || p.State == Running {
		logw.Error("Process %s already started", p.Name)
		return
	}
	syscall.Umask(int(p.GetUmask()))
	err = p.Cmd.Start()
	if err != nil {
		started <- false
		return
	}
	p.SetRuntime(time.Now())
	p.SetPid(p.Cmd.Process.Pid)
	started <- true
	p.Cmd.Wait()
	p.SetPid(0)
	processEnd <- true
}
