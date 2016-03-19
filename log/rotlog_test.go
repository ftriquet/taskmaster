package logw

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestInitRotlog(t *testing.T) {
	rl, err := InitRotlog("testrotlog/rotlog", 1000, 8)
	assert.Nil(t, err)
	assert.NotNil(t, rl)
	assert.NotNil(t, rl.current)
	assert.Equal(t, "testrotlog/rotlog", rl.filename)
	assert.Equal(t, uint64(1000), rl.rot_every)
	assert.Equal(t, int(8), rl.nbFiles)
	_, err = os.Stat("testrotlog")
	if os.IsNotExist(err) {
		t.Errorf("Directory should exist, sacrebleu")
	}
	if rl.current != nil {
		rl.current.Close()
	}

	_, err = InitRotlog("", 8, 3)
	assert.NotNil(t, err)
	_, err = InitRotlog("bonjours", 0, 3333)
	assert.NotNil(t, err)
	_, err = InitRotlog("bonjours", 8, 0)
	assert.NotNil(t, err)
	_, err = InitRotlog("bonjours", 0, -46)
	assert.NotNil(t, err)
	//clean directory
	os.RemoveAll("testrotlog")
}

func TestIfFileExists(t *testing.T) {
	dir, err := ioutil.TempDir("", "testrotlog")
	if err != nil {
		fmt.Println("Failed to create test dir in TestIfFileExists, skipping...")
		return
	}
	testfile := dir + "/test"
	exists := fileExists(testfile)
	assert.Equal(t, false, exists)
	_, err = os.Create(testfile)
	if err != nil {
		fmt.Println("Failed to create test file in TestIfFileExists, skipping...")
		return
	}
	exists = fileExists(testfile)
	assert.Equal(t, true, exists)
	_ = os.RemoveAll(dir)
}

func TestFileNumberExists(t *testing.T) {
	var i int
	dir, err := ioutil.TempDir("", "testrotlog")
	if err != nil {
		fmt.Println("Failed to create test dir in TestIfFileExists, skipping...")
		return
	}
	r := new(Rotlog)
	r.nbFiles = 8
	r.filename = dir + "/test"
	//with no file
	for i = 0; i < r.nbFiles; i++ {
		assert.Equal(t, false, r.fileNumberExists(i))
	}
	for i = 0; i < r.nbFiles; i++ {
		_, err := r.createFileNb(i)
		if err != nil {
			fmt.Println("Failed to create file in TestFileNumberExists, skipping...")
		}
	}
	//with the files
	for i = 0; i < r.nbFiles; i++ {
		assert.Equal(t, true, r.fileNumberExists(i))
	}
	_ = os.RemoveAll(dir)
}

func TestShallRotate(t *testing.T) {
	var err error
	r := new(Rotlog)
	assert.Equal(t, true, r.shallRotate())
	r.current, err = os.Open("rotlog.go")
	defer r.current.Close()
	if err != nil {
		fmt.Println("Failed to open rotlog.go, where are we ? skipping...")
		return
	}
	r.current_size = 99
	r.rot_every = 100
	assert.Equal(t, false, r.shallRotate())
	r.current_size++
	assert.Equal(t, true, r.shallRotate())
}

func TestRotateFileNb(t *testing.T) {
	dir, err := ioutil.TempDir("", "testrotlog")
	if err != nil {
		fmt.Println("Failed to create test dir in TestIfFileExists, skipping...")
		return
	}
	r := new(Rotlog)
	r.nbFiles = 8
	r.filename = dir + "/test"
	_, err = r.createFileNb(0)
	if err != nil {
		fmt.Println("Failed to create test dir in TestIfFileExists, skipping...")
		return
	}
	assert.Equal(t, true, r.fileNumberExists(0))
	assert.Nil(t, r.rotateFileNb(0))
	assert.Equal(t, true, r.fileNumberExists(1))
	assert.Nil(t, r.rotateFileNb(1))
	assert.Equal(t, true, r.fileNumberExists(2))
	assert.NotNil(t, r.rotateFileNb(12))
	err = os.RemoveAll(dir)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
}

//with only 3 files
func TestSmallRotate(t *testing.T) {
	var i int
	dir, err := ioutil.TempDir("", "testrotlog")
	if err != nil {
		fmt.Println("Failed to create test dir in TestSmallRotate, skipping...")
		return
	}
	r := new(Rotlog)
	r.nbFiles = 8
	r.filename = dir + "/test"
	//with no file
	for i = 0; i < 3; i++ {
		_, err := r.createFileNb(i)
		if err != nil {
			fmt.Println("Failed to create file in TestFileNumberExists, skipping...")
		}
	}
	//just check if they are here
	for i = 0; i < 3; i++ {
		assert.Equal(t, true, r.fileNumberExists(i))
	}
	r.rotate()
	assert.NotNil(t, r.current)
	assert.Equal(t, uint64(0), r.current_size)
	defer r.current.Close()
	for i = 0; i < 4; i++ {
		if !r.fileNumberExists(i) {
			t.Errorf("File %d does not exist !", i)
		}
	}
	err = os.RemoveAll(dir)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
}

func TestFullRotate(t *testing.T) {
	var i int
	dir, err := ioutil.TempDir("", "testrotlog")
	if err != nil {
		fmt.Println("Failed to create test dir in TestIfFileExists, skipping...")
		return
	}
	r := new(Rotlog)
	r.nbFiles = 8
	r.filename = dir + "/test"
	//with no file
	for i = 0; i < r.nbFiles; i++ {
		_, err := r.createFileNb(i)
		if err != nil {
			fmt.Println("Failed to create file in TestFileNumberExists, skipping...")
		}
	}
	//just check if they are here
	for i = 0; i < r.nbFiles; i++ {
		assert.Equal(t, true, r.fileNumberExists(i))
	}
	r.rotate()
	assert.NotNil(t, r.current)
	assert.Equal(t, uint64(0), r.current_size)
	defer r.current.Close()
	for i = 0; i < r.nbFiles; i++ {
		assert.Equal(t, true, r.fileNumberExists(i))
	}
	assert.Equal(t, false, r.fileNumberExists(i+1))
	err = os.RemoveAll(dir)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
}

func TestWrite(t *testing.T) {
	const s string = "All work and no play makes Jack a dull boy"
	dir, err := ioutil.TempDir("", "testrotlog")
	if err != nil {
		fmt.Println("Failed to create test dir in TestIfFileExists, skipping...")
		return
	}
	r, err := InitRotlog(dir+"/test", 43, 3)
	assert.Nil(t, err)
	assert.NotNil(t, r)
	assert.Equal(t, dir+"/test", r.filename)
	assert.Equal(t, true, r.fileNumberExists(0))
	assert.NotNil(t, r.current)

	for i := 0; i < 10; i++ {
		n, err := r.Write([]byte(s))
		if err != nil {
			t.Errorf(err.Error())
		}
		assert.Equal(t, len(s)+1, n)
		//on lit le fichier, pour voir
		content, err := ioutil.ReadFile(r.filename)
		if err != nil {
			t.Errorf(err.Error())
		}
		assert.Equal(t, s+"\n", string(content))
		if i < 3 {
			assert.Equal(t, true, r.fileNumberExists(i))
		} else {
			assert.Equal(t, false, r.fileNumberExists(i))
		}
	}
	os.RemoveAll(dir)
}
