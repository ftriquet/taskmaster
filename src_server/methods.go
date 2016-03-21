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
}

func (h *Handler) handleProcess(proc *common.Process, state chan error) {
	var tries = 0
	processEnd := make(chan bool)
	started := make(chan bool)
	for tries <= proc.StartRetries || proc.AutoRestart == common.Always {
		tries++
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
				logw.Info("%s started successfully with pid %d", proc.Name, proc.Pid)
				UpdateStatus(proc, common.Running)
				<-processEnd
				if proc.Killed {
					//process killed by stop command
					logw.Info("Stopped %s", proc.Name)
					UpdateStatus(proc, common.Stopped)
					close(state)
					return
				} else {
					//process exited normally
					UpdateStatus(proc, common.Exited)
					if proc.AutoRestart == common.Always && proc.HasCorrectlyExit() {
						close(state)
						return
					}
				}
			case <-processEnd:
				if proc.Killed {
					//process killed by stop command
					logw.Info("Stopped %s", proc.Name)
					UpdateStatus(proc, common.Stopped)
					close(state)
					return
				}
				//process has exited  too quickly
				logw.Warning("Process %s exited too quickly", proc.Name)
				UpdateStatus(proc, common.Backoff)
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
	proc.Killed = true
	syscall.Kill(proc.Pid, proc.StopSignal)
	time.AfterFunc(time.Second*time.Duration(proc.StopTime), func() {
		if proc.State != common.Stopped {
			syscall.Kill(proc.Pid, syscall.SIGKILL)
		}
	})
	*res = []common.ProcStatus{proc.ProcStatus}
	return nil
}
