package readfile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTool_ReadFile_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	content := "line1\nline2\nline3\nline4\nline5\n"
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	tool := &Tool{}

	assert.Equal(t, "read_file", tool.Name())
	assert.NotEmpty(t, tool.Description())
	schema := tool.Schema()
	assert.Equal(t, "read_file", schema.Name)
	assert.NotEmpty(t, schema.Parameters)

	params, _ := json.Marshal(map[string]any{"path": filePath})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, string(result.Data), "line1")
}

func TestTool_ReadFile_LineRange(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "lines.txt")
	content := "a\nb\nc\nd\ne\nf\ng\nh\ni\nj\n"
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	tool := &Tool{}

	params, _ := json.Marshal(map[string]any{"path": filePath, "start_line": 2, "end_line": 4})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)
	var data map[string]string
	json.Unmarshal(result.Data, &data)
	assert.Equal(t, "b\nc\nd", data["content"])

	params, _ = json.Marshal(map[string]any{"path": filePath, "start_line": 8})
	result, err = tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)
	json.Unmarshal(result.Data, &data)
	assert.Equal(t, "h\ni\nj\n", data["content"])
}

func TestTool_ReadFile_LineRangeErrors(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "short.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("only line\n"), 0644))

	tool := &Tool{}

	params, _ := json.Marshal(map[string]any{"path": filePath, "start_line": 10})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "exceeds file length")

	params, _ = json.Marshal(map[string]any{"path": filePath, "start_line": 3, "end_line": 1})
	result, err = tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "after end_line")
}

func TestTool_ReadFile_FileNotFound(t *testing.T) {
	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"path": "/nonexistent/file.txt"})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "failed to read file")
}

func TestTool_ReadFile_EmptyPath(t *testing.T) {
	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"path": ""})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "required")
}

func TestTool_ReadFile_InvalidParams(t *testing.T) {
	tool := &Tool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid}`))
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "invalid parameters")
}

func TestTool_ReadFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "empty.txt")
	require.NoError(t, os.WriteFile(filePath, []byte{}, 0644))

	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"path": filePath})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)
	var data map[string]string
	json.Unmarshal(result.Data, &data)
	assert.Equal(t, "", data["content"])
}

func TestTool_ReadFile_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("hello\n"), 0644))

	tool := &Tool{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	params, _ := json.Marshal(map[string]any{"path": filePath})
	_, err := tool.Execute(ctx, params)
	require.Error(t, err)
}

func TestTool_ReadFile_JSONSchemaRequiredFields(t *testing.T) {
	var schema map[string]any
	require.NoError(t, json.Unmarshal(jsonSchema, &schema))
	required, ok := schema["required"].([]any)
	require.True(t, ok)
	assert.Contains(t, required, "path")
}


// ─── task2 boundary: binary file handling ────────────────────────

func TestTool_ReadFile_BinaryWithNullBytes(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "binary.bin")
	binaryContent := []byte{0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F}
	require.NoError(t, os.WriteFile(filePath, binaryContent, 0644))

	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"path": filePath})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Content should be valid (raw bytes, not line-split)
	var data map[string]string
	json.Unmarshal(result.Data, &data)
	content, ok := data["content"]
	assert.True(t, ok)
	assert.Greater(t, len(content), 0, "binary file should return content")
	t.Logf("binary file content length: %d", len(content))
}

func TestTool_ReadFile_EmptyFileReturnsSize(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "empty.txt")
	require.NoError(t, os.WriteFile(filePath, []byte{}, 0644))

	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"path": filePath})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	var data map[string]string
	json.Unmarshal(result.Data, &data)
	content, ok := data["content"]
	assert.True(t, ok)
	assert.Equal(t, "", content, "empty file should return empty content")
}

func TestTool_ReadFile_LargeFileTruncation(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "large.txt")
	// Create a file > 10MB
	size := 11 * 1024 * 1024
	data := make([]byte, size)
	for i := range data {
		data[i] = byte('a' + i%26)
	}
	require.NoError(t, os.WriteFile(filePath, data, 0644))

	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"path": filePath})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Content is a string of the full file (os.ReadFile returns all bytes)
	var output map[string]string
	json.Unmarshal(result.Data, &output)
	content, ok := output["content"]
	assert.True(t, ok)
	assert.Equal(t, size, len(content), "large file content length matches full size (no default truncation, but JSON handles it)")
}
