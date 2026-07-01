package editfile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTool_EditFile_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "edit.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("hello world foo bar"), 0644))

	tool := &Tool{}

	assert.Equal(t, "edit_file", tool.Name())
	assert.NotEmpty(t, tool.Description())
	schema := tool.Schema()
	assert.Equal(t, "edit_file", schema.Name)
	assert.NotEmpty(t, schema.Parameters)

	params, _ := json.Marshal(map[string]any{"path": filePath, "old_string": "world", "new_string": "there"})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	data, _ := os.ReadFile(filePath)
	assert.Equal(t, "hello there foo bar", string(data))
}

func TestTool_EditFile_OldStringNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "edit.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("hello world"), 0644))

	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"path": filePath, "old_string": "nonexistent", "new_string": "replacement"})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "not found")
}

func TestTool_EditFile_MultipleMatches(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "edit.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("foo foo foo"), 0644))

	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"path": filePath, "old_string": "foo", "new_string": "bar"})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "appears")
	assert.Contains(t, result.Error, "3 times")
}

func TestTool_EditFile_EmptyPath(t *testing.T) {
	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"path": "", "old_string": "old", "new_string": "new"})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "path is required")
}

func TestTool_EditFile_EmptyOldString(t *testing.T) {
	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"path": "/tmp/file.txt", "old_string": "", "new_string": "new"})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "old_string is required")
}

func TestTool_EditFile_InvalidParams(t *testing.T) {
	tool := &Tool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid}`))
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "invalid parameters")
}

func TestTool_EditFile_FileNotFound(t *testing.T) {
	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"path": "/nonexistent/file.txt", "old_string": "old", "new_string": "new"})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "failed to read file")
}

func TestTool_EditFile_ContextCancellation(t *testing.T) {
	tool := &Tool{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	params, _ := json.Marshal(map[string]any{"path": "/tmp/edit_test.txt", "old_string": "old", "new_string": "new"})
	_, err := tool.Execute(ctx, params)
	require.Error(t, err)
}

func TestTool_EditFile_JSONSchemaRequiredFields(t *testing.T) {
	var schema map[string]any
	require.NoError(t, json.Unmarshal(jsonSchema, &schema))
	required, ok := schema["required"].([]any)
	require.True(t, ok)
	assert.Contains(t, required, "path")
	assert.Contains(t, required, "old_string")
	assert.Contains(t, required, "new_string")
}
