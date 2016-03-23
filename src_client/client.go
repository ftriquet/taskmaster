package main

import (
	"flag"
	"fmt"
	"net/rpc"
	"os"
	"strings"

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

	for {
		l, _ := line.Prompt("taskmaster> ")
		if l != "" {
			params := strings.Fields(l)
			if params[0] == "quit" {
				client.Close()
				line.Close()
				break
			}
			CallMethod(client, params[0], params[1:])
		}
	}
}
