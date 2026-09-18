package skill

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/core"
	skills "github.com/supcode/supcode/internal/skill"
	skilltool "github.com/supcode/supcode/internal/tools/skilltool"
	"github.com/supcode/supcode/pkg"
)

// recordingRegistry is a pkg.ToolRegistry that records registered tool names.
// It keeps the test independent of plugin-tools.
type recordingRegistry struct {
	mu    sync.Mutex
	tools []string
}

func (r *recordingRegistry) Register(tool pkg.Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools = append(r.tools, tool.Name())
	return nil
}

func (r *recordingRegistry) Unregister(string) error { return nil }

func (r *recordingRegistry) Get(string) (pkg.Tool, error) { return nil, nil }

func (r *recordingRegistry) List() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.tools))
	copy(out, r.tools)
	return out
}

func (r *recordingRegistry) ListSchemas() []pkg.ToolSchema { return nil }

func (r *recordingRegistry) Execute(context.Context, string, json.RawMessage) (pkg.ToolResult, error) {
	return pkg.ToolResult{}, nil
}

func (r *recordingRegistry) RegisterHook(pkg.ToolHook) error { return nil }

func (r *recordingRegistry) UnregisterHook(string) error { return nil }

func (r *recordingRegistry) ListHookNames() []string { return nil }

// fakeToolsPlugin provides pkg.ServiceTools backed by reg.
func fakeToolsPlugin(reg pkg.ToolRegistry) core.Plugin {
	return core.Plugin{
		Name:     "fake-tools",
		Provides: []string{pkg.ServiceTools},
		Apply: func(ctx *core.Context) error {
			core.Provide(ctx, pkg.ServiceTools, reg)
			return nil
		},
	}
}

// writeBuiltinSkill creates a valid skill directory under root.
func writeBuiltinSkill(t *testing.T, root, name string) {
	t.Helper()
	dir := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "prompts"), 0o755))

	manifest := `{"name":"` + name + `","version":"1.0.0","description":"test skill"}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "skill.json"), []byte(manifest), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "prompts", "system.md"), []byte("follow the guide"), 0o644))
}

func TestPluginProvidesLoaderAndRegistersTool(t *testing.T) {
	builtin := t.TempDir()
	writeBuiltinSkill(t, builtin, "greet")

	reg := &recordingRegistry{}
	k := core.New()
	root := k.Root()
	require.NoError(t, k.Load(root, fakeToolsPlugin(reg)))
	require.NoError(t, k.Load(root, Plugin(Options{BuiltinDir: builtin})))

	loader := core.Use[*skills.SkillLoader](root, pkg.ServiceSkills)
	require.NotNil(t, loader)

	infos, err := loader.Discover()
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Equal(t, "greet", infos[0].Name)

	assert.Contains(t, reg.List(), skilltool.New(loader).Name())
}

func TestPluginWithoutToolsFails(t *testing.T) {
	k := core.New()
	err := k.Load(k.Root(), Plugin(Options{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing dependencies")
	assert.Contains(t, err.Error(), pkg.ServiceTools)
}

func TestManifestMatchesPlugin(t *testing.T) {
	m := Manifest()
	assert.Equal(t, "plugin-skill", m.Name)
	assert.Equal(t, []string{pkg.ServiceSkills}, m.Provides)
	assert.Equal(t, []string{pkg.ServiceTools}, m.Inject)
	assert.Equal(t, "builtin:skill", m.Entry)

	p := Plugin(Options{})
	assert.Equal(t, m.Name, p.Name)
	assert.Equal(t, m.Provides, p.Provides)
	assert.Equal(t, m.Inject, p.Inject)
}
