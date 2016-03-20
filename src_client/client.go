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

type MethodFunc func(*rpc.Client, []string) error

var (
	methodMap = map[string]MethodFunc{
		"status":   GetStatus,
		"start":    StartProc,
		"stop":     StopProc,
		"shutdown": ShutDownServ,
		"restart":  RestartProc,
		"log":      GetLog,
	}
)

func main() {
	port := flag.String("p", "4242", "Port for server connection")
	flag.Parse()
	comp := []string{"status", "start", "stop", "restart", "shutdown", "log"}

	client, err := rpc.DialHTTP("tcp", "127.0.0.1:"+*port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to the server\n%s", err)
		os.Exit(1)
	}
	procList, err := LoadProcNames(client)
	fmt.Printf("%+v\n", procList)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to the server\n%s", err)
		os.Exit(1)
	}

	comp = append(comp, procList...)
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

	for {
		line, _ := line.Prompt("taskmaster> ")
		if line != "" {
			params := strings.Fields(line)
			f := methodMap[params[0]]
			err := f(client, params[1:])
			if err != nil {
				fmt.Println(err.Error())
			}
		}
	}
}
