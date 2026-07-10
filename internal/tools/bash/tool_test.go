package bash

import (
	"context"
	"encoding/json"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// shellCommand returns the platform-appropriate command string.
func shellCommand(t *testing.T, unixCmd, winCmd string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		return winCmd
	}
	return unixCmd
}

func TestGetShell_Platform(t *testing.T) {
	shell := getShell()
	if runtime.GOOS == "windows" {
		assert.Contains(t, shell, "powershell")
	} else {
		assert.Equal(t, "bash", shell)
	}
}

func TestGetShellFlag_Platform(t *testing.T) {
	flag := getShellFlag()
	if runtime.GOOS == "windows" {
		assert.Equal(t, "-Command", flag)
	} else {
		assert.Equal(t, "-c", flag)
	}
}

func TestTool_Bash_Basic(t *testing.T) {
	tool := &Tool{}
	assert.Equal(t, "bash", tool.Name())
	assert.NotEmpty(t, tool.Description())
	schema := tool.Schema()
	assert.Equal(t, "bash", schema.Name)
	assert.NotEmpty(t, schema.Parameters)

	cmd := shellCommand(t, "echo hello", "Write-Output hello")
	params, _ := json.Marshal(map[string]any{"command": cmd})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	var output map[string]any
	json.Unmarshal(result.Data, &output)
		stdout, _ := output["stdout"].(string)
		assert.Equal(t, "hello", strings.TrimSpace(stdout))
}

func TestTool_Bash_WithWorkDir(t *testing.T) {
	tool := &Tool{}
	cmd := shellCommand(t, "pwd", "Get-Location")
	params, _ := json.Marshal(map[string]any{"command": cmd})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestTool_Bash_CommandNotFound(t *testing.T) {
	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"command": "nonexistentcommand12345"})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.False(t, result.Success)
}

func TestTool_Bash_Timeout(t *testing.T) {
	tool := &Tool{}
	cmd := shellCommand(t, "sleep 10", "Start-Sleep -Seconds 10")
	params, _ := json.Marshal(map[string]any{"command": cmd, "timeout": 1})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.False(t, result.Success)

	var output map[string]any
	json.Unmarshal(result.Data, &output)
	exitCode, _ := output["exit_code"].(float64)
	assert.NotEqual(t, float64(0), exitCode)
}

func TestTool_Bash_EmptyCommand(t *testing.T) {
	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"command": ""})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "command is required")
}

func TestTool_Bash_InvalidParams(t *testing.T) {
	tool := &Tool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid}`))
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "invalid parameters")
}

func TestTool_Bash_ContextCancellation(t *testing.T) {
	tool := &Tool{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	params, _ := json.Marshal(map[string]any{"command": "echo hello"})
	_, err := tool.Execute(ctx, params)
	require.Error(t, err)
}

func TestTool_Bash_SchemaRequiredFields(t *testing.T) {
	var schema map[string]any
	require.NoError(t, json.Unmarshal(jsonSchema, &schema))
	required, ok := schema["required"].([]any)
	require.True(t, ok)
	assert.Contains(t, required, "command")
}

func TestTool_Bash_ExitCode(t *testing.T) {
	tool := &Tool{}
	cmd := shellCommand(t, "exit 42", "exit 42")
	params, _ := json.Marshal(map[string]any{"command": cmd})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.False(t, result.Success)

	var output map[string]any
	json.Unmarshal(result.Data, &output)
	assert.Equal(t, float64(42), output["exit_code"])
}

func TestTool_Bash_StderrCapture(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stderr redirect syntax differs on Windows")
	}
	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"command": "echo stderr >&2; echo stdout"})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	var output map[string]any
	json.Unmarshal(result.Data, &output)
	assert.Equal(t, "stdout\n", output["stdout"])
	assert.Equal(t, "stderr\n", output["stderr"])
}

func TestTool_Bash_OutputTruncation(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	if runtime.GOOS == "windows" {
		t.Skip("large output test uses Unix-specific command")
	}
	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"command": "python3 -c \"print('x' * 200000)\""})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	var output map[string]any
	json.Unmarshal(result.Data, &output)
	stdout, _ := output["stdout"].(string)
	assert.LessOrEqual(t, len(stdout), maxOutputSize)
}

func TestTool_Bash_NullByteOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("null byte printf command is Unix-specific")
	}
	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{
		"command": "printf \"\\x00hello\\x00world\"",
	})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	var output map[string]any
	unmarshalErr := json.Unmarshal(result.Data, &output)
	require.NoError(t, unmarshalErr, "result must be valid JSON even with null bytes in stdout")

	stdout, _ := output["stdout"].(string)
	assert.Contains(t, stdout, "hello")
	assert.Contains(t, stdout, "world")
	t.Logf("stdout length with null bytes: %d", len(stdout))
}
