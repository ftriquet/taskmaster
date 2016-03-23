package main

import (
	"errors"
	"fmt"
	"syscall"
	"taskmaster/common"
	"taskmaster/log"
	"time"
)

func UpdateStatus(p *common.Process, state string) {
	lock.Lock()
	p.State = state
	lock.Unlock()
	logw.Info("Process %s entered status %s", p.Name, state)
}

func (h *Handler) handleProcess(proc *common.Process, state chan error) {
	var tries = 0
	processEnd := make(chan bool)
	started := make(chan bool)
	for tries <= proc.StartRetries || proc.AutoRestart == common.Always {
		tries++
		proc.Killed = false
		timeout := make(chan bool, 1)
		go func() {
			time.Sleep(time.Second * time.Duration(proc.StartTime))
			timeout <- true
		}()
		go proc.Start(started, processEnd)
		ok := <-started
		//process has started normally
		if ok {
			UpdateStatus(proc, common.Starting)
			state <- nil
			//waiting for timestart
			select {
			case <-timeout:
				//process has run enough time
				UpdateStatus(proc, common.Running)
				logw.Info("%s started successfully with pid %d", proc.Name, proc.Pid)
				<-processEnd
				if proc.Killed {
					//process killed by stop command
					UpdateStatus(proc, common.Stopped)
					logw.Info("Stopped %s", proc.Name)
					close(state)
					return
				} else {
					//process exited normally
					UpdateStatus(proc, common.Exited)
					if proc.AutoRestart == common.Unexpected && proc.HasCorrectlyExit() {
						close(state)
						return
					}
				}
			case <-processEnd:
				if proc.Killed {
					//process killed by stop command
					UpdateStatus(proc, common.Stopped)
					logw.Info("Stopped %s", proc.Name)
					close(state)
					return
				}
				//process has exited  too quickly
				UpdateStatus(proc, common.Backoff)
				logw.Warning("Process %s exited too quickly", proc.Name)
			}
			if proc.AutoRestart == common.Never {
				break
			}
		} else {
			UpdateStatus(proc, common.Backoff)
			state <- errors.New(fmt.Sprintf("Unable to start process %s", proc.Name))
			logw.Warning("Unable to start process %s", proc.Name)
		}
	}
	close(state)
	UpdateStatus(proc, common.Fatal)
}

func (h *Handler) StartProc(params []string, res *[]common.ProcStatus) error {
	var statuses []common.ProcStatus
	procName := params[0]
	proc, exists := g_procs[procName]
	if !exists {
		logw.Warning("Process not found: %s", procName)
		return errors.New(fmt.Sprintf("Process not found: %s", procName))
	}
	if proc.State == common.Starting || proc.State == common.Stopping ||
		proc.State == common.Running {
		return errors.New(fmt.Sprintf("Process already running: %s", procName))
	}
	state := make(chan error)
	go h.handleProcess(proc, state)
	err := <-state
	go func() {
		for {
			err, open := <-state
			if open {
				if err != nil {
					logw.Warning(err.Error())
				}
			} else {
				break
			}
		}
	}()
	statuses = []common.ProcStatus{proc.ProcStatus}
	if err != nil {
		return err
	}
	*res = statuses
	fmt.Printf("Send: %+v\n", statuses)
	return nil
}

func (h *Handler) StopProc(params []string, res *[]common.ProcStatus) error {
	procName := params[0]
	proc, exists := g_procs[procName]
	if !exists {
		logw.Warning("Process not found: %s", procName)
		return errors.New(fmt.Sprintf("Process not found: %s", procName))
	}
	if proc.State != common.Starting && proc.State != common.Running {
		return errors.New(fmt.Sprintf("Process %s is not running", proc.Name))
	}
	timeout := make(chan bool, 2)
	stopped := make(chan bool, 1)
	proc.Killed = true
	syscall.Kill(proc.Cmd.Process.Pid, proc.StopSignal)
	go func() {
		time.Sleep(time.Duration(proc.StopTime) * time.Second)
		timeout <- true
		timeout <- true
	}()
	go func() {
		for {
			if proc.State == common.Stopped {
				stopped <- true
				return
			}
			select {
			case <-timeout:
				return
			default:
			}
		}
	}()
	select {
	case <-timeout:
		if proc.State != common.Stopped {
			syscall.Kill(proc.Cmd.Process.Pid, syscall.SIGKILL)
			logw.Info("Process %s was killed by SIGKILL dans sa face", proc.Name)
		}
	case <-stopped:
		logw.Info("Process %s was killed normally", proc.Name)
	}
	*res = []common.ProcStatus{proc.ProcStatus}
	return nil
}
