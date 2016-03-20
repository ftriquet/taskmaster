package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

type Handler struct {
	Programs   []string
	cmds       chan string
	logger     *log.Logger
	Progs      []*Proc
	logfile    string
	configfile string
}

func NewHandler(programs []*Proc, logfile, config string) *Handler {
	fmt.Printf("%+v\n%+v\n%+v\n%d", programs, logfile, config, len(programs))
	h := &Handler{}
	h.Programs = make([]string, len(programs))
	for _, p := range programs {
		h.Programs = append(h.Programs, p.Name)
	}
	h.cmds = make(chan string)
	h.logfile = logfile
	h.configfile = config
	file, err := os.OpenFile(logfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		os.FileMode(0666))
	if err == nil {
		h.logger = log.New(file, "Taskmaster: ", log.Ltime)
	}
	return h
}

func (h *Handler) GetProc(name string) (*Proc, error) {
	for i, _ := range h.Progs {
		if h.Progs[i].Name == name {
			return h.Progs[i], nil
		}
	}
	return nil, errors.New(fmt.Sprintf("%s: program not found", name))
}

func (h *Handler) DisplayLog() {
	h.displayFile(h.logfile)
}

func (h *Handler) DisplayConfig() {
	h.displayFile(h.configfile)
}

func (h *Handler) displayFile(filename string) {
	cmd := exec.Command("/bin/cat", filename)
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func (h *Handler) CorrectProcExit(name string) bool {
	p, err := h.GetProc(name)
	if err != nil {
		return false
	}
	return p.HasCorrectlyExit()
}

func (h *Handler) PrintStatus(procName string) error {
	if procName == "all" {
		for _, n := range h.Programs {
			h.PrintStatus(n)
		}
	} else {
		p, err := h.GetProc(procName)
		if err == nil {
			fmt.Print(p.StrStatus())
		} else {
			return err
		}
	}
	return nil
}

func (h *Handler) RestartProcess(procName string) error {
	p, err := h.GetProc(procName)
	if err != nil {
		return err
	}
	if p.Status != Running {
		return errors.New("Program not started")
	}
	h.StopProcess(procName)
	h.StartProcess(procName)
	return nil
}

func (h *Handler) StopProcess(name string) error {
	p, err := h.GetProc(name)
	if err != nil {
		return err
	}
	if p.Status != Running {
		return nil
	}
	p.Killed = true
	p.Stop()
	return nil
}

func (h *Handler) StartProcess(name string) error {
	p, err := h.GetProc(name)
	if err != nil {
		return err
	}
	if p.Status == Running || p.Status == Starting {
		return errors.New(fmt.Sprintf("Process already started: %s", p.Name))
	}
	tries := 0
	//g
	for tries <= p.StartRetries || p.Autorestart == Always {
		//Lecture
		//coindition qui break
		timeout := make(chan bool)
		started := make(chan bool)
		p.Init()
		p.Status = Starting
		go func() {
			time.Sleep(time.Second * time.Duration(p.StartTime))
			timeout <- true
		}()
		go p.Launch(started)
		select {
		case <-timeout:
			h.logger.Printf("Process %s has not started after %d seconds\n",
				p.Name, p.StartTime)
		case val := <-started:
			if val {
				h.logger.Printf("Started %s successfully with pid %d",
					p.Name, p.Cmd.Process.Pid)
				<-p.Finish
				if p.HasCorrectlyExit() {
					if p.Killed {
						h.logger.Printf("Process %s has been killed manually\n", p.Name)
					} else {
						h.logger.Printf("Process %s has exited with exit code %d (Expected)\n", p.Name, p.GetExitCode())
					}
					return nil
				} else {
					h.logger.Printf("Process %s has exited with exit code %d (Unexpected)\n", p.Name, p.GetExitCode())
					if p.Autorestart == Never {
						return nil
					}
				}
			} else {
				h.logger.Printf("Unable to start process %s\n", p.Name)
				p.Status = Starting
			}
		}
		tries++
	}
	p.Status = Fatal
	h.logger.Printf("Process %s's status changin to FATAL (too many unexpected exits)\n", p.Name)
	return nil
}

func (h *Handler) Try(name string, fn func(string) error) {
	err := fn(name)
	if err != nil {
		h.logger.Printf("Error: %s\n", err)
	}
}
