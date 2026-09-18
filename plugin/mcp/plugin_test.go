package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/core"
	mcpclient "github.com/supcode/supcode/internal/mcp"
	"github.com/supcode/supcode/pkg"
)

// fakeRegistry is a minimal pkg.ToolRegistry. plugin-mcp only requires the
// tools service to be present, so every method is inert.
type fakeRegistry struct{}

func (fakeRegistry) Register(pkg.Tool) error { return nil }

func (fakeRegistry) Unregister(string) error { return nil }

func (fakeRegistry) Get(string) (pkg.Tool, error) { return nil, errors.New("not found") }

func (fakeRegistry) List() []string { return nil }

func (fakeRegistry) ListSchemas() []pkg.ToolSchema { return nil }

func (fakeRegistry) Execute(context.Context, string, json.RawMessage) (pkg.ToolResult, error) {
	return pkg.ToolResult{}, nil
}

func (fakeRegistry) RegisterHook(pkg.ToolHook) error { return nil }

func (fakeRegistry) UnregisterHook(string) error { return nil }

func (fakeRegistry) ListHookNames() []string { return nil }

// fakeToolsPlugin provides pkg.ServiceTools with the minimal fake registry,
// keeping the tests independent of plugin-tools.
func fakeToolsPlugin() core.Plugin {
	return core.Plugin{
		Name:     "fake-tools",
		Provides: []string{pkg.ServiceTools},
		Apply: func(ctx *core.Context) error {
			core.Provide(ctx, pkg.ServiceTools, fakeRegistry{})
			return nil
		},
	}
}

func TestPluginProvidesMCPClient(t *testing.T) {
	k := core.New()
	root := k.Root()
	require.NoError(t, k.Load(root, fakeToolsPlugin()))
	require.NoError(t, k.Load(root, Plugin(Options{})))

	client := core.Use[pkg.MCPClient](root, pkg.ServiceMCP)
	require.NotNil(t, client)
	assert.Empty(t, client.ListServers())
}

func TestPluginUnloadDisposesClient(t *testing.T) {
	k := core.New()
	root := k.Root()
	require.NoError(t, k.Load(root, fakeToolsPlugin()))
	require.NoError(t, k.Load(root, Plugin(Options{})))

	client := core.Use[pkg.MCPClient](root, pkg.ServiceMCP)
	require.NotNil(t, client)

	// Unload replays the Apply effect, which closes the client and removes the
	// service. Close is idempotent, so a second explicit call must stay
	// panic-free and error-free.
	require.NoError(t, k.Unload(pluginName))
	assert.False(t, root.Has(pkg.ServiceMCP))
	assert.NotPanics(t, func() { assert.NoError(t, client.Close()) })
	assert.Empty(t, client.ListServers())
}

func TestPluginWithoutToolsFails(t *testing.T) {
	k := core.New()
	err := k.Load(k.Root(), Plugin(Options{}))
	require.Error(t, err)
	assert.ErrorContains(t, err, "missing dependencies")
	assert.ErrorContains(t, err, pkg.ServiceTools)
	assert.False(t, k.Root().Has(pkg.ServiceMCP))
}

func TestManifestMatchesPlugin(t *testing.T) {
	m := Manifest()
	assert.Equal(t, pluginName, m.Name)
	assert.Equal(t, pluginVersion, m.Version)
	assert.Equal(t, []string{pkg.ServiceTools}, m.Inject)
	assert.Equal(t, []string{pkg.ServiceMCP}, m.Provides)
	assert.Equal(t, "builtin:mcp", m.Entry)

	p := Plugin(Options{})
	assert.Equal(t, m.Name, p.Name)
	assert.Equal(t, m.Inject, p.Inject)
	assert.Equal(t, m.Provides, p.Provides)
}

func TestBridgeWiresClientAndRegistry(t *testing.T) {
	b := Bridge(mcpclient.NewClient(), fakeRegistry{})
	require.NotNil(t, b)
	assert.NoError(t, b.RegisterAllTools(context.Background()))
}
