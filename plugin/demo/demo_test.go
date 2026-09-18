package demo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supcode/supcode/core"
)

func TestDemoPluginSet(t *testing.T) {
	k := core.New()
	root := k.Root()
	require.NoError(t, k.LoadPlugins(root, Plugins()))
	assert.Equal(t, []string{"demo-counter", "demo-logger"}, k.Plugins())

	logger := core.Use[*Logger](root, "logger")
	counter := core.Use[*Counter](root, "counter")
	require.NotNil(t, logger)
	require.NotNil(t, counter)

	core.Emit(context.Background(), k, "demo/add", 5)
	core.Emit(context.Background(), k, "demo/add", 2)
	assert.Equal(t, 7, counter.Value())
	assert.Contains(t, logger.Lines(), "add 5 -> 5")
	assert.Contains(t, logger.Lines(), "add 2 -> 7")

	// A payload that violates the handler's expectation is ignored by Emit.
	core.Emit(context.Background(), k, "demo/add", "bad")
	assert.Equal(t, 7, counter.Value())

	// The plugin's default config is bound to its scope and the config tree.
	cfg, ok := k.Config("demo-counter")
	require.True(t, ok)
	assert.Equal(t, map[string]any{"start": 0}, cfg)
}

func TestDemoUnloadRemovesService(t *testing.T) {
	k := core.New()
	root := k.Root()
	require.NoError(t, k.LoadPlugins(root, Plugins()))
	logger := core.Use[*Logger](root, "logger")

	require.NoError(t, k.Unload("demo-counter"))
	assert.False(t, root.Has("counter"))
	assert.True(t, root.Has("logger"))
	assert.Contains(t, logger.Lines(), "counter disposed")
}

func TestDemoMissingDependency(t *testing.T) {
	k := core.New()
	err := k.Load(k.Root(), CounterPlugin())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing dependencies")
}
