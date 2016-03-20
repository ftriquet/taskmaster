package main

import (
	"taskmaster/common"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProc(t *testing.T) {
	p := common.NewProc()
	if p.Umask != 022 {
		t.Errorf("Wrong umask")
	}
	assert.Equal(t, 3, p.StartRetries)
	assert.Equal(t, false, p.AutoStart)
	assert.Equal(t, common.DflNumProcs, p.NumProcs)
	assert.Equal(t, common.DflAutoRestart, p.AutoRestart)
	g_procs = map[string]*common.Process{}
}

func TestLoadFile(t *testing.T) {
	err := LoadFile("../config/config.json")
	assert.Nil(t, err)
	proc, exists := g_procs["TailDeFou0"]
	assert.Equal(t, exists, true)
	assert.Equal(t, proc.Command, "/usr/bin/tail -f /tmp/FICHIER")
	assert.Equal(t, proc.Outfile, "/tmp/tail_log_out")
	assert.Equal(t, proc.Errfile, "/tmp/tail_log_err")
	assert.Equal(t, 4, len(g_procs))
	if g_procs["TailDeFou0"].Umask != 022 {
		t.Errorf("LOL T NULL")
	}
	g_procs = map[string]*common.Process{}
}

func TestEmptyFields(t *testing.T) {
	err := LoadFile("../config/invalid.json")
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
	g_procs = map[string]*common.Process{}
}

func TestPassword(t *testing.T) {
	err := LoadFile("../config/password.json")
	if err != nil {
		t.Fatal()
	}
	t.Log("Test mot de passe correct (motdepasse)\n")
	assert.True(t, checkPassword(true))
	t.Log("Test mot de passe incorrect (motdepasse)\n")
	assert.False(t, checkPassword(true))
}
