package tools

import (
	"context"
	"encoding/json"

	"github.com/supcode/supcode/pkg"
)

// MockTool implements pkg.Tool for testing.
type MockTool struct {
	NameFunc        func() string
	DescriptionFunc func() string
	SchemaFunc      func() pkg.ToolSchema
	ExecuteFunc     func(ctx context.Context, params json.RawMessage) (pkg.ToolResult, error)
}

func (m *MockTool) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return ""
}

func (m *MockTool) Description() string {
	if m.DescriptionFunc != nil {
		return m.DescriptionFunc()
	}
	return ""
}

func (m *MockTool) Schema() pkg.ToolSchema {
	if m.SchemaFunc != nil {
		return m.SchemaFunc()
	}
	return pkg.ToolSchema{}
}

func (m *MockTool) Execute(ctx context.Context, params json.RawMessage) (pkg.ToolResult, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, params)
	}
	return pkg.ToolResult{Success: true, Data: json.RawMessage(`{}`)}, nil
}
