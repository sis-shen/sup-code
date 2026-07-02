package cli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndLoadCustomCommands(t *testing.T) {
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	oldProfile := os.Getenv("USERPROFILE")
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)
	defer func() { os.Setenv("HOME", oldHome); os.Setenv("USERPROFILE", oldProfile) }()

	cmds := []CustomCommand{{Name: "review", Description: "Code review", Prompt: "Review the code", Tools: []string{"bash", "readfile"}}, {Name: "deploy", Description: "Deploy", Prompt: "Deploy to staging"}}
	err := SaveCustomCommands(cmds)
	require.NoError(t, err)

	loaded, err := LoadCustomCommands()
	require.NoError(t, err)
	assert.Len(t, loaded, 2)
}

func TestLoadCustomCommands_NotFound(t *testing.T) {
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)
	cmds, err := LoadCustomCommands()
	require.NoError(t, err)
	assert.Len(t, cmds, 0, "should be nil or empty when commands file does not exist")
}

func TestSaveCustomCommands_Empty(t *testing.T) {
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)
	err := SaveCustomCommands([]CustomCommand{})
	require.NoError(t, err)
	loaded, err := LoadCustomCommands()
	require.NoError(t, err)
	assert.Empty(t, loaded)
}
