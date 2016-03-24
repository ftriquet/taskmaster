package main

import (
	"fmt"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"taskmaster/common"
)

func GetStatus(client *rpc.Client, args []string) error {
	var ret []common.ProcStatus
	err := client.Call("Handler.GetStatus", args, &ret)
	if err != nil {
		return err
	}
	if len(ret) > 0 {
		procList = strings.Fields(ret[len(ret)-1].Name)
		fmt.Printf("%v\n", procList)
		ret = ret[:len(ret)-1]
	}
	for _, p := range ret {
		fmt.Printf(p.String())
	}
	return nil
}

func GetLog(client *rpc.Client, params []string) error {
	var ret []string
	var param int
	var err error
	if len(params) > 0 {
		param, err = strconv.Atoi(params[0])
		if err != nil {
			return err
		}
	} else {
		param = 0
	}
	err = client.Call("Handler.GetLog", param, &ret)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		return err
	} else {
		for _, log := range ret {
			fmt.Println(log)
		}
	}
	return nil
}

func CallMethod(client *rpc.Client, command string, args []string) error {
	var argList []string
	if command == "log" {
		return GetLog(client, args)
	}
	if command == "status" {
		if len(args) == 0 || args[0] == "all" {
			return GetStatus(client, []string{""})
		}
		return GetStatus(client, args)
	}
	if len(args) == 0 || args[0] == "all" {
		argList = procList
	} else {
		argList = args
	}
	if command == "shutdown" {
		argList = []string{""}
	}
	f, exists := methodMap[command]
	if exists {
		for _, proc := range argList {
			err := f(client, proc)
			if err != nil {
				fmt.Println(err.Error())
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
	}
	return nil
}

func StartProc(client *rpc.Client, procName string) error {
	var ret []common.ProcStatus
	method := common.ServerMethod{MethodName: "StartProc", Param: procName}
	err := client.Call("Handler.AddMethod", method, &ret)
	if err != nil {
		return err
	}
	for _, status := range ret {
		fmt.Printf("Started %s with pid %d\n", status.Name, status.Pid)
	}
	return nil
}

func StopProc(client *rpc.Client, procName string) error {
	var ret []common.ProcStatus
	method := common.ServerMethod{MethodName: "StopProc", Param: procName}
	err := client.Call("Handler.AddMethod", method, &ret)
	if err != nil {
		return err
	}
	for _, status := range ret {
		fmt.Printf("Stopped %s\n", status.Name)
	}
	return nil
}

func RestartProc(client *rpc.Client, procName string) error {
	var err error
	err = StopProc(client, procName)
	if err == nil {
		err = StartProc(client, procName)
	}
	if err != nil {
		//fmt.Fprintf(os.Stderr, err.Error())
		return err
	}
	return nil
}

func ShutDownServ(client *rpc.Client, commands string) error {
	var ret []common.ProcStatus
	//Stop all process
	tmp, err := LoadProcNames(client)
	if err == nil {
		for _, name := range tmp {
			StopProc(client, name)
		}
	}
	//Shutdown server
	method := common.ServerMethod{MethodName: "Shutdown", Param: commands}
	err = client.Call("Handler.AddMethod", method, &ret)
	if err != nil {
		return err
	}
	if ret != nil && len(ret) == 1 {
		fmt.Println(ret[0].State)
	}
	return nil

}
