package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProc(t *testing.T) {
	p := NewProc()
	if p.Umask != 022 {
		t.Errorf("Wrong umask")
	}
	assert.Equal(t, 3, p.StartRetries)
	assert.Equal(t, false, p.AutoStart)
	assert.Equal(t, DflNumProcs, p.NumProcs)
	assert.Equal(t, DflAutoRestart, p.AutoRestart)
	g_procs = map[string]*Process{}
}

func TestLoadFile(t *testing.T) {
	err := LoadFile("config.json")
	assert.Nil(t, err)
	proc, exists := g_procs["TailDeFou"]
	assert.Equal(t, exists, true)
	assert.Equal(t, proc.Command, "/usr/bin/tail -f /tmp/FICHIER")
	assert.Equal(t, proc.Outfile, "/tmp/tail_log_out")
	assert.Equal(t, proc.Errfile, "/tmp/tail_log_err")
	if g_procs["TailDeFou"].Umask != 022 {
		t.Errorf("LOL T NULL")
	}
	g_procs = map[string]*Process{}
}

func TestEmptyFields(t *testing.T) {
	err := LoadFile("invalid.json")
	assert.Nil(t, err)
	_, exists := g_procs["NONAME"]
	assert.False(t, exists)
	_, exists = g_procs[""]
	assert.False(t, exists)
	_, exists = g_procs["NOCOMMAND"]
	assert.False(t, exists)
	_, exists = g_procs["NORMAL"]
	assert.True(t, exists)
	assert.Equal(t, 1, len(g_procs))
	g_procs = map[string]*Process{}
}
