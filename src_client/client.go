package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/rpc"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/peterh/liner"
)

func LoadProcNames(client *rpc.Client) ([]string, error) {
	var list []string
	err := client.Call("Handler.GetProcList", &list, &list)
	if err != nil {
		return nil, err
	}
	return list, nil
}

type MethodFunc func(*rpc.Client, string) error

var (
	methodMap = map[string]MethodFunc{
		"status":   GetStatus,
		"start":    StartProc,
		"stop":     StopProc,
		"shutdown": ShutDownServ,
		"restart":  RestartProc,
	}
	procList []string
)

func main() {
	port := flag.String("p", "4242", "Port for server connection")
	flag.Parse()

	client, err := rpc.DialHTTP("tcp", "127.0.0.1:"+*port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to the server\n%s\n", err)
		os.Exit(1)
	}
	procList, err = LoadProcNames(client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to the server\n%s\n", err)
		os.Exit(1)
	}

	line := liner.NewLiner()
	line.SetCtrlCAborts(true)
	line.SetCompleter(func(line string) (c []string) {
		comp := []string{"status", "start", "stop", "restart", "shutdown", "log"}
		if len(line) == 0 {
			return comp
		}
		comp = append(comp, procList...)
		f := strings.Fields(line)
		for _, n := range comp {
			if strings.HasPrefix(strings.ToLower(n), strings.ToLower(f[len(f)-1])) {
				f[len(f)-1] = n
				c = append(c, strings.Join(f, " "))
			}
		}
		return
	})
	if f, err := os.Open("./.history"); err == nil {
		line.ReadHistory(f)
		defer f.Close()
	}
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGHUP)
	for {
		l, err := line.Prompt("taskmaster> ")
		if err == io.EOF {
			fmt.Println("Bye")
			break
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading line\n")
			break
		}
		if l != "" {
			line.AppendHistory(l)
			params := strings.Fields(l)
			if params[0] == "quit" {
				client.Close()
				line.Close()
				break
			}
			CallMethod(client, params[0], params[1:])
		}
	}
	if f, err := os.Create("./.history"); err != nil {
		log.Print("Error writing history file: ", err)
	} else {
		line.WriteHistory(f)
		f.Close()
	}
}
