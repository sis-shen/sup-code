package tools

import (
	"context"
	"encoding/json"

	"github.com/supcode/supcode/pkg"
)

// MockToolRegistry implements pkg.ToolRegistry for testing.
type MockToolRegistry struct {
	RegisterFunc    func(tool pkg.Tool) error
	UnregisterFunc  func(name string) error
	GetFunc         func(name string) (pkg.Tool, error)
	ListFunc        func() []string
	ListSchemasFunc func() []pkg.ToolSchema
	ExecuteFunc     func(ctx context.Context, name string, params json.RawMessage) (pkg.ToolResult, error)
	RegisterHookFunc    func(hook pkg.ToolHook) error
	UnregisterHookFunc  func(hookName string) error
	ListHookNamesFunc   func() []string
}

func (m *MockToolRegistry) Register(tool pkg.Tool) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(tool)
	}
	return nil
}

func (m *MockToolRegistry) Unregister(name string) error {
	if m.UnregisterFunc != nil {
		return m.UnregisterFunc(name)
	}
	return nil
}

func (m *MockToolRegistry) Get(name string) (pkg.Tool, error) {
	if m.GetFunc != nil {
		return m.GetFunc(name)
	}
	return nil, &pkg.ErrToolNotFound{ToolName: name}
}

func (m *MockToolRegistry) List() []string {
	if m.ListFunc != nil {
		return m.ListFunc()
	}
	return nil
}

func (m *MockToolRegistry) ListSchemas() []pkg.ToolSchema {
	if m.ListSchemasFunc != nil {
		return m.ListSchemasFunc()
	}
	return nil
}

func (m *MockToolRegistry) Execute(ctx context.Context, name string, params json.RawMessage) (pkg.ToolResult, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, name, params)
	}
	return pkg.ToolResult{Success: true, Data: json.RawMessage(`{}`)}, nil
}

func (m *MockToolRegistry) RegisterHook(hook pkg.ToolHook) error {
	if m.RegisterHookFunc != nil {
		return m.RegisterHookFunc(hook)
	}
	return nil
}

func (m *MockToolRegistry) UnregisterHook(hookName string) error {
	if m.UnregisterHookFunc != nil {
		return m.UnregisterHookFunc(hookName)
	}
	return nil
}

func (m *MockToolRegistry) ListHookNames() []string {
	if m.ListHookNamesFunc != nil {
		return m.ListHookNamesFunc()
	}
	return nil
}
