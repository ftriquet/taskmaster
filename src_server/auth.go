package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

func generateHash() {
	fmt.Printf("Password: ")
	bytepass, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nUnable to generate hash\n")
		return
	}
	fmt.Printf("\nConfirm password: ")
	bytepass2, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nUnable to generate hash\n")
		return
	}
	if !bytes.Equal(bytepass, bytepass2) {
		fmt.Fprintf(os.Stderr, "\nPassword incorrect\n")
		return
	}
	hash := sha256.New()
	fmt.Printf("\n%x\n", hash.Sum(bytepass))
}

func (h *Handler) HasPassword(i bool, ret *bool) error {
	*ret = (getPassword() != "")
	return nil
}

func (h *Handler) isUserAuth() bool {
	var has bool
	h.HasPassword(false, &has)
	if !has {
		return true
	} else {
		return getIsUserAuth()
	}
}

func setIsUserAuth(is bool) {
	isauthlock.Lock()
	defer isauthlock.Unlock()
	isUserAuth = is
}

func getIsUserAuth() bool {
	isauthlock.RLock()
	defer isauthlock.RUnlock()
	return isUserAuth
}

func setPassword(pass string) {
	passlock.Lock()
	password = pass
	passlock.Unlock()
}
func getPassword() string {
	passlock.RLock()
	defer passlock.RUnlock()
	return password
}

func (h *Handler) Authenticate(pass string, ret *bool) error {
	hash := sha256.New()
	hashedPass := fmt.Sprintf("%x", hash.Sum([]byte(pass)))
	if hashedPass == password {
		*ret = true
		setIsUserAuth(true)
	} else {
		*ret = false
		setIsUserAuth(false)
	}
	return nil
}

func checkPassword(testing bool) bool {
	fmt.Println("Password:")
	var bytepass []byte
	var err error
	if testing {
		bytepass, err = terminal.ReadPassword(int(syscall.Stderr))
	} else {
		bytepass, err = terminal.ReadPassword(int(syscall.Stdin))
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read password\n")
		return false
	}
	hash := sha256.New()
	hashedPass := fmt.Sprintf("%x", hash.Sum(bytepass))
	if hashedPass == password {
		return true
	}
	return false
}
