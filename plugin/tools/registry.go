package tools

import (
	"context"
	"encoding/json"

	"github.com/supcode/supcode/core"
	toolreg "github.com/supcode/supcode/internal/tools"
	"github.com/supcode/supcode/pkg"
)

// eventRegistry decorates the v1 tool registry so that every Execute call
// traverses the tool pipeline events. All other ToolRegistry methods delegate
// unchanged to the wrapped registry.
//
// Pipeline order (see pkg/pipeline.go):
//
//	tools/pre-execute  (waterfall)  permission short-circuit / param rewrite
//	tools/execute      (waterfall)  execution wrapper
//	tools/post-execute (waterfall)  result rewrite
//	tools/result       (emit)       audit / telemetry
type eventRegistry struct {
	kernel *core.Kernel
	inner  *toolreg.Registry
}

var _ pkg.ToolRegistry = (*eventRegistry)(nil)

// Register delegates to the wrapped registry.
func (r *eventRegistry) Register(tool pkg.Tool) error { return r.inner.Register(tool) }

// Unregister delegates to the wrapped registry.
func (r *eventRegistry) Unregister(name string) error { return r.inner.Unregister(name) }

// Get delegates to the wrapped registry.
func (r *eventRegistry) Get(name string) (pkg.Tool, error) { return r.inner.Get(name) }

// List delegates to the wrapped registry.
func (r *eventRegistry) List() []string { return r.inner.List() }

// ListSchemas delegates to the wrapped registry.
func (r *eventRegistry) ListSchemas() []pkg.ToolSchema { return r.inner.ListSchemas() }

// Execute routes a tool invocation through the pre-execute and execute
// waterfalls, runs the wrapped registry, then routes the result through the
// post-execute waterfall and the result event.
func (r *eventRegistry) Execute(ctx context.Context, name string, params json.RawMessage) (pkg.ToolResult, error) {
	inv := &pkg.ToolInvocation{Name: name, Params: params}

	if _, err := core.Waterfall[*pkg.ToolInvocation, *pkg.ToolInvocation](ctx, r.kernel, pkg.EventToolsPreExecute, inv); err != nil {
		return pkg.ToolResult{}, err
	}
	if _, err := core.Waterfall[*pkg.ToolInvocation, *pkg.ToolInvocation](ctx, r.kernel, pkg.EventToolsExecute, inv); err != nil {
		return pkg.ToolResult{}, err
	}

	res, err := r.inner.Execute(ctx, inv.Name, inv.Params)
	inv.Result = res
	if err == nil {
		_, _ = core.Waterfall[*pkg.ToolInvocation, *pkg.ToolInvocation](ctx, r.kernel, pkg.EventToolsPostExecute, inv)
	}
	core.Emit(ctx, r.kernel, pkg.EventToolsResult, inv)

	return inv.Result, err
}

// RegisterHook delegates to the wrapped registry.
func (r *eventRegistry) RegisterHook(hook pkg.ToolHook) error { return r.inner.RegisterHook(hook) }

// UnregisterHook delegates to the wrapped registry.
func (r *eventRegistry) UnregisterHook(hookName string) error {
	return r.inner.UnregisterHook(hookName)
}

// ListHookNames delegates to the wrapped registry.
func (r *eventRegistry) ListHookNames() []string { return r.inner.ListHookNames() }
