package tools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/core"
	"github.com/supcode/supcode/pkg"
)

// fakePermission is a PermissionEngine that allows every action. It keeps the
// test independent of plugin-permission.
type fakePermission struct{}

func (fakePermission) Check(context.Context, pkg.Action) (pkg.Decision, error) {
	return pkg.DecisionAllow, nil
}

func (fakePermission) AddRule(pkg.PermissionRule) error { return nil }

func (fakePermission) RemoveRule(string) error { return nil }

func (fakePermission) ListRules() []pkg.PermissionRule { return nil }

func (fakePermission) LogAction(context.Context, pkg.Action, pkg.Decision, string) error {
	return nil
}

func fakePermissionPlugin() core.Plugin {
	return core.Plugin{
		Name:     "fake-permission",
		Provides: []string{pkg.ServicePermission},
		Apply: func(ctx *core.Context) error {
			core.Provide(ctx, pkg.ServicePermission, fakePermission{})
			return nil
		},
	}
}

// loadTools builds a kernel with the fake permission engine and plugin-tools.
func loadTools(t *testing.T) (*core.Kernel, *core.Context, pkg.ToolRegistry) {
	t.Helper()
	k := core.New()
	root := k.Root()
	require.NoError(t, k.Load(root, fakePermissionPlugin()))
	require.NoError(t, k.Load(root, Plugin(Options{})))
	reg := core.Use[pkg.ToolRegistry](root, pkg.ServiceTools)
	require.NotNil(t, reg)
	return k, root, reg
}

func TestPluginRegistersBuiltinTools(t *testing.T) {
	_, _, reg := loadTools(t)
	assert.ElementsMatch(t,
		[]string{"bash", "read_file", "write_file", "edit_file", "glob", "grep"},
		reg.List())
}

func TestManifestMatchesPlugin(t *testing.T) {
	assert.Equal(t, "plugin-tools", Manifest().Name)
	assert.Equal(t, []string{"tools"}, Manifest().Provides)
	assert.Equal(t, []string{"permission"}, Manifest().Inject)
	assert.Equal(t, "builtin:tools", Manifest().Entry)

	p := Plugin(Options{})
	assert.Equal(t, Manifest().Name, p.Name)
	assert.Equal(t, Manifest().Provides, p.Provides)
	assert.Equal(t, Manifest().Inject, p.Inject)
}

func TestExecuteEmitsResultAndRunsTools(t *testing.T) {
	_, root, reg := loadTools(t)

	var invoked []string
	root.On(pkg.EventToolsResult, func(_ context.Context, payload any, _ core.Next) (any, error) {
		inv, ok := payload.(*pkg.ToolInvocation)
		require.True(t, ok)
		invoked = append(invoked, inv.Name)
		return nil, nil
	})

	dir := t.TempDir()
	path := filepath.Join(dir, "note.txt")

	writeParams, err := json.Marshal(map[string]string{"path": path, "content": "hello sup"})
	require.NoError(t, err)
	res, err := reg.Execute(context.Background(), "write_file", writeParams)
	require.NoError(t, err)
	require.True(t, res.Success)

	readParams, err := json.Marshal(map[string]string{"path": path})
	require.NoError(t, err)
	res, err = reg.Execute(context.Background(), "read_file", readParams)
	require.NoError(t, err)
	require.True(t, res.Success)
	assert.Contains(t, string(res.Data), "hello sup")

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello sup", string(data))

	assert.Equal(t, []string{"write_file", "read_file"}, invoked)
}

func TestPreExecuteShortCircuits(t *testing.T) {
	k, root, reg := loadTools(t)

	sentinel := errors.New("blocked by policy")
	root.On(pkg.EventToolsPreExecute, func(_ context.Context, _ any, _ core.Next) (any, error) {
		return nil, sentinel
	})

	var resultEvents int
	k.Root().On(pkg.EventToolsResult, func(context.Context, any, core.Next) (any, error) {
		resultEvents++
		return nil, nil
	})

	_, err := reg.Execute(context.Background(), "bash", json.RawMessage(`{"command":"echo hi"}`))
	require.ErrorIs(t, err, sentinel)
	assert.Zero(t, resultEvents)
}

func TestRegistryDelegates(t *testing.T) {
	_, _, reg := loadTools(t)

	assert.NotEmpty(t, reg.ListSchemas())
	assert.Len(t, reg.ListSchemas(), 6)

	get, err := reg.Get("read_file")
	require.NoError(t, err)
	assert.Equal(t, "read_file", get.Name())

	require.NoError(t, reg.Unregister("grep"))
	assert.NotContains(t, reg.List(), "grep")
	assert.Error(t, reg.Unregister("grep"))

	require.NoError(t, reg.Register(&fakeTool{name: "custom"}))
	assert.Contains(t, reg.List(), "custom")

	hook := &fakeHook{name: "audit"}
	require.NoError(t, reg.RegisterHook(hook))
	assert.Equal(t, []string{"audit"}, reg.ListHookNames())
	require.NoError(t, reg.UnregisterHook("audit"))
	assert.Empty(t, reg.ListHookNames())
	assert.Error(t, reg.UnregisterHook("missing"))
}

type fakeTool struct{ name string }

func (f *fakeTool) Name() string        { return f.name }
func (f *fakeTool) Description() string { return "fake" }
func (f *fakeTool) Schema() pkg.ToolSchema {
	return pkg.ToolSchema{Name: f.name, Description: "fake"}
}
func (f *fakeTool) Execute(context.Context, json.RawMessage) (pkg.ToolResult, error) {
	return pkg.ToolResult{Success: true}, nil
}

type fakeHook struct{ name string }

func (f *fakeHook) Name() string { return f.name }
func (f *fakeHook) BeforeTool(_ context.Context, _ string, params json.RawMessage) (json.RawMessage, error) {
	return params, nil
}
func (f *fakeHook) AfterTool(context.Context, string, json.RawMessage, pkg.ToolResult) error {
	return nil
}
