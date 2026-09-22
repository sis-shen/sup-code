package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/core"
	intconfig "github.com/supcode/supcode/internal/config"
	"github.com/supcode/supcode/pkg"
)

// isolate gives config discovery a clean environment: a temporary home so
// ~/.supcode/config.yaml is not read, a scrubbed SUPCODE_* environment so
// AutomaticEnv cannot inject values, and a deep temporary working directory so
// the project-level lookup (which walks up at most ten levels) cannot reach a
// developer's real home and pick up its .supcode/config.yaml.
func isolate(t *testing.T) {
	t.Helper()

	scrubSupcodeEnv(t)

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	dir := t.TempDir()
	for i := 0; i < 12; i++ {
		dir = filepath.Join(dir, "d")
	}
	require.NoError(t, os.MkdirAll(dir, 0755))
	t.Chdir(dir)
}

// scrubSupcodeEnv unsets every SUPCODE_* variable for the duration of the test
// and restores the original values on cleanup. t.Setenv cannot express "unset",
// which is required because viper's AutomaticEnv treats a set-but-empty
// variable as an override.
func scrubSupcodeEnv(t *testing.T) {
	t.Helper()

	for _, kv := range os.Environ() {
		key, _, ok := strings.Cut(kv, "=")
		if !ok || !strings.HasPrefix(key, "SUPCODE_") {
			continue
		}
		value := os.Getenv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
		t.Cleanup(func() { _ = os.Setenv(key, value) })
	}
}

func TestPluginLoadsExplicitFile(t *testing.T) {
	isolate(t)

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("llm:\n  provider: deepseek\n"), 0644))

	k := core.New()
	require.NoError(t, k.Load(k.Root(), Plugin(Options{Path: path})))

	cfg := core.Use[pkg.Config](k.Root(), pkg.ServiceConfig)
	assert.Equal(t, "deepseek", cfg.GetString("llm.provider"))
	assert.Equal(t, []string{pluginName}, k.Plugins())
}

func TestPluginFallsBackToDefaults(t *testing.T) {
	isolate(t)

	k := core.New()
	require.NoError(t, k.Load(k.Root(), Plugin(Options{})))

	cfg := core.Use[pkg.Config](k.Root(), pkg.ServiceConfig)
	want := intconfig.DefaultConfig[intconfig.ConfigKeyLLMProvider]
	assert.Equal(t, want, cfg.GetString("llm.provider"))
	assert.Equal(t, "openai", cfg.GetString("llm.provider"))
}

// TestPluginLoadsWithoutAPIKey is a regression guard for the v1 behavior that
// config loads offline: key validation is deferred to the agent layer.
func TestPluginLoadsWithoutAPIKey(t *testing.T) {
	isolate(t)

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("llm:\n  provider: deepseek\n"), 0644))

	k := core.New()
	require.NoError(t, k.Load(k.Root(), Plugin(Options{Path: path})))

	cfg := core.Use[pkg.Config](k.Root(), pkg.ServiceConfig)
	assert.Empty(t, cfg.GetString("llm.api_key"))
}

func TestManifestMatchesPlugin(t *testing.T) {
	m := Manifest()
	assert.Equal(t, pluginName, m.Name)
	assert.Equal(t, []string{"config"}, m.Provides)
	assert.Equal(t, "builtin:config", m.Entry)

	p := Plugin(Options{})
	assert.Equal(t, m.Name, p.Name)
	assert.Equal(t, m.Provides, p.Provides)
	assert.Empty(t, p.Inject)
}
