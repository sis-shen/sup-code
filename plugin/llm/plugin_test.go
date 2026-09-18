package llm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/core"
	"github.com/supcode/supcode/pkg"
)

type fakeClient struct {
	name string
}

func (f *fakeClient) Chat(context.Context, string, []pkg.Message, []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
	return nil, nil
}

func (f *fakeClient) Models(context.Context) ([]pkg.ModelInfo, error) {
	return nil, nil
}

func (f *fakeClient) ProviderName() string { return f.name }

type fakeConfig struct {
	values map[string]any
}

func (c *fakeConfig) Load() error                     { return nil }
func (c *fakeConfig) Get(key string) any              { return c.values[key] }
func (c *fakeConfig) Set(key string, value any) error { c.values[key] = value; return nil }
func (c *fakeConfig) Save() error                     { return nil }
func (c *fakeConfig) AllSettings() map[string]any     { return c.values }
func (c *fakeConfig) UnmarshalKey(string, any) error  { return nil }

func (c *fakeConfig) GetString(key string) string {
	s, _ := c.values[key].(string)
	return s
}

func (c *fakeConfig) GetInt(key string) int {
	n, _ := c.values[key].(int)
	return n
}

func (c *fakeConfig) GetBool(key string) bool {
	b, _ := c.values[key].(bool)
	return b
}

func TestPluginUsesInjectedClient(t *testing.T) {
	k := core.New()
	root := k.Root()
	fake := &fakeClient{name: "fake"}

	require.NoError(t, k.Load(root, Plugin(Options{Client: fake})))

	got := core.Use[pkg.LLMClient](root, pkg.ServiceLLM)
	assert.Equal(t, fake, got)
	assert.Equal(t, []string{pluginName}, k.Plugins())
}

func TestPluginBuildsClientFromProvider(t *testing.T) {
	k := core.New()
	root := k.Root()

	require.NoError(t, k.Load(root, Plugin(Options{Provider: "openai", APIKey: "x"})))

	got := core.Use[pkg.LLMClient](root, pkg.ServiceLLM)
	require.NotNil(t, got)
	assert.Equal(t, "openai", got.ProviderName())
}

func TestPluginUnknownProviderFails(t *testing.T) {
	k := core.New()
	err := k.Load(k.Root(), Plugin(Options{Provider: "bogus"}))
	require.Error(t, err)
	assert.False(t, k.Root().Has(pkg.ServiceLLM))
}

func TestPluginReadsOptionalConfig(t *testing.T) {
	k := core.New()
	root := k.Root()
	core.Provide[pkg.Config](root, pkg.ServiceConfig, &fakeConfig{values: map[string]any{
		configKeyProvider: "anthropic",
		configKeyAPIKey:   "from-config",
	}})

	require.NoError(t, k.Load(root, Plugin(Options{})))

	got := core.Use[pkg.LLMClient](root, pkg.ServiceLLM)
	require.NotNil(t, got)
	assert.Equal(t, "anthropic", got.ProviderName())
}

func TestPluginOptionsOverrideConfig(t *testing.T) {
	k := core.New()
	root := k.Root()
	core.Provide[pkg.Config](root, pkg.ServiceConfig, &fakeConfig{values: map[string]any{
		configKeyProvider: "anthropic",
	}})

	require.NoError(t, k.Load(root, Plugin(Options{Provider: "openai", APIKey: "x"})))

	got := core.Use[pkg.LLMClient](root, pkg.ServiceLLM)
	require.NotNil(t, got)
	assert.Equal(t, "openai", got.ProviderName())
}

func TestManifest(t *testing.T) {
	m := Manifest()
	assert.Equal(t, pluginName, m.Name)
	assert.Equal(t, pluginVersion, m.Version)
	assert.Equal(t, []string{pkg.ServiceLLM}, m.Provides)
	assert.Empty(t, m.Inject)
	assert.Equal(t, "plugin/llm", m.Entry)
}

func TestAvailableProviders(t *testing.T) {
	assert.ElementsMatch(t, []string{"openai", "anthropic", "deepseek", "ollama"}, AvailableProviders())
}
