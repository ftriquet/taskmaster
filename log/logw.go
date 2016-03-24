package logw

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
)

var (
	g_info    *log.Logger
	g_warning *log.Logger
	g_alert   *log.Logger
	g_err     *log.Logger
	g_rlog    *Rotlog
	g_rotlock *sync.Mutex
	g_silent  bool
)

func Init() {
	g_info = log.New(os.Stdout, "INFO    ", log.Ldate|log.Ltime)
	g_warning = log.New(os.Stdout, "WARNING ", log.Ldate|log.Ltime)
	g_alert = log.New(os.Stdout, "ALERT   ", log.Ldate|log.Ltime)
	g_err = log.New(os.Stderr, "ERROR   ", log.Ldate|log.Ltime)
	g_rlog = nil
	g_silent = false
	g_rotlock = &sync.Mutex{}
}

// for tests, or just if you like silence, emptyness, darkness, oblivion
func InitSilent() {
	g_info = log.New(ioutil.Discard, "INFO    ", log.Ldate|log.Ltime)
	g_warning = log.New(ioutil.Discard, "WARNING ", log.Ldate|log.Ltime)
	g_alert = log.New(ioutil.Discard, "ALERT   ", log.Ldate|log.Ltime)
	g_err = log.New(ioutil.Discard, "ERROR   ", log.Ldate|log.Ltime)
	g_silent = true
	g_rotlock = &sync.Mutex{}
}

func InitRotatingLog(name string, rotate_every int, nbfiles int) error {
	var err error
	g_rlog, err = InitRotlog(name, uint64(rotate_every), nbfiles)
	if err != nil {
		g_rlog = nil
		return err
	}
	return nil
}

func writeAndSwap(log *log.Logger, msg string, stderr bool) {
	g_rotlock.Lock()
	defer g_rotlock.Unlock()
	log.SetOutput(g_rlog)
	log.Printf(msg)
	if g_silent {
		log.SetOutput(ioutil.Discard)
	} else if stderr {
		log.SetOutput(os.Stderr)
	} else {
		log.SetOutput(os.Stdout)
	}
}

func Info(s string, values ...interface{}) {
	msg := fmt.Sprintf(s, values...)
	g_info.Printf(msg)
	if g_rlog != nil {
		writeAndSwap(g_info, msg, false)
	}
}

func Warning(s string, values ...interface{}) {
	msg := fmt.Sprintf(s, values...)
	g_warning.Printf(msg)
	if g_rlog != nil {
		writeAndSwap(g_warning, msg, false)
	}
}

func Alert(s string, values ...interface{}) {
	msg := fmt.Sprintf(s, values...)
	g_alert.Printf(msg)
	if g_rlog != nil {
		writeAndSwap(g_alert, msg, false)
	}
}

func Error(s string, values ...interface{}) {
	msg := fmt.Sprintf(s, values...)
	g_err.Printf(msg)
	if g_rlog != nil {
		writeAndSwap(g_err, msg, true)
	}
}
