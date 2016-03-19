package logw

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInitLogw(t *testing.T) {
	assert.Nil(t, g_info)
	assert.Nil(t, g_warning)
	assert.Nil(t, g_alert)
	assert.Nil(t, g_err)
	Init()
	assert.NotNil(t, g_info)
	assert.NotNil(t, g_warning)
	assert.NotNil(t, g_alert)
	assert.NotNil(t, g_err)
	//reset them to null for next test
	g_info = nil
	g_warning = nil
	g_alert = nil
	g_err = nil
}

func TestInitSilentLogw(t *testing.T) {
	assert.Nil(t, g_info)
	assert.Nil(t, g_warning)
	assert.Nil(t, g_alert)
	assert.Nil(t, g_err)
	InitSilent()
	assert.NotNil(t, g_info)
	assert.NotNil(t, g_warning)
	assert.NotNil(t, g_alert)
	assert.NotNil(t, g_err)
}
