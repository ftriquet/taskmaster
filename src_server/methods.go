package main

import (
	"errors"
	"fmt"
	"taskmaster/common"
	"taskmaster/log"
	"time"
)

func (h *Handler) handleProcess(proc *common.Process, state chan error) {
	var tries = 0
	for tries <= proc.StartRetries || proc.AutoRestart == common.Always {
		tries++
		lock.Lock()
		proc.State = common.Starting
		lock.Unlock()
		timeout := make(chan bool, 1)
		processEnd := make(chan bool, 1)
		started := make(chan bool, 1)
		go func() {
			time.Sleep(time.Second * time.Duration(proc.StartTime))
			timeout <- true
		}()
		go proc.Start(started, processEnd)
		ok := <-started
		if ok {
			select {
			case <-timeout:
				//succes
				logw.Info("%s started successfully with pid %d", proc.Name, proc.Pid)
				lock.Lock()
				proc.State = common.Running
				lock.Unlock()
				<-processEnd
				if proc.Killed {
					logw.Info("Stopped %s", proc.Name)
					lock.Lock()
					proc.State = common.Stopped
					lock.Unlock()
					return
				} else {
					lock.Lock()
					proc.State = common.Exited
					lock.Unlock()
					if proc.AutoRestart == common.Always && proc.HasCorrectlyExit() {
						return
					}
				}
			case <-processEnd:
				logw.Warning("Process %s exited too quickly", proc.Name)
				//fail
			}
			if proc.AutoRestart == common.Never {
				break
			}
		} else {
			logw.Warning("Unable to start process %s", proc.Name)
		}
	}
	lock.Lock()
	proc.State = common.Fatal
	lock.Unlock()
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
	ok := make(chan error)
	go h.handleProcess(proc, ok)
	err := <-ok
	if err != nil {
		return err
	}
	*res = statuses
	return nil
}
