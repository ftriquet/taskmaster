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
			proc.UpdateStatus(common.Starting)
			state <- nil
			//waiting for timestart
			select {
			case <-timeout:
				//process has run enough time
				proc.UpdateStatus(common.Running)
				logw.Info("%s started successfully with pid %d", proc.Name, proc.Pid)
				select {
				case <-processEnd:
				case resp := <-proc.Die:
					//Process will be killed normally (going from backoff)
					resp <- true
					<-processEnd
				}
				if proc.Killed {
					//process killed by stop command
					proc.UpdateStatus(common.Stopped)
					logw.Info("Stopped %s", proc.Name)
					close(state)
					return
				} else {
					//process exited normally
					proc.UpdateStatus(common.Exited)
					if proc.AutoRestart == common.Unexpected && proc.HasCorrectlyExit() {
						close(state)
						return
					}
				}
			case <-processEnd:
				if proc.Killed {
					//process killed by stop command
					proc.UpdateStatus(common.Stopped)
					logw.Info("Stopped %s", proc.Name)
					close(state)
					return
				}
				//process has exited  too quickly
				select {
				case resp := <-proc.Die:
					//Backoff reload
					resp <- false
					proc.UpdateStatus(common.Stopped)
					return
				default:
				}
				proc.UpdateStatus(common.Backoff)
				logw.Warning("Process %s exited too quickly", proc.Name)
			}
			if proc.AutoRestart == common.Never {
				break
			}
		} else {
			select {
			case resp := <-proc.Die:
				//Backoff reload
				proc.UpdateStatus(common.Stopped)
				resp <- false
				return
			default:
			}
			proc.UpdateStatus(common.Backoff)
			state <- errors.New(fmt.Sprintf("Unable to start process %s", proc.Name))
			logw.Warning("Unable to start process %s", proc.Name)
		}
	}
	select {
	case resp := <-proc.Die:
		//Backoff reload
		proc.UpdateStatus(common.Stopped)
		resp <- false
		return
	default:
	}
	close(state)
	proc.UpdateStatus(common.Fatal)
}

func (h *Handler) StartProc(param string, res *[]common.ProcStatus) error {
	var statuses []common.ProcStatus
	proc, exists := g_procs[param]
	if !exists {
		logw.Warning("Process not found: %s", param)
		return errors.New(fmt.Sprintf("Process not found: %s", param))
	}
	if proc.State == common.Starting || proc.State == common.Stopping ||
		proc.State == common.Running {
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
	statuses = []common.ProcStatus{proc.ProcStatus}
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
	if proc.State == common.Backoff {
		response := make(chan bool)
		proc.Die <- response
		v := <-response
		if !v {
			*res = []common.ProcStatus{proc.ProcStatus}
			return nil
		}
	}
	if proc.State != common.Starting && proc.State != common.Running {
		return errors.New(fmt.Sprintf("Process %s is not running", proc.Name))
	}
	timeout := make(chan bool, 1)
	stopped := make(chan bool, 1)
	proc.Killed = true
	syscall.Kill(proc.Cmd.Process.Pid, proc.StopSignal)
	go func() {
		time.Sleep(time.Duration(proc.StopTime) * time.Second)
		timeout <- true
	}()
	go func() {
		for {
			if proc.State == common.Stopped {
				stopped <- true
				return
			}
		}
	}()
	fmt.Println("Waiting for the dead of process")
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
	fmt.Println("Returning for sStopProc")
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
