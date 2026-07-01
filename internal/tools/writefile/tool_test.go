package writefile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTool_WriteFile_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "output.txt")

	tool := &Tool{}

	assert.Equal(t, "write_file", tool.Name())
	assert.NotEmpty(t, tool.Description())
	schema := tool.Schema()
	assert.Equal(t, "write_file", schema.Name)
	assert.NotEmpty(t, schema.Parameters)

	content := "hello world"
	params, _ := json.Marshal(map[string]any{"path": filePath, "content": content})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestTool_WriteFile_MkdirAll(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "a", "b", "c", "nested.txt")

	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"path": nestedPath, "content": "nested content"})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	_, err = os.Stat(nestedPath)
	require.NoError(t, err)
	data, err := os.ReadFile(nestedPath)
	require.NoError(t, err)
	assert.Equal(t, "nested content", string(data))
}

func TestTool_WriteFile_EmptyContent(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "empty.txt")

	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"path": filePath, "content": ""})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Empty(t, data)
}

func TestTool_WriteFile_EmptyPath(t *testing.T) {
	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"path": "", "content": "content"})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "required")
}

func TestTool_WriteFile_InvalidParams(t *testing.T) {
	tool := &Tool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid}`))
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "invalid parameters")
}

func TestTool_WriteFile_ContextCancellation(t *testing.T) {
	tool := &Tool{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	params, _ := json.Marshal(map[string]any{"path": "/tmp/test_write.txt", "content": "content"})
	_, err := tool.Execute(ctx, params)
	require.Error(t, err)
}

func TestTool_WriteFile_JSONSchemaRequiredFields(t *testing.T) {
	var schema map[string]any
	require.NoError(t, json.Unmarshal(jsonSchema, &schema))
	required, ok := schema["required"].([]any)
	require.True(t, ok)
	assert.Contains(t, required, "path")
	assert.Contains(t, required, "content")
}
