package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/supcode/supcode/pkg"
)

const defaultHookTimeout = 30 * time.Second

// HookEngine manages a middleware chain of ToolHook instances.
type HookEngine struct {
	mu    sync.RWMutex
	hooks []pkg.ToolHook
}

// NewHookEngine creates a HookEngine with no hooks.
func NewHookEngine() *HookEngine {
	return &HookEngine{
		hooks: make([]pkg.ToolHook, 0),
	}
}

// Register adds a hook to the chain.
func (e *HookEngine) Register(hook pkg.ToolHook) error {
	if hook == nil {
		return fmt.Errorf("cannot register nil hook")
	}
	if hook.Name() == "" {
		return fmt.Errorf("hook name cannot be empty")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	for _, h := range e.hooks {
		if h.Name() == hook.Name() {
			return fmt.Errorf("hook already registered: %s", hook.Name())
		}
	}

	e.hooks = append(e.hooks, hook)
	return nil
}

// Unregister removes a hook by name.
func (e *HookEngine) Unregister(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, h := range e.hooks {
		if h.Name() == name {
			e.hooks = append(e.hooks[:i], e.hooks[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("hook not found: %s", name)
}

// List returns all registered hook names.
func (e *HookEngine) List() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	names := make([]string, len(e.hooks))
	for i, h := range e.hooks {
		names[i] = h.Name()
	}
	return names
}

// ExecuteBefore runs all BeforeTool hooks in order.
// If any hook returns an error, the chain stops and returns that error.
// Each hook has a 30-second timeout.
func (e *HookEngine) ExecuteBefore(ctx context.Context, toolName string, params json.RawMessage) (json.RawMessage, error) {
	e.mu.RLock()
	hooks := make([]pkg.ToolHook, len(e.hooks))
	copy(hooks, e.hooks)
	e.mu.RUnlock()

	currentParams := params
	for _, hook := range hooks {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		hookCtx, cancel := context.WithTimeout(ctx, defaultHookTimeout)
		newParams, err := hook.BeforeTool(hookCtx, toolName, currentParams)
		cancel()

		if err != nil {
			return nil, fmt.Errorf("hook %s rejected: %w", hook.Name(), err)
		}
		if newParams != nil {
			currentParams = newParams
		}
	}
	return currentParams, nil
}

// ExecuteAfter runs all AfterTool hooks in order.
// Errors are logged but do not interrupt the chain.
// Each hook has a 30-second timeout.
func (e *HookEngine) ExecuteAfter(ctx context.Context, toolName string, params json.RawMessage, result pkg.ToolResult) {
	e.mu.RLock()
	hooks := make([]pkg.ToolHook, len(e.hooks))
	copy(hooks, e.hooks)
	e.mu.RUnlock()

	for _, hook := range hooks {
		select {
		case <-ctx.Done():
			return
		default:
		}

		hookCtx, cancel := context.WithTimeout(ctx, defaultHookTimeout)
		err := hook.AfterTool(hookCtx, toolName, params, result)
		cancel()

		if err != nil {
			log.Printf("[hooks] AfterTool hook %s error: %v", hook.Name(), err)
		}
	}
}

var _ pkg.ToolHook = (*HookWrapper)(nil)

// HookWrapper wraps a ToolHook with optional enable/disable.
type HookWrapper struct {
	hook    pkg.ToolHook
	enabled bool
}

// NewHookWrapper creates a HookWrapper.
func NewHookWrapper(hook pkg.ToolHook) *HookWrapper {
	return &HookWrapper{hook: hook, enabled: true}
}

func (w *HookWrapper) Name() string            { return w.hook.Name() }
func (w *HookWrapper) BeforeTool(ctx context.Context, toolName string, params json.RawMessage) (json.RawMessage, error) {
	if !w.enabled {
		return params, nil
	}
	return w.hook.BeforeTool(ctx, toolName, params)
}
func (w *HookWrapper) AfterTool(ctx context.Context, toolName string, params json.RawMessage, result pkg.ToolResult) error {
	if !w.enabled {
		return nil
	}
	return w.hook.AfterTool(ctx, toolName, params, result)
}
func (w *HookWrapper) Enable()  { w.enabled = true }
func (w *HookWrapper) Disable() { w.enabled = false }
