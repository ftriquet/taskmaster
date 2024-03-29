package main

import (
	"crypto/sha256"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"
	"taskmaster/common"
)

type BasicAuth struct {
	Login    string
	Password string
}

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
	lock.RLock()
	t.Execute(w, g_procs)
	lock.RUnlock()
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
		a := generateBasicAuth()
		if a != nil {
			a.BasicAuthHandler(w, r, h)
		} else {
			handleRequest(w, r, h)
		}
	}
}

func generateBasicAuth() *BasicAuth {
	if getPassword() != "" {
		return NewBasicAuth()
	}
	return nil
}

func handleRequest(w http.ResponseWriter, r *http.Request, h *Handler) {
	if r.URL.Path == "/" {
		indexHandler(w, r)
		return
	}
	actionHandler(w, r, h)
}

func NewBasicAuth() *BasicAuth {
	return &BasicAuth{Login: "taskmaster", Password: getPassword()}
}

func (a *BasicAuth) Authenticate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", `Basic realm="taskmaster"`)
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}

func (a *BasicAuth) ValidAuth(r *http.Request) bool {
	username, passUser, ok := r.BasicAuth()
	if !ok {
		return false
	}
	hash := sha256.New()
	hashedPass := fmt.Sprintf("%x", hash.Sum([]byte(passUser)))
	return username == a.Login && password == hashedPass
}

func (a *BasicAuth) BasicAuthHandler(w http.ResponseWriter, r *http.Request, h *Handler) {
	if !a.ValidAuth(r) {
		a.Authenticate(w, r)
	} else {
		handleRequest(w, r, h)
	}
}
