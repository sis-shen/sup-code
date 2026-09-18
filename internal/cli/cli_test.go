package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/internal"
)

func setupTestApp(t *testing.T) {
	t.Setenv("SUPCODE_LLM_API_KEY", "sk-cli-test")
	sc, err := internal.NewSupCode("")
	require.NoError(t, err)
	t.Cleanup(func() { sc.Close() })
	SetApp(sc)
}

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
	setupTestApp(t)
	cmd := RootCmd()
	require.NotNil(t, cmd)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"config", "list"})
	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "llm")
}

func TestConfigGetExistingKey(t *testing.T) {
	setupTestApp(t)
	cmd := RootCmd()
	require.NotNil(t, cmd)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"config", "get", "llm.model"})
	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "gpt-4o")
}

func TestVersionAndApp(t *testing.T) {
	cmd := RootCmd()
	require.NotNil(t, cmd)

	v := Version()
	assert.Equal(t, "0.1.0", v)
}
