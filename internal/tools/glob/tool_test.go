package glob

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTool_Glob_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("a"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("b"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "c.go"), []byte("c"), 0644))

	tool := &Tool{}

	assert.Equal(t, "glob", tool.Name())
	assert.NotEmpty(t, tool.Description())
	schema := tool.Schema()
	assert.Equal(t, "glob", schema.Name)
	assert.NotEmpty(t, schema.Parameters)

	params, _ := json.Marshal(map[string]any{"pattern": "*.txt", "base_path": tmpDir})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	var output map[string]any
	require.NoError(t, json.Unmarshal(result.Data, &output))
	matches, _ := output["matches"].([]any)
	assert.Len(t, matches, 2)
}

func TestTool_Glob_Doublestar(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "sub1"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "sub2"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "sub1", "a.txt"), []byte("a"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "sub2", "b.txt"), []byte("b"), 0644))

	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"pattern": "**/*.txt", "base_path": tmpDir})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	var output map[string]any
	require.NoError(t, json.Unmarshal(result.Data, &output))
	matches, _ := output["matches"].([]any)
	assert.Len(t, matches, 2)
}

func TestTool_Glob_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()

	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"pattern": "*.nonexistent", "base_path": tmpDir})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	var output map[string]any
	require.NoError(t, json.Unmarshal(result.Data, &output))
	assert.Equal(t, float64(0), output["count"])
}

func TestTool_Glob_EmptyPattern(t *testing.T) {
	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"pattern": ""})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "pattern is required")
}

func TestTool_Glob_InvalidParams(t *testing.T) {
	tool := &Tool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid}`))
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "invalid parameters")
}

func TestTool_Glob_ContextCancellation(t *testing.T) {
	tool := &Tool{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	params, _ := json.Marshal(map[string]any{"pattern": "*.txt"})
	_, err := tool.Execute(ctx, params)
	require.Error(t, err)
}

func TestTool_Glob_MaxResults(t *testing.T) {
	tmpDir := t.TempDir()
	for i := 0; i < 5; i++ {
		baseName := "file" + string(rune('0'+i)) + ".txt"
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, baseName), []byte("x"), 0644))
	}

	tool := &Tool{}
	params, _ := json.Marshal(map[string]any{"pattern": "*.txt", "base_path": tmpDir, "max_results": 3})
	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	var output map[string]any
	require.NoError(t, json.Unmarshal(result.Data, &output))
	count, _ := output["count"].(float64)
	assert.Equal(t, float64(3), count)
}

func TestTool_Glob_JSONSchemaRequiredFields(t *testing.T) {
	var schema map[string]any
	require.NoError(t, json.Unmarshal(jsonSchema, &schema))
	required, ok := schema["required"].([]any)
	require.True(t, ok)
	assert.Contains(t, required, "pattern")
}
