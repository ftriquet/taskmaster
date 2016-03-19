package logw

import (
	"errors"
	"os"
	"path"
	"strconv"
	"strings"
)

type Rotlog struct {
	filename     string
	rot_every    uint64
	nbFiles      int
	current      *os.File
	current_size uint64
}

func InitRotlog(name string, rotate_every uint64, nbfiles int) (*Rotlog, error) {
	if name == "" || rotate_every == 0 || nbfiles <= 0 {
		return nil, errors.New("tried to initiate rotating log with a zero value")
	}
	//if no "/", let's assume the dir is the current directory
	if strings.IndexByte(name, '/') != -1 {
		path := path.Dir(name)
		_, err := os.Stat(path)
		if err != nil && os.IsNotExist(err) {
			//try to create the directory
			err = os.MkdirAll(path, 0777)
			if err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
	}
	rl := new(Rotlog)
	rl.filename = name
	rl.rot_every = rotate_every
	rl.nbFiles = nbfiles
	err := rl.initFirstFile()
	return rl, err
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

//mieux
func (r *Rotlog) fileNumberExists(num int) bool {
	if num == 0 {
		return fileExists(r.filename)
	} else {
		return fileExists(r.filename + "." + strconv.FormatUint(uint64(num), 10))
	}
}

func (r *Rotlog) createFileNb(num int) (*os.File, error) {
	if num == 0 {
		return os.Create(r.filename)
	} else {
		return os.Create(r.filename + "." + strconv.FormatUint(uint64(num), 10))
	}
}

//write in file. Metjhod to satisfy the io.Writer interface
func (r *Rotlog) Write(p []byte) (n int, err error) {
	if r.shallRotate() {
		r.rotate()
	}
	n, err = r.current.Write(p)
	if err != nil {
		return n, err
	}
	if p[len(p)-1] != '\n' {
		_, err = r.current.Write([]byte{10})
		if err == nil {
			n++
		}
	}
	r.current_size += uint64(n)
	return n, err
}

func (r *Rotlog) shallRotate() bool {
	return r.current_size >= r.rot_every
}

func (r *Rotlog) initFirstFile() error {
	var err error
	if r.current != nil {
		r.current.Close()
		r.current = nil
	}
	r.current, err = os.OpenFile(r.filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	stat, err := r.current.Stat()
	if err != nil {
		return err
	}
	r.current_size = uint64(stat.Size())
	return nil
}

func (r *Rotlog) rotateFileNb(n int) error {
	var err error
	if n == r.nbFiles-1 {
		err = os.Remove(r.filename + "." + strconv.FormatUint(uint64(n), 10))
	} else if n == 0 {
		err = os.Rename(r.filename, r.filename+"."+strconv.FormatUint(uint64(n+1), 10))
	} else {
		err = os.Rename(r.filename+"."+strconv.FormatUint(uint64(n), 10), r.filename+"."+strconv.FormatUint(uint64(n+1), 10))
	}
	return err
}

//woohooo
func (r *Rotlog) rotate() error {
	var err error
	//close the current file
	if r.current != nil {
		err = r.current.Close()
		if err != nil {
			return err
		}
	}
	for i := r.nbFiles - 1; i >= 0; i-- {
		if r.fileNumberExists(i) {
			err = r.rotateFileNb(i)
			if err != nil {
				return err
			}
		}
	}
	err = r.initFirstFile()
	if err != nil {
		return err
	}
	return nil
}
