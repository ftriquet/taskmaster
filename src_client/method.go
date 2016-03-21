package main

import (
	"errors"
	"fmt"
	"net/rpc"
	"os"
	"strconv"
	"taskmaster/common"
)

func GetStatus(client *rpc.Client, commands []string) error {
	var ret []common.ProcStatus
	if commands == nil {
		fmt.Fprint(os.Stderr, "Missing process parameter\n")
		return errors.New("Missing process parameter")
	}
	err := client.Call("Handler.GetStatus", commands, &ret)
	if err != nil {
		return err
	}
	for _, p := range ret {
		fmt.Printf(p.String())
	}
	return nil
}

func StartProc(client *rpc.Client, commands []string) error {
	var ret []common.ProcStatus
	if commands == nil {
		fmt.Fprint(os.Stderr, "Missing process parameter\n")
		return errors.New("Missing process parameter")
	}
	method := common.ServerMethod{MethodName: "StartProc", Params: commands}
	err := client.Call("Handler.AddMethod", method, &ret)
	if err != nil {
		return err
	}
	for _, status := range ret {
		fmt.Printf("Started %s with pid %d\n", status.Name, status.Pid)
	}
	return nil
}

func StopProc(client *rpc.Client, commands []string) error {
	var ret []common.ProcStatus
	if commands == nil {
		fmt.Fprint(os.Stderr, "Missing process parameter\n")
		return errors.New("Missing process parameter")
	}
	method := common.ServerMethod{MethodName: "StopProc", Params: commands}
	err := client.Call("Handler.AddMethod", method, &ret)
	if err != nil {
		return err
	}
	for _, status := range ret {
		fmt.Printf("Stopped %s\n", status.Name)
	}
	return nil
}

func RestartProc(client *rpc.Client, commands []string) error {
	var list []string
	var err error
	if commands == nil {
		fmt.Fprint(os.Stderr, "Missing process parameter\n")
		return errors.New("Missing process parameter")
	}
	if len(commands) == 1 && commands[0] == "all" {
		list, err = LoadProcNames(client)
		if err != nil {
			return err
		}
	} else {
		list = commands
	}
	for _, name := range list {
		err = StopProc(client, []string{name})
		if err == nil {
			err = StartProc(client, []string{name})
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			return err
		}
	}
	return nil
}

func ShutDownServ(client *rpc.Client, commands []string) error {
	var ret string
	method := common.ServerMethod{MethodName: "Shutdown", Params: commands}
	err := client.Call("Handler.AddMethod", method, &ret)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		return err
	}
	fmt.Println(ret)
	return nil
}

func GetLog(client *rpc.Client, commands []string) error {
	var ret []string
	if commands != nil {
		nbLines, err := strconv.Atoi(commands[0])
		if err != nil || nbLines <= 0 {
			return errors.New(fmt.Sprintf("Invalid number: %s\n", commands[0]))
		}
	}
	err := client.Call("Handler.GetLog", commands, &ret)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		return err
	} else {
		for _, log := range ret {
			fmt.Println(log)
		}
	}
	return nil
}
