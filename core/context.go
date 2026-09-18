package core

import (
	"errors"
	"fmt"
	"sync"
)

// Context is the Cordis service container and lifecycle scope.
//
// A Context holds a set of services addressed by stable string keys, a list of
// registered reversible side effects, and (optionally) a parent scope. Service
// lookups walk the parent chain, so child scopes inherit from their parent
// unless they are created with Isolate.
//
// Contexts are safe for concurrent use.
type Context struct {
	kernel   *Kernel
	parent   *Context
	name     string
	isolated bool
	config   any

	mu        sync.RWMutex
	services  map[string]any
	disposers []Disposer
	disposed  bool
}

func newContext(k *Kernel, parent *Context, name string) *Context {
	return &Context{
		kernel:   k,
		parent:   parent,
		name:     name,
		services: make(map[string]any),
	}
}

// Kernel returns the owning kernel.
func (c *Context) Kernel() *Kernel { return c.kernel }

// Name returns the scope name (the plugin name for plugin scopes).
func (c *Context) Name() string { return c.name }

// Parent returns the parent scope, or nil for the root scope.
func (c *Context) Parent() *Context { return c.parent }

// Config returns the configuration value bound to this scope. For plugin
// scopes this is the plugin's config tree; it is nil for plain scopes.
func (c *Context) Config() any { return c.config }

// Fork derives a child scope that inherits services from its parent. A fork of
// an isolated scope remains isolated.
func (c *Context) Fork(name string) *Context {
	child := newContext(c.kernel, c, name)
	child.isolated = c.isolated
	return child
}

// Isolate derives a child scope that does not resolve services from the
// kernel or its parent. It is used to give a session or sub-agent a private
// service set.
func (c *Context) Isolate(name string) *Context {
	child := newContext(c.kernel, c, name)
	child.isolated = true
	return child
}

// Provide registers a service value under key and returns a Disposer that
// restores the previous state. Registration is reversible: unloading the owning
// scope removes the service.
//
// Non-isolated scopes register into the kernel-wide registry so that sibling
// plugins can resolve each other's services. Isolated scopes register locally.
func (c *Context) Provide(key string, impl any) Disposer {
	if key == "" {
		panic("core: service key must not be empty")
	}

	var (
		mu     *sync.RWMutex
		bucket map[string]any
	)
	if c.isolated {
		c.mu.Lock()
		bucket = c.services
		prev, had := bucket[key]
		bucket[key] = impl
		c.mu.Unlock()

		var once sync.Once
		return func() error {
			once.Do(func() {
				c.mu.Lock()
				defer c.mu.Unlock()
				if had {
					bucket[key] = prev
				} else {
					delete(bucket, key)
				}
			})
			return nil
		}
	}

	k := c.kernel
	k.mu.Lock()
	bucket = k.services
	mu = &k.mu
	prev, had := bucket[key]
	bucket[key] = impl
	k.mu.Unlock()

	var once sync.Once
	restore := func() error {
		once.Do(func() {
			mu.Lock()
			defer mu.Unlock()
			if had {
				bucket[key] = prev
			} else {
				delete(bucket, key)
			}
		})
		return nil
	}
	c.registerDisposer(restore)
	return restore
}

// lookup resolves a service: local scope chain first (respecting isolation
// barriers), then the kernel-wide registry for non-isolated scopes.
func (c *Context) lookup(key string) (any, bool) {
	for ctx := c; ctx != nil; ctx = ctx.parent {
		ctx.mu.RLock()
		v, ok := ctx.services[key]
		ctx.mu.RUnlock()
		if ok {
			return v, true
		}
		if ctx.isolated {
			return nil, false
		}
	}
	c.kernel.mu.RLock()
	v, ok := c.kernel.services[key]
	c.kernel.mu.RUnlock()
	return v, ok
}

// registerDisposer appends d to the scope's disposer list, or runs it
// immediately if the scope is already disposed.
func (c *Context) registerDisposer(d Disposer) {
	c.mu.Lock()
	if c.disposed {
		c.mu.Unlock()
		_ = d()
		return
	}
	c.disposers = append(c.disposers, d)
	c.mu.Unlock()
}

// Has reports whether a service is visible from this scope.
func (c *Context) Has(key string) bool {
	_, ok := c.lookup(key)
	return ok
}

// Effect registers a reversible side effect. fn is invoked immediately and
// must return a Disposer that undoes it; the disposer is replayed (in reverse
// order) when the scope is disposed. A nil disposer means "nothing to undo".
func (c *Context) Effect(fn func() (Disposer, error)) error {
	if fn == nil {
		return fmt.Errorf("core: effect function must not be nil")
	}
	c.mu.Lock()
	if c.disposed {
		c.mu.Unlock()
		return fmt.Errorf("core: context %q already disposed", c.name)
	}
	c.mu.Unlock()

	d, err := fn()
	if err != nil {
		return err
	}
	if d == nil {
		return nil
	}
	c.registerDisposer(d)
	return nil
}

// On registers an event listener scoped to this context. The listener is
// automatically removed when the scope is disposed.
func (c *Context) On(event string, h Handler, opts ...EventOption) Disposer {
	if h == nil {
		panic("core: event handler must not be nil")
	}
	remove := c.kernel.addListener(event, h, opts...)
	wrapped := func() error {
		remove()
		return nil
	}
	c.mu.Lock()
	if c.disposed {
		c.mu.Unlock()
		remove()
		return wrapped
	}
	c.disposers = append(c.disposers, wrapped)
	c.mu.Unlock()
	return wrapped
}

// Disposed reports whether the scope has been disposed.
func (c *Context) Disposed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.disposed
}

// Dispose replays all registered disposers in reverse (LIFO) order. It is
// idempotent: subsequent calls are no-ops.
func (c *Context) Dispose() error {
	c.mu.Lock()
	if c.disposed {
		c.mu.Unlock()
		return nil
	}
	c.disposed = true
	pending := c.disposers
	c.disposers = nil
	c.mu.Unlock()

	var errs []error
	for i := len(pending) - 1; i >= 0; i-- {
		if err := pending[i](); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
