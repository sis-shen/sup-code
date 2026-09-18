package core

import "fmt"

// Disposer reverses a side effect registered on a Context.
type Disposer func() error

// Provide registers a service value under key and returns a Disposer that
// restores the previous state. It is the typed form of Context.Provide.
func Provide[T any](c *Context, key string, impl T) Disposer {
	return c.Provide(key, impl)
}

// Use resolves a service by key, panicking if it is missing or has the wrong
// type. Callers are expected to declare the dependency via Plugin.Inject so
// that the loader can validate it before the plugin starts.
func Use[T any](c *Context, key string) T {
	v, ok := c.lookup(key)
	if !ok {
		panic(fmt.Sprintf("core: service %q is not provided", key))
	}
	typed, ok := v.(T)
	if !ok {
		panic(fmt.Sprintf("core: service %q has type %T, want %T", key, v, *new(T)))
	}
	return typed
}

// MaybeUse resolves a service by key without panicking. The boolean reports
// whether the service was found and had the expected type.
func MaybeUse[T any](c *Context, key string) (T, bool) {
	v, ok := c.lookup(key)
	if !ok {
		var zero T
		return zero, false
	}
	typed, ok := v.(T)
	return typed, ok
}
