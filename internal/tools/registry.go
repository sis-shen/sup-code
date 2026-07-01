package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/supcode/supcode/pkg"
)

// Registry implements pkg.ToolRegistry.
type Registry struct {
	mu      sync.RWMutex
	tools   map[string]pkg.Tool
	hooks   []pkg.ToolHook
	permEng pkg.PermissionEngine
}

// NewRegistry creates a new ToolRegistry with the given PermissionEngine.
func NewRegistry(permEng pkg.PermissionEngine) *Registry {
	return &Registry{
		tools:   make(map[string]pkg.Tool),
		permEng: permEng,
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(tool pkg.Tool) error {
	if tool == nil {
		return fmt.Errorf("cannot register nil tool")
	}
	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool already registered: %s", name)
	}

	r.tools[name] = tool
	return nil
}

// Unregister removes a tool from the registry.
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return &pkg.ErrToolNotFound{ToolName: name}
	}

	delete(r.tools, name)
	return nil
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (pkg.Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, &pkg.ErrToolNotFound{ToolName: name}
	}
	return tool, nil
}

// List returns all registered tool names, sorted alphabetically.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ListSchemas returns the schemas of all registered tools.
func (r *Registry) ListSchemas() []pkg.ToolSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schemas := make([]pkg.ToolSchema, 0, len(r.tools))
	for _, tool := range r.tools {
		schemas = append(schemas, tool.Schema())
	}
	return schemas
}

// Execute runs a tool by name after checking permissions and running hooks.
func (r *Registry) Execute(ctx context.Context, name string, params json.RawMessage) (pkg.ToolResult, error) {
	r.mu.RLock()
	tool, exists := r.tools[name]
	if !exists {
		r.mu.RUnlock()
		return pkg.ToolResult{Success: false, Error: fmt.Sprintf("tool not found: %s", name)}, &pkg.ErrToolNotFound{ToolName: name}
	}

	hooks := make([]pkg.ToolHook, len(r.hooks))
	copy(hooks, r.hooks)
	r.mu.RUnlock()

	if r.permEng != nil {
		action := pkg.Action{
			Type:     "tool",
			ToolName: name,
			Params:   string(params),
		}
		decision, err := r.permEng.Check(ctx, action)
		if err != nil {
			return pkg.ToolResult{Success: false, Error: fmt.Sprintf("permission error: %v", err)}, err
		}
		if decision == pkg.DecisionDeny {
			return pkg.ToolResult{Success: false, Error: fmt.Sprintf("permission denied: %s", name)}, &pkg.ErrPermissionDenied{
				Action: action,
				Rule:   "deny",
			}
		}
	}

	currentParams := params
	for _, hook := range hooks {
		newParams, err := hook.BeforeTool(ctx, name, currentParams)
		if err != nil {
			return pkg.ToolResult{Success: false, Error: fmt.Sprintf("hook %s rejected: %v", hook.Name(), err)}, err
		}
		if newParams != nil {
			currentParams = newParams
		}
	}

	result, err := tool.Execute(ctx, currentParams)
	if err != nil {
		for _, hook := range hooks {
			hook.AfterTool(ctx, name, currentParams, result)
		}
		return result, err
	}

	for _, hook := range hooks {
		hook.AfterTool(ctx, name, currentParams, result)
	}

	return result, nil
}

// RegisterHook adds a global hook.
func (r *Registry) RegisterHook(hook pkg.ToolHook) error {
	if hook == nil {
		return fmt.Errorf("cannot register nil hook")
	}
	if hook.Name() == "" {
		return fmt.Errorf("hook name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, h := range r.hooks {
		if h.Name() == hook.Name() {
			return fmt.Errorf("hook already registered: %s", hook.Name())
		}
	}

	r.hooks = append(r.hooks, hook)
	return nil
}

// UnregisterHook removes a hook by name.
func (r *Registry) UnregisterHook(hookName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, h := range r.hooks {
		if h.Name() == hookName {
			r.hooks = append(r.hooks[:i], r.hooks[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("hook not found: %s", hookName)
}

var _ pkg.ToolRegistry = (*Registry)(nil)
