package grep

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTool_Grep_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("hello world\nfoo bar\nhello again\n"), 0644))

	tool := &Tool{}

	assert.Equal(t, "grep", tool.Name())
	assert.NotEmpty(t, tool.Description())
	schema := tool.Schema()
	assert.Equal(t, "grep", schema.Name)
	assert.NotEmpty(t, schema.Parameters)

	params, _ := json.Marshal(map[string]any{"pattern": "hello", "path": filePath})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	var output map[string]any
	require.NoError(t, json.Unmarshal(result.Data, &output))
	matches, _ := output["matches"].([]any)
	assert.Len(t, matches, 2)
}

func TestTool_Grep_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("hello world"), 0644))

	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"pattern": "nonexistent", "path": filePath})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	var output map[string]any
	require.NoError(t, json.Unmarshal(result.Data, &output))
	assert.Equal(t, float64(0), output["count"])
}

func TestTool_Grep_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("match in a\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("nothing here\n"), 0644))

	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"pattern": "match in", "path": tmpDir})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	var output map[string]any
	require.NoError(t, json.Unmarshal(result.Data, &output))
	matches, _ := output["matches"].([]any)
	assert.Len(t, matches, 1)
}

func TestTool_Grep_ContextLines(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	content := "before1\nbefore2\nmatch\nafter1\nafter2\n"
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"pattern": "match", "path": filePath, "context_lines": 2})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	var output map[string]any
	require.NoError(t, json.Unmarshal(result.Data, &output))
	matches, _ := output["matches"].([]any)
	require.Len(t, matches, 1)

	m := matches[0].(map[string]any)
	before, _ := m["before"].([]any)
	after, _ := m["after"].([]any)
	assert.Len(t, before, 2)
	assert.Len(t, after, 2)
}

func TestTool_Grep_EmptyPattern(t *testing.T) {
	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"pattern": "", "path": "/tmp"})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "pattern is required")
}

func TestTool_Grep_EmptyPath(t *testing.T) {
	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"pattern": "hello", "path": ""})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "path is required")
}

func TestTool_Grep_InvalidRegex(t *testing.T) {
	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"pattern": "[invalid", "path": "/tmp"})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "invalid regex")
}

func TestTool_Grep_InvalidParams(t *testing.T) {
	tool := &Tool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid}`))
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "invalid parameters")
}

func TestTool_Grep_ContextCancellation(t *testing.T) {
	tool := &Tool{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	params, _ := json.Marshal(map[string]any{"pattern": "hello", "path": "/tmp"})
	_, err := tool.Execute(ctx, params)
	require.Error(t, err)
}

func TestTool_Grep_FileNotFound(t *testing.T) {
	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"pattern": "hello", "path": "/nonexistent/path/file.txt"})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "path error")
}

func TestTool_Grep_BinaryFileSkip(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.png")
	require.NoError(t, os.WriteFile(filePath, []byte("fake png content"), 0644))

	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"pattern": "content", "path": tmpDir})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	var output map[string]any
	require.NoError(t, json.Unmarshal(result.Data, &output))
	assert.Equal(t, float64(0), output["count"])
}

func TestTool_Grep_LiteralMatch(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("foo.bar(1)\n"), 0644))

	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"pattern": "foo.bar(1)", "path": filePath, "pattern_is_regex": false})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	var output map[string]any
	require.NoError(t, json.Unmarshal(result.Data, &output))
	assert.Equal(t, float64(1), output["count"])
}

func TestTool_Grep_JSONSchemaRequiredFields(t *testing.T) {
	var schema map[string]any
	require.NoError(t, json.Unmarshal(jsonSchema, &schema))
	required, ok := schema["required"].([]any)
	require.True(t, ok)
	assert.Contains(t, required, "pattern")
	assert.Contains(t, required, "path")
}

func TestGrepFile_BinaryContent(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "binary.bin")
	require.NoError(t, os.WriteFile(filePath, []byte{0x00, 0x01, 0x02, 0x03}, 0644))

	re := regexp.MustCompile("test")
	matches, err := grepFile(filePath, re, 0, 100)
	require.NoError(t, err)
	assert.Empty(t, matches)
}
