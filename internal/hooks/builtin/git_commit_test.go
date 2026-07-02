package builtin

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supcode/supcode/pkg"
)

func TestGitCommitHook_Disabled(t *testing.T) {
	hook := NewGitCommitHook(false)
	assert.False(t, hook.IsEnabled())
	err := hook.AfterTool(context.Background(), "write_file", json.RawMessage(`{"path":"test.txt"}`), pkg.ToolResult{Success: true})
	require.NoError(t, err)
}

func TestGitCommitHook_NonFileTool(t *testing.T) {
	hook := NewGitCommitHook(true)
	err := hook.AfterTool(context.Background(), "bash", json.RawMessage(`{"command":"echo hi"}`), pkg.ToolResult{Success: true})
	require.NoError(t, err)
}

func TestGitCommitHook_FailedTool(t *testing.T) {
	hook := NewGitCommitHook(true)
	err := hook.AfterTool(context.Background(), "write_file", json.RawMessage(`{"path":"/tmp/test.txt"}`), pkg.ToolResult{Success: false})
	require.NoError(t, err)
}

func TestGitCommitHook_NotInGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	hook := NewGitCommitHook(true)
	filePath := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("test"), 0644))
	params, _ := json.Marshal(map[string]any{"path": filePath})
	err := hook.AfterTool(context.Background(), "write_file", params, pkg.ToolResult{Success: true})
	require.NoError(t, err)
}

func TestGitCommitHook_InGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	runCmd(t, tmpDir, "git", "init")
	runCmd(t, tmpDir, "git", "config", "user.email", "test@test.com")
	runCmd(t, tmpDir, "git", "config", "user.name", "Test")

	initialFile := filepath.Join(tmpDir, "initial.txt")
	require.NoError(t, os.WriteFile(initialFile, []byte("initial"), 0644))
	runCmd(t, tmpDir, "git", "add", "initial.txt")
	runCmd(t, tmpDir, "git", "commit", "-m", "initial")

	hook := NewGitCommitHook(true)
	newFile := filepath.Join(tmpDir, "new_file.txt")
	require.NoError(t, os.WriteFile(newFile, []byte("new content"), 0644))
	params, _ := json.Marshal(map[string]any{"path": newFile})
	err := hook.AfterTool(context.Background(), "write_file", params, pkg.ToolResult{Success: true})
	require.NoError(t, err)

	out := runCmd(t, tmpDir, "git", "log", "--oneline")
	assert.Contains(t, string(out), "SupCode: write_file")
}

func TestGitCommitHook_Basic(t *testing.T) {
	hook := NewGitCommitHook(true)
	assert.Equal(t, "git_auto_commit", hook.Name())
	assert.True(t, hook.IsEnabled())

	params := json.RawMessage(`{"path":"test.txt"}`)
	result, err := hook.BeforeTool(context.Background(), "write_file", params)
	require.NoError(t, err)
	assert.Equal(t, params, result)

	hook.Disable()
	assert.False(t, hook.IsEnabled())
	hook.Enable()
	assert.True(t, hook.IsEnabled())
}

func runCmd(t *testing.T, dir, name string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "cmd %s %v failed: %s", name, args, string(out))
	return out
}
