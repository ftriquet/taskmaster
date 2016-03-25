package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/rpc"
	"os"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"

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
		"start":    StartProc,
		"stop":     StopProc,
		"shutdown": ShutDownServ,
		"restart":  RestartProc,
		"reload":   ReloadConfig,
	}
	procList []string
)

func getProcList() []string {
	return procList
}

func checkAuth(client *rpc.Client) {
	var needPass bool
	err := client.Call("Handler.HasPassword", true, &needPass)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to the server\n%s\n", err)
		client.Close()
		os.Exit(1)
	}
	if !needPass {
		return
	}

	fmt.Printf("Password: ")
	bytePass, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read password\n")
		client.Close()
		os.Exit(1)
	}
	var ok bool
	err = client.Call("Handler.Authenticate", string(bytePass), &ok)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to the server\n%s\n", err)
		client.Close()
		os.Exit(1)
	}
	if !ok {
		fmt.Fprintf(os.Stderr, "Authentication failed\n")
		client.Close()
		os.Exit(1)
	}
}

func getSuffixFrom(word string, list []string) []string {
	var res []string

	for _, cmd := range list {
		if strings.HasPrefix(strings.ToLower(cmd), strings.ToLower(word)) {
			res = append(res, cmd)
		}
	}
	return res
}

func autoComplete(line string) (c []string) {
	comp := []string{"status", "start", "stop", "restart", "shutdown", "log"}
	if len(line) == 0 {
		return comp
	}
	f := strings.Fields(line)
	if len(f) == 0 {
		return comp
	} else if len(f) == 1 && line[len(line)-1] != ' ' {
		return getSuffixFrom(line, comp)
	}
	var completion []string
	if line[len(line)-1] == ' ' {
		completion = getProcList()
		line += "  a"
	} else {
		completion = getSuffixFrom(f[len(f)-1], getProcList())
	}
	for _, proc := range completion {
		l := strings.Fields(line)
		l[len(l)-1] = proc
		c = append(c, strings.Join(l, " "))
	}
	return
}

func main() {
	port := flag.String("p", "4242", "Port for server connection")
	flag.Parse()
	client, err := rpc.DialHTTP("tcp", "127.0.0.1:"+*port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to the server\n%s\n", err)
		os.Exit(1)
	}
	checkAuth(client)
	CallMethod(client, "status", []string{""})
	line := liner.NewLiner()
	line.SetCtrlCAborts(false)
	line.SetCompleter(autoComplete)
	if f, err := os.Open("./.history"); err == nil {
		line.ReadHistory(f)
		defer f.Close()
	}
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
