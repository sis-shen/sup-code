package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/core"
	"github.com/supcode/supcode/pkg"
	permissionplugin "github.com/supcode/supcode/plugin/permission"
	skillplugin "github.com/supcode/supcode/plugin/skill"
	toolsplugin "github.com/supcode/supcode/plugin/tools"
)

// TestPluginPool_BuiltinToolsAndSkillTool loads the permission, tools and skill
// leaf plugins into one kernel and verifies the 6 builtin tools and the skill
// tool share a single pool, with execution flowing through the tool pipeline.
func TestPluginPool_BuiltinToolsAndSkillTool(t *testing.T) {
	k := core.New()
	root := k.Root()

	builtinDir := t.TempDir()
	writeDemoSkill(t, builtinDir, "demo-skill")

	// Passed in a non-topological order on purpose: LoadPlugins must order
	// permission -> tools -> skill from Provides/Inject.
	require.NoError(t, k.LoadPlugins(root, []core.Plugin{
		toolsplugin.Plugin(toolsplugin.Options{}),
		permissionplugin.Plugin(permissionplugin.Options{}),
		skillplugin.Plugin(skillplugin.Options{BuiltinDir: builtinDir}),
	}))

	reg := core.Use[pkg.ToolRegistry](root, pkg.ServiceTools)
	names := reg.List()
	for _, want := range []string{"bash", "read_file", "write_file", "edit_file", "glob", "grep", "skill"} {
		assert.Contains(t, names, want)
	}
	assert.Len(t, reg.ListSchemas(), 7)

	// The skill tool executes through the shared registry / event pipeline.
	res, err := reg.Execute(context.Background(), "skill", json.RawMessage(`{"action":"list"}`))
	require.NoError(t, err)
	assert.True(t, res.Success)

	require.NoError(t, k.Shutdown(context.Background()))
	assert.Empty(t, k.Plugins())
}

func writeDemoSkill(t *testing.T, dir, name string) {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Join(skillDir, "prompts"), 0o755))
	manifest := `{"name":"` + name + `","version":"1.0.0","description":"demo"}`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "skill.json"), []byte(manifest), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "prompts", "system.md"), []byte("demo prompt"), 0o644))
}
