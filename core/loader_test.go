package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadProvideAndUnload(t *testing.T) {
	k := New()
	p := Plugin{
		Name:     "provider",
		Provides: []string{"svc"},
		Apply: func(c *Context) error {
			Provide(c, "svc", "value")
			return nil
		},
	}

	require.NoError(t, k.Load(k.Root(), p))
	assert.Equal(t, []string{"provider"}, k.Plugins())
	assert.Equal(t, "value", Use[string](k.Root(), "svc"))

	require.NoError(t, k.Unload("provider"))
	assert.Empty(t, k.Plugins())
	assert.False(t, k.Root().Has("svc"))
}

func TestLoadBindsConfig(t *testing.T) {
	k := New()
	defaultCfg := map[string]any{"port": 8080}
	var seen any
	p := Plugin{
		Name:   "configurable",
		Config: func() any { return defaultCfg },
		Apply: func(c *Context) error {
			seen = c.Config()
			return nil
		},
	}

	require.NoError(t, k.Load(k.Root(), p))
	assert.Equal(t, defaultCfg, seen)

	got, ok := k.Config("configurable")
	require.True(t, ok)
	assert.Equal(t, defaultCfg, got)
}

func TestLoadErrors(t *testing.T) {
	k := New()
	root := k.Root()

	assert.Error(t, k.Load(nil, Plugin{Name: "x", Apply: func(*Context) error { return nil }}))
	assert.Error(t, k.Load(root, Plugin{Apply: func(*Context) error { return nil }}))
	assert.Error(t, k.Load(root, Plugin{Name: "x"}))

	injector := Plugin{
		Name:     "consumer",
		Inject:   []string{"missing-a", "missing-b"},
		Provides: []string{"consumer"},
		Apply:    func(*Context) error { return nil },
	}
	err := k.Load(root, injector)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing dependencies")
	assert.Contains(t, err.Error(), "missing-a")
	assert.Contains(t, err.Error(), "missing-b")

	ok := Plugin{Name: "ok", Apply: func(*Context) error { return nil }}
	require.NoError(t, k.Load(root, ok))
	assert.Error(t, k.Load(root, ok))
}

func TestLoadApplyErrorDisposesScope(t *testing.T) {
	k := New()
	disposed := false
	p := Plugin{
		Name: "failing",
		Apply: func(c *Context) error {
			require.NoError(t, c.Effect(func() (Disposer, error) {
				return func() error { disposed = true; return nil }, nil
			}))
			return assert.AnError
		},
	}

	err := k.Load(k.Root(), p)
	require.Error(t, err)
	assert.True(t, disposed)
	assert.Empty(t, k.Plugins())

	// A failed load must not block a retry with the same name.
	require.NoError(t, k.Load(k.Root(), Plugin{Name: "failing", Apply: func(*Context) error { return nil }}))
}

func TestUnloadUnknown(t *testing.T) {
	assert.Error(t, New().Unload("nope"))
}

func TestReloadReplaysEffects(t *testing.T) {
	k := New()
	applies, disposes := 0, 0
	p := Plugin{
		Name:     "reloadable",
		Provides: []string{"reloadable"},
		Apply: func(c *Context) error {
			applies++
			Provide(c, "reloadable-value", applies)
			return c.Effect(func() (Disposer, error) {
				return func() error { disposes++; return nil }, nil
			})
		},
	}

	require.NoError(t, k.Load(k.Root(), p))
	require.NoError(t, k.Reload(k.Root(), "reloadable"))
	assert.Equal(t, 2, applies)
	assert.Equal(t, 1, disposes)
	assert.Equal(t, 2, Use[int](k.Root(), "reloadable-value"))

	assert.Error(t, k.Reload(k.Root(), "unknown"))
}

func TestLoadPluginsTopology(t *testing.T) {
	k := New()
	var order []string
	makePlugin := func(name string, inject, provides []string) Plugin {
		return Plugin{
			Name:     name,
			Inject:   inject,
			Provides: provides,
			Apply: func(c *Context) error {
				order = append(order, name)
				for _, key := range provides {
					Provide(c, key, name)
				}
				return nil
			},
		}
	}

	a := makePlugin("a", nil, []string{"a"})
	b := makePlugin("b", []string{"a"}, []string{"b"})
	c := makePlugin("c", []string{"b"}, []string{"c"})

	// Deliberately pass consumers before providers.
	require.NoError(t, k.LoadPlugins(k.Root(), []Plugin{c, b, a}))
	assert.Equal(t, []string{"a", "b", "c"}, order)
}

func TestLoadPluginsDirectNameDependency(t *testing.T) {
	k := New()
	var order []string
	provider := Plugin{Name: "provider", Apply: func(c *Context) error {
		order = append(order, "provider")
		Provide(c, "provider", "provided")
		return nil
	}}
	// Inject the provider's name directly (not declared in Provides), which
	// exercises the by-name dependency fallback.
	consumer := Plugin{Name: "consumer", Inject: []string{"provider"}, Provides: []string{"consumer"}, Apply: func(c *Context) error {
		order = append(order, "consumer")
		Provide(c, "consumer", "provided")
		return nil
	}}

	require.NoError(t, k.LoadPlugins(k.Root(), []Plugin{consumer, provider}))
	assert.Equal(t, []string{"provider", "consumer"}, order)
}

func TestLoadPluginsCycle(t *testing.T) {
	k := New()
	a := Plugin{Name: "a", Inject: []string{"b"}, Provides: []string{"a"}, Apply: func(*Context) error { return nil }}
	b := Plugin{Name: "b", Inject: []string{"a"}, Provides: []string{"b"}, Apply: func(*Context) error { return nil }}

	err := k.LoadPlugins(k.Root(), []Plugin{a, b})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cycle")
}

func TestLoadPluginsDuplicate(t *testing.T) {
	k := New()
	p := Plugin{Name: "dup", Apply: func(*Context) error { return nil }}
	err := k.LoadPlugins(k.Root(), []Plugin{p, p})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

func TestLoadPluginsPropagatesLoadError(t *testing.T) {
	k := New()
	p := Plugin{Name: "broken", Apply: func(*Context) error { return assert.AnError }}
	assert.Error(t, k.LoadPlugins(k.Root(), []Plugin{p}))
}

func TestConfigSetAndGet(t *testing.T) {
	k := New()
	_, ok := k.Config("x")
	assert.False(t, ok)

	k.SetConfig("x", 1)
	got, ok := k.Config("x")
	assert.True(t, ok)
	assert.Equal(t, 1, got)
}

func TestStartAndShutdown(t *testing.T) {
	k := New()
	assert.False(t, k.Started())

	require.NoError(t, k.Start(context.Background()))
	assert.True(t, k.Started())

	var unloaded []string
	for _, name := range []string{"a", "b"} {
		name := name
		p := Plugin{
			Name: name,
			Apply: func(c *Context) error {
				return c.Effect(func() (Disposer, error) {
					return func() error { unloaded = append(unloaded, name); return nil }, nil
				})
			},
		}
		require.NoError(t, k.Load(k.Root(), p))
	}

	require.NoError(t, k.Shutdown(context.Background()))
	assert.False(t, k.Started())
	assert.Empty(t, k.Plugins())
	assert.ElementsMatch(t, []string{"a", "b"}, unloaded)
	assert.True(t, k.Root().Disposed())
}
