package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"syscall"
	"taskmaster/common"
	"taskmaster/log"
	"time"
)

func (h *Handler) handleProcess(proc *common.Process, state chan error) {
	var tries = uint(0)
	processEnd := make(chan bool)
	started := make(chan bool)
	for tries <= proc.GetStartRetries() || proc.GetAutoRestart() == common.Always {
		tries++
		proc.SetKilled(false)
		timeout := make(chan bool, 1)
		go func() {
			time.Sleep(time.Second * time.Duration(proc.GetStartTime()))
			timeout <- true
		}()
		go proc.Start(started, processEnd)
		ok := <-started
		//process has started normally
		if ok {
			proc.SetStatus(common.Starting)
			state <- nil
			//waiting for timestart
			select {
			case <-timeout:
				//process has run enough time
				proc.SetStatus(common.Running)
				logw.Info("%s started successfully with pid %d", proc.Name, proc.GetPid())
				select {
				case <-processEnd:
				case resp := <-proc.Die:
					//Process will be killed normally (going from backoff)
					resp <- true
					<-processEnd
				}
				if proc.GetKilled() {
					//process killed by stop command
					proc.SetStatus(common.Stopped)
					logw.Info("Stopped %s", proc.Name)
					close(state)
					return
				} else {
					//process exited normally
					proc.SetStatus(common.Exited)
					if proc.GetAutoRestart() == common.Unexpected && proc.HasCorrectlyExit() {
						close(state)
						return
					}
				}
			case <-processEnd:
				if proc.GetKilled() {
					//process killed by stop command
					proc.SetStatus(common.Stopped)
					logw.Info("Stopped %s", proc.Name)
					close(state)
					return
				}
				//process has exited  too quickly
				select {
				case resp := <-proc.Die:
					//Backoff reload
					resp <- false
					proc.SetStatus(common.Stopped)
					return
				default:
				}
				proc.SetStatus(common.Backoff)
				logw.Warning("Process %s exited too quickly", proc.Name)
			}
			if proc.GetAutoRestart() == common.Never {
				break
			}
		} else {
			select {
			case resp := <-proc.Die:
				//Backoff reload
				proc.SetStatus(common.Stopped)
				resp <- false
				return
			default:
			}
			proc.SetStatus(common.Backoff)
			state <- errors.New(fmt.Sprintf("Unable to start process %s", proc.Name))
			logw.Warning("Unable to start process %s", proc.Name)
		}
	}
	select {
	case resp := <-proc.Die:
		//Backoff reload
		proc.SetStatus(common.Stopped)
		resp <- false
		return
	default:
	}
	close(state)
	proc.CloseLogs()
	proc.SetStatus(common.Fatal)
}

func (h *Handler) StartProc(param string, res *[]common.ProcStatus) error {
	var statuses []common.ProcStatus
	proc, exists := g_procs[param]
	if !exists {
		logw.Warning("Process not found: %s", param)
		return errors.New(fmt.Sprintf("Process not found: %s", param))
	}
	status := proc.GetProcStatus()
	if status.State == common.Starting || status.State == common.Stopping ||
		status.State == common.Running {
		return errors.New(fmt.Sprintf("Process already running: %s", param))
	}
	state := make(chan error)
	go h.handleProcess(proc, state)
	err := <-state
	go func() {
		for {
			_, open := <-state
			if !open {
				break
			}
		}
	}()
	statuses = []common.ProcStatus{proc.GetProcStatus()}
	if err != nil {
		return err
	}
	*res = statuses
	return nil
}

func (h *Handler) StopProc(param string, res *[]common.ProcStatus) error {
	proc, exists := g_procs[param]
	if !exists {
		logw.Warning("Process not found: %s", param)
		return errors.New(fmt.Sprintf("Process not found: %s", param))
	}
	status := proc.GetProcStatus()
	if status.State == common.Backoff {
		response := make(chan bool)
		proc.Die <- response
		v := <-response
		if !v {
			*res = []common.ProcStatus{status}
			return nil
		}
	}
	statu := proc.GetProcStatus().State
	if statu != common.Starting && statu != common.Running {
		return errors.New(fmt.Sprintf("Process %s is not running", proc.Name))
	}
	timeout := make(chan bool, 1)
	stopped := make(chan bool, 1)
	proc.SetKilled(true)
	syscall.Kill(proc.Cmd.Process.Pid, proc.StopSignal)
	go func() {
		time.Sleep(time.Duration(proc.GetStopTime()) * time.Second)
		timeout <- true
	}()
	go func() {
		for {
			if proc.GetProcStatus().State == common.Stopped {
				stopped <- true
				return
			}
		}
	}()
	select {
	case <-timeout:
		if proc.GetProcStatus().State != common.Stopped {
			syscall.Kill(proc.Cmd.Process.Pid, syscall.SIGKILL)
			logw.Info("Process %s was killed by SIGKILL dans sa face", proc.Name)
		}
	case <-stopped:
		logw.Info("Process %s was killed normally", proc.Name)
	}
	proc.CloseLogs()
	*res = []common.ProcStatus{proc.GetProcStatus()}
	return nil
}

func (h *Handler) GetLog(nbLines int, res *[]string) error {
	file, err := ioutil.ReadFile(h.logfile)
	if err != nil {
		return err
	}
	split := bytes.Split(file, []byte{'\n'})
	size := len(split) - 1
	if nbLines > 0 && nbLines < size {
		size = nbLines
	}
	lines := make([]string, size)
	j := size - 1
	for i := len(split) - 2; j >= 0; i-- {
		lines[j] = string(split[i])
		j--
	}
	*res = lines
	return nil
}

func (h *Handler) ReloadConfig(param string, res *[]common.ProcStatus) error {

	newConf, err := LoadFile(h.configFile)
	if err != nil {
		logw.Error("Unable to laod config file: %s", h.configFile)
		return err
	}
	h.Pause <- true
	h.removeProcs(newConf)
	h.updateWhatMustBeUpdated(newConf)
	h.handleAutoStart()
	h.Continue <- true
	*res = []common.ProcStatus{}
	return nil
}

func (h *Handler) RestartProc(param string, res *[]common.ProcStatus) error {
	err := h.StopProc(param, res)
	if err == nil {
		err = h.StartProc(param, res)
	}
	return err
}

func (h *Handler) Shutdown(param string, res *[]common.ProcStatus) error {
	*res = []common.ProcStatus{{State: "Server has shutddown"}}
	lock.Lock()
	for k, proc := range g_procs {
		s := proc.GetProcStatus().State
		if s == common.Running || s == common.Starting || s == common.Backoff {
			var u []common.ProcStatus
			h.StopProc(k, &u)
		}
	}
	return nil
}

func (h *Handler) handleAutoStart() {
	for k, v := range g_procs {
		var useless []common.ProcStatus
		if v.AutoStart {
			h.StartProc(k, &useless)
		}
	}
}
