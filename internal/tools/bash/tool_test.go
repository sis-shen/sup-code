package bash

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTool_Bash_Basic(t *testing.T) {
	tool := &Tool{}

	assert.Equal(t, "bash", tool.Name())
	assert.NotEmpty(t, tool.Description())
	schema := tool.Schema()
	assert.Equal(t, "bash", schema.Name)
	assert.NotEmpty(t, schema.Parameters)

	params, _ := json.Marshal(map[string]any{"command": "echo hello"})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	var output map[string]any
	json.Unmarshal(result.Data, &output)
	assert.Equal(t, "hello\n", output["stdout"])
	assert.Equal(t, float64(0), output["exit_code"])
}

func TestTool_Bash_WithWorkDir(t *testing.T) {
	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"command": "pwd"})
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
	params, _ := json.Marshal(map[string]any{"command": "sleep 10", "timeout": 1})
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
	params, _ := json.Marshal(map[string]any{"command": "exit 42"})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.False(t, result.Success)

	var output map[string]any
	json.Unmarshal(result.Data, &output)
	assert.Equal(t, float64(42), output["exit_code"])
}

func TestTool_Bash_StderrCapture(t *testing.T) {
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
	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"command": "python3 -c \"print('x' * 200000)\""})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	var output map[string]any
	json.Unmarshal(result.Data, &output)
	stdout, _ := output["stdout"].(string)
	assert.LessOrEqual(t, len(stdout), maxOutputSize)
}
