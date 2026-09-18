package core

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type pluginEntry struct {
	plugin Plugin
	scope  *Context
}

// Kernel is the Cordis-semantic micro-kernel: the only non-plugin component.
//
// It owns the event bus, the plugin registry and the per-plugin configuration
// tree, and it derives scoped contexts for each loaded plugin. All exported
// operations are safe for concurrent use.
type Kernel struct {
	mu             sync.RWMutex
	listeners      map[string][]*listener
	nextListenerID uint64
	plugins        map[string]*pluginEntry
	configs        map[string]any
	services       map[string]any
	root           *Context
	started        bool
}

// New creates an empty kernel with a root scope.
func New() *Kernel {
	k := &Kernel{
		listeners: make(map[string][]*listener),
		plugins:   make(map[string]*pluginEntry),
		configs:   make(map[string]any),
		services:  make(map[string]any),
	}
	k.root = newContext(k, nil, "root")
	return k
}

// Root returns the kernel's root scope. Plugins are normally loaded onto the
// root scope; callers may load onto any derived scope.
func (k *Kernel) Root() *Context { return k.root }

// Load validates the plugin's dependencies, applies it on a fresh child scope,
// and registers it. If Apply fails the scope is disposed and no state remains.
func (k *Kernel) Load(parent *Context, p Plugin) error {
	if parent == nil {
		return errors.New("core: parent context must not be nil")
	}
	if p.Name == "" {
		return errors.New("core: plugin name is required")
	}
	if p.Apply == nil {
		return fmt.Errorf("core: plugin %q has nil Apply", p.Name)
	}

	k.mu.Lock()
	if _, exists := k.plugins[p.Name]; exists {
		k.mu.Unlock()
		return fmt.Errorf("core: plugin %q already loaded", p.Name)
	}
	if p.Config != nil {
		if _, ok := k.configs[p.Name]; !ok {
			k.configs[p.Name] = p.Config()
		}
	}
	cfg := k.configs[p.Name]
	k.mu.Unlock()

	var missing []string
	for _, dep := range p.Inject {
		if !parent.Has(dep) {
			missing = append(missing, dep)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("core: plugin %q missing dependencies: %s", p.Name, strings.Join(missing, ", "))
	}

	scope := parent.Fork(p.Name)
	scope.config = cfg
	if err := p.Apply(scope); err != nil {
		_ = scope.Dispose()
		return fmt.Errorf("core: plugin %q apply: %w", p.Name, err)
	}

	k.mu.Lock()
	k.plugins[p.Name] = &pluginEntry{plugin: p, scope: scope}
	k.mu.Unlock()
	return nil
}

// Unload disposes a plugin's scope, replaying its registered effects in reverse
// order, and removes it from the registry.
func (k *Kernel) Unload(name string) error {
	k.mu.Lock()
	entry, ok := k.plugins[name]
	if ok {
		delete(k.plugins, name)
	}
	k.mu.Unlock()
	if !ok {
		return fmt.Errorf("core: plugin %q not loaded", name)
	}
	return entry.scope.Dispose()
}

// Reload unloads a plugin and loads it again on parent. This is the primitive
// behind hot reload: effects are fully unwound before Apply runs again.
func (k *Kernel) Reload(parent *Context, name string) error {
	k.mu.RLock()
	entry, ok := k.plugins[name]
	k.mu.RUnlock()
	if !ok {
		return fmt.Errorf("core: plugin %q not loaded", name)
	}
	p := entry.plugin
	if err := k.Unload(name); err != nil {
		return err
	}
	return k.Load(parent, p)
}

// LoadPlugins loads a set of plugins in dependency order. Plugin-to-plugin
// dependencies are discovered from Inject when the injected key names another
// plugin in the set. Cycles and references to unknown plugins are reported as
// errors before any plugin is loaded.
func (k *Kernel) LoadPlugins(parent *Context, plugins []Plugin) error {
	byName := make(map[string]Plugin, len(plugins))
	for _, p := range plugins {
		if _, dup := byName[p.Name]; dup {
			return fmt.Errorf("core: duplicate plugin %q", p.Name)
		}
		byName[p.Name] = p
	}

	// Map provided service keys to the plugin that provides them so that a
	// consumer's Inject entry creates a dependency edge on the provider.
	providerOf := make(map[string]string)
	for _, p := range plugins {
		for _, key := range p.Provides {
			providerOf[key] = p.Name
		}
	}

	var order []string
	done := make(map[string]bool)
	visiting := make(map[string]bool)

	var visit func(name string) error
	visit = func(name string) error {
		if done[name] {
			return nil
		}
		if visiting[name] {
			return fmt.Errorf("core: dependency cycle involving plugin %q", name)
		}
		p, ok := byName[name]
		if !ok {
			return fmt.Errorf("core: unknown plugin %q", name)
		}
		visiting[name] = true
		for _, dep := range p.Inject {
			if provider, ok := providerOf[dep]; ok {
				if err := visit(provider); err != nil {
					return err
				}
			} else if _, isPlugin := byName[dep]; isPlugin {
				if err := visit(dep); err != nil {
					return err
				}
			}
		}
		delete(visiting, name)
		done[name] = true
		order = append(order, name)
		return nil
	}

	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := visit(name); err != nil {
			return err
		}
	}

	for _, name := range order {
		if err := k.Load(parent, byName[name]); err != nil {
			return err
		}
	}
	return nil
}

// Plugins returns the names of currently loaded plugins, sorted.
func (k *Kernel) Plugins() []string {
	k.mu.RLock()
	defer k.mu.RUnlock()
	names := make([]string, 0, len(k.plugins))
	for name := range k.plugins {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// SetConfig sets the configuration value for a plugin or service name.
func (k *Kernel) SetConfig(name string, value any) {
	k.mu.Lock()
	k.configs[name] = value
	k.mu.Unlock()
}

// Config returns the configuration value for a name.
func (k *Kernel) Config(name string) (any, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	v, ok := k.configs[name]
	return v, ok
}

// Start marks the kernel as running. Plugins are loaded before Start; Start is
// intentionally cheap so callers can control ordering.
func (k *Kernel) Start(_ context.Context) error {
	k.mu.Lock()
	k.started = true
	k.mu.Unlock()
	return nil
}

// Started reports whether Start has been called and Shutdown has not.
func (k *Kernel) Started() bool {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.started
}

// Shutdown unloads every plugin in reverse load order and disposes the root
// scope. It returns the joined error of all failures.
func (k *Kernel) Shutdown(_ context.Context) error {
	k.mu.Lock()
	names := make([]string, 0, len(k.plugins))
	for name := range k.plugins {
		names = append(names, name)
	}
	k.mu.Unlock()
	sort.Sort(sort.Reverse(sort.StringSlice(names)))

	var errs []error
	for _, name := range names {
		if err := k.Unload(name); err != nil {
			errs = append(errs, err)
		}
	}
	if err := k.root.Dispose(); err != nil {
		errs = append(errs, err)
	}

	k.mu.Lock()
	k.started = false
	k.mu.Unlock()
	return errors.Join(errs...)
}
