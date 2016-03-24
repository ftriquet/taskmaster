package main

import (
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"
	"taskmaster/common"
)

func indexHandler(w http.ResponseWriter, req *http.Request) {
	content, err := ioutil.ReadFile("./common/templates/index.html")
	if err != nil {
		return
	}
	t := template.New("template")
	t, err = t.Parse(string(content))
	if err != nil {
		return
	}
	t.Execute(w, g_procs)
}

func actionHandler(w http.ResponseWriter, r *http.Request, h *Handler) {
	var method common.ServerMethod
	var res []common.ProcStatus
	split := strings.Split(r.URL.Path, "/")
	if split[0] == "" {
		split = split[1:]
	}
	if len(split) == 2 {
		method.MethodName = split[0]
		method.Param = split[1]
		h.AddMethod(method, &res)
	} else if len(split) == 1 && split[0] == "reload" {
		h.ReloadConfig("", &res)
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func generateRenderer(h *Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleRequest(w, r, h)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request, h *Handler) {
	if r.URL.Path == "/" {
		indexHandler(w, r)
		return
	}
	actionHandler(w, r, h)
}
