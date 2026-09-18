package permission

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/core"
	perm "github.com/supcode/supcode/internal/permission"
	"github.com/supcode/supcode/pkg"
)

func loadKernel(t *testing.T, opts Options) *core.Kernel {
	t.Helper()
	k := core.New()
	require.NoError(t, k.LoadPlugins(k.Root(), []core.Plugin{Plugin(opts)}))
	return k
}

func TestPluginProvidesPermissionEngine(t *testing.T) {
	k := loadKernel(t, Options{})

	eng := core.Use[pkg.PermissionEngine](k.Root(), pkg.ServicePermission)
	require.NotNil(t, eng)
	assert.Equal(t, []string{"plugin-permission"}, k.Plugins())
}

func TestPluginDenyShortCircuits(t *testing.T) {
	eng := perm.NewWithRules([]pkg.PermissionRule{
		{ID: "deny-bash", Scope: "tool", Pattern: "bash", Decision: pkg.DecisionDeny, Priority: 1},
	})
	k := loadKernel(t, Options{Engine: eng})

	inv := &pkg.ToolInvocation{Name: "bash", Params: []byte(`{}`)}
	got, err := core.Waterfall[*pkg.ToolInvocation, *pkg.ToolInvocation](
		context.Background(), k, pkg.EventToolsPreExecute, inv)

	require.Error(t, err)
	assert.Nil(t, got)

	var denied *pkg.ErrPermissionDenied
	require.ErrorAs(t, err, &denied)
	assert.Equal(t, pkg.DecisionDeny, inv.Decision)
}

func TestPluginAllowSetsDecision(t *testing.T) {
	k := loadKernel(t, Options{})

	inv := &pkg.ToolInvocation{Name: "echo", Params: []byte(`{}`)}
	got, err := core.Waterfall[*pkg.ToolInvocation, *pkg.ToolInvocation](
		context.Background(), k, pkg.EventToolsPreExecute, inv)

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Same(t, inv, got)
	assert.Equal(t, pkg.DecisionAllow, got.Decision)
}

func TestPluginUsesCustomRules(t *testing.T) {
	k := loadKernel(t, Options{Rules: []pkg.PermissionRule{
		{ID: "deny-git", Scope: "tool", Pattern: "git", Decision: pkg.DecisionDeny, Priority: 1},
	}})

	inv := &pkg.ToolInvocation{Name: "git", Params: []byte(`{}`)}
	_, err := core.Waterfall[*pkg.ToolInvocation, *pkg.ToolInvocation](
		context.Background(), k, pkg.EventToolsPreExecute, inv)

	var denied *pkg.ErrPermissionDenied
	require.ErrorAs(t, err, &denied)
}

func TestManifestMatchesPlugin(t *testing.T) {
	m := Manifest()
	assert.Equal(t, "plugin-permission", m.Name)
	assert.Equal(t, "2.0.0", m.Version)
	assert.Equal(t, []string{"permission"}, m.Provides)
	assert.Equal(t, "builtin:permission", m.Entry)
}
