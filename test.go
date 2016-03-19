package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/peterh/liner"
)

type jsonContainer struct {
	Programs []Proc
}

func getNextLine(reader *bufio.Reader) string {
	var line string

	line, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}
	line = strings.Replace(line, "\n", "", 1)
	return line
}

func LoadFile(filename string) ([]*Proc, error) {
	configFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	container := jsonContainer{}
	var resPtr []*Proc
	err = json.Unmarshal(configFile, &container)
	if err != nil {
		return nil, err
	}
	//fmt.Printf("CONF:\n%+v\n", container)
	size := len(container.Programs)
	container.Programs = make([]Proc, size)
	for i := 0; i < size; i++ {
		container.Programs[i] = NewProc()
	}
	err = json.Unmarshal(configFile, &container)
	//fmt.Printf("CONF:\n%+v\n", container)
	if err != nil {
		return nil, err
	}
	container.Programs = CreateMultiProcess(container.Programs)
	resPtr = make([]*Proc, len(container.Programs))
	for i, _ := range container.Programs {
		resPtr[i] = &container.Programs[i]
	}
	return resPtr, nil
}

func CreateMultiProcess(progs []Proc) []Proc {
	newSlice := make([]Proc, 0)
	for _, p := range progs {
		if p.NumProcs > 1 {
			for i := uint(0); i < p.NumProcs; i++ {
				tmp := p
				nb := strconv.Itoa(int(i))
				tmp.Name = p.Name + nb
				newSlice = append(newSlice, tmp)
			}
		} else {
			newSlice = append(newSlice, p)
		}
	}
	return newSlice
}

func main() {
	configFile := flag.String("c", "./config.json", "Config-file name")
	logfile := flag.String("l", "./taskmaster_logs", "Taskmaster's log file")
	flag.Parse()
	programs, err := LoadFile(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	handler := NewHandler(programs, *logfile, *configFile)
	handler.Progs = programs
	handler.logger.Printf("Starting taskmaster with pid %d\n", os.Getpid())
	cmds := make(chan string)
	go handler.HandleProgs(cmds)
	comp := []string{"status", "start", "stop", "restart", "shutdown", "log"}
	for _, n := range programs {
		comp = append(comp, n.Name)
	}
	line := liner.NewLiner()
	line.SetCtrlCAborts(true)
	line.SetCompleter(func(line string) (c []string) {
		f := strings.Fields(line)
		for _, n := range comp {
			if strings.HasPrefix(strings.ToLower(n), strings.ToLower(f[len(f)-1])) {
				f[len(f)-1] = n
				c = append(c, strings.Join(f, " "))
			}
		}
		return
	})
	if f, err := os.Create("./.history"); err == nil {
		line.ReadHistory(f)
		f.Close()
	}
	for {
		l, err := line.Prompt("> ")
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			os.Exit(1)
		}
		if l != "" {
			cmds <- l
			<-cmds
		}
		if l == "shutdown" {
			if f, err := os.Create("./.history"); err == nil {
				line.WriteHistory(f)
				f.Close()
			}
			line.Close()
			os.Exit(0)
		}
	}
}

func (h *Handler) HandleProgs(cmds chan string) {
	for _, p := range h.Progs {
		if p.Autostart {
			go h.Try(p.Name, h.StartProcess)
		}
	}
	for {
		cmd := <-cmds
		spl := strings.Fields(cmd)
		if len(spl) > 0 {
			if spl[0] == "log" {
				h.DisplayLog()
			} else if spl[0] == "config" {
				h.DisplayConfig()
			} else if spl[0] == "status" {
				for _, s := range spl[1:] {
					h.Try(s, h.PrintStatus)
				}
				if len(spl) == 1 {
					h.Try("all", h.PrintStatus)
				}
			} else if spl[0] == "shutdown" {
				for _, s := range h.Programs {
					h.Try(s, h.StopProcess)
				}
			} else if spl[0] == "stop" {
				for _, s := range spl[1:] {
					if s == "all" {
						for _, n := range h.Programs {
							go h.Try(n, h.StopProcess)
						}
					} else {
						go h.Try(s, h.StopProcess)
					}
				}
			} else if spl[0] == "start" {
				if len(spl) > 1 {
					for _, s := range spl[1:] {
						if s == "all" {
							for _, n := range h.Programs {
								go h.Try(n, h.StartProcess)
							}
						} else {
							go h.Try(s, h.StartProcess)
						}
					}
				}
			} else if spl[0] == "restart" {
				if len(spl) > 1 {
					for _, s := range spl[1:] {
						if s == "all" {
							for _, n := range h.Programs {
								go h.Try(n, h.RestartProcess)
							}
						} else {
							go h.Try(s, h.RestartProcess)
						}
					}
				}
			} else {
				h.logger.Printf("Error: %s, unknown command\n", spl[0])
			}
		}
		cmds <- "LOL"
	}
}
