package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValid(t *testing.T) {
	proc := NewProc()
	proc.Command = "/bin/ls"
	proc.Name = "\tlol   swag"
	assert.False(t, proc.IsValid())
	proc.Name = "Unnomtreslong"
	assert.True(t, proc.IsValid())
	proc.Name = strings.TrimSpace("     LOL      ")
	assert.True(t, proc.IsValid())
	proc.Name = "lol\rswag"
	assert.False(t, proc.IsValid())
	proc.Name = ""
	assert.False(t, proc.IsValid())
	proc.Command = ""
	proc.Name = "Nom"
	assert.False(t, proc.IsValid())
}
