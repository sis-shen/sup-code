package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootHelp(t *testing.T) {
	cmd := RootCmd()
	require.NotNil(t, cmd)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "SupCode")
	assert.Contains(t, buf.String(), "Usage:")
}

func TestRootVersion(t *testing.T) {
	cmd := RootCmd()
	require.NotNil(t, cmd)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})
	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), version)
}

func TestConfigHelp(t *testing.T) {
	cmd := RootCmd()
	require.NotNil(t, cmd)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"config", "--help"})
	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "config")
	assert.Contains(t, buf.String(), "Manage")
}

func TestConfigList(t *testing.T) {
	cmd := RootCmd()
	require.NotNil(t, cmd)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"config", "list"})
	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "llm = map")
}

func TestConfigGetExistingKey(t *testing.T) {
	cmd := RootCmd()
	require.NotNil(t, cmd)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"config", "get", "llm.model"})
	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "gpt-4o")
}

func TestConfigSetAndGet(t *testing.T) {
	cmd := RootCmd()
	require.NotNil(t, cmd)

	// Set a value
	setBuf := new(bytes.Buffer)
	cmd.SetOut(setBuf)
	cmd.SetArgs([]string{"config", "set", "llm.model", "gpt-4o-mini"})
	err := cmd.Execute()
	require.NoError(t, err)

	// Get the value
	getCmd := RootCmd()
	getBuf := new(bytes.Buffer)
	getCmd.SetOut(getBuf)
	getCmd.SetArgs([]string{"config", "get", "llm.model"})
	err = getCmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, getBuf.String(), "gpt-4o-mini")
}

func TestVersionAndConfigManager(t *testing.T) {
	cmd := RootCmd()
	require.NotNil(t, cmd)

	v := Version()
	assert.Equal(t, "0.1.0", v)

	cm := ConfigManager()
	assert.NotNil(t, cm)
}
