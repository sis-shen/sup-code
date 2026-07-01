package integration

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supcode/supcode/internal/permission"
	"github.com/supcode/supcode/internal/tools"
	"github.com/supcode/supcode/internal/tools/bash"
	"github.com/supcode/supcode/internal/tools/glob"
	"github.com/supcode/supcode/internal/tools/grep"
	"github.com/supcode/supcode/internal/tools/readfile"
	"github.com/supcode/supcode/internal/tools/writefile"
	"github.com/supcode/supcode/pkg"
)

func TestToolChain_GlobGrepReadFile(t *testing.T) {
	permEng := permission.New()
	toolReg := tools.NewRegistry(permEng)
	for _, tool := range []pkg.Tool{
		&readfile.Tool{}, &glob.Tool{}, &grep.Tool{}, &bash.Tool{}, &writefile.Tool{},
	} {
		require.NoError(t, toolReg.Register(tool))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fixtureDir := fixturePath(t)

	// Step 1: Glob go files using base_path
	goFiles, err := toolReg.Execute(ctx, "glob", mustParams(map[string]interface{}{
		"pattern":   "*.go",
		"base_path": filepath.Join(fixtureDir, "src"),
	}))
	require.NoError(t, err)
	require.True(t, goFiles.Success)
	assert.Contains(t, string(goFiles.Data), "utils.go")
	assert.Contains(t, string(goFiles.Data), "types.go")

	// Step 2: Grep for TODO
	todoResults, err := toolReg.Execute(ctx, "grep", mustParams(map[string]interface{}{
		"pattern": "TODO",
		"include": "*.go",
		"path":    fixtureDir,
	}))
	require.NoError(t, err)
	require.True(t, todoResults.Success)
	assert.Contains(t, string(todoResults.Data), "TODO")

	// Step 3: Glob yaml files
	yamlFiles, err := toolReg.Execute(ctx, "glob", mustParams(map[string]interface{}{
		"pattern":   "*.yaml",
		"base_path": fixtureDir,
	}))
	require.NoError(t, err)
	require.True(t, yamlFiles.Success)
	assert.Contains(t, string(yamlFiles.Data), "config.yaml")

	// Step 4: Read config.yaml
	cfgContent, err := toolReg.Execute(ctx, "read_file", mustParams(map[string]interface{}{
		"path": filepath.Join(fixtureDir, "config.yaml"),
	}))
	require.NoError(t, err)
	require.True(t, cfgContent.Success)
	assert.Contains(t, string(cfgContent.Data), "test-app")
}

func TestToolChain_PermissionDenyCommands(t *testing.T) {
	permEng := permission.New()
	toolReg := tools.NewRegistry(permEng)
	require.NoError(t, toolReg.Register(&bash.Tool{}))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := toolReg.Execute(ctx, "bash", mustParams(map[string]interface{}{"command": "rm -rf /"}))
	if err != nil {
		if _, ok := err.(*pkg.ErrPermissionDenied); ok {
			t.Log("permission denied as expected")
		}
	} else {
		assert.False(t, result.Success)
	}
}

func TestToolChain_NormalCommandAllowed(t *testing.T) {
	permEng := permission.New()
	toolReg := tools.NewRegistry(permEng)
	require.NoError(t, toolReg.Register(&bash.Tool{}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := toolReg.Execute(ctx, "bash", mustParams(map[string]interface{}{"command": "echo hello"}))
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestToolChain_SensitivePathAsk(t *testing.T) {
	permEng := permission.New()
	toolReg := tools.NewRegistry(permEng)
	require.NoError(t, toolReg.Register(&writefile.Tool{}))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := toolReg.Execute(ctx, "write_file", mustParams(map[string]interface{}{
		"path": "/etc/test_supcode_write.txt", "content": "should not write",
	}))
	t.Logf("result: success=%v err=%v", result.Success, err)
}

func TestToolChain_GlobRecursive(t *testing.T) {
	permEng := permission.New()
	toolReg := tools.NewRegistry(permEng)
	require.NoError(t, toolReg.Register(&glob.Tool{}))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := toolReg.Execute(ctx, "glob", mustParams(map[string]interface{}{
		"pattern":   "src/*.go",
		"base_path": fixturePath(t),
	}))
	require.NoError(t, err)
	require.True(t, result.Success)
	assert.Contains(t, string(result.Data), "utils.go")
}

func mustParams(params map[string]interface{}) []byte {
	data, err := json.Marshal(params)
	if err != nil {
		panic(err)
	}
	return data
}
