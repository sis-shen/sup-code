package mcp

import (
	"context"
	"encoding/json"

	"github.com/supcode/supcode/pkg"
)

// MockClient implements pkg.MCPClient for testing.
type MockClient struct {
	ConnectFunc    func(ctx context.Context, config pkg.MCPServerConfig) error
	DisconnectFunc func(ctx context.Context, serverName string) error
	ListServersFunc func() []string
	ListToolsFunc  func(ctx context.Context, serverName string) ([]pkg.ToolSchema, error)
	ExecuteToolFunc func(ctx context.Context, serverName string, toolName string, params json.RawMessage) (pkg.ToolResult, error)
	CloseFunc      func() error
}

func (m *MockClient) Connect(ctx context.Context, config pkg.MCPServerConfig) error {
	if m.ConnectFunc != nil {
		return m.ConnectFunc(ctx, config)
	}
	return nil
}

func (m *MockClient) Disconnect(ctx context.Context, serverName string) error {
	if m.DisconnectFunc != nil {
		return m.DisconnectFunc(ctx, serverName)
	}
	return nil
}

func (m *MockClient) ListServers() []string {
	if m.ListServersFunc != nil {
		return m.ListServersFunc()
	}
	return nil
}

func (m *MockClient) ListTools(ctx context.Context, serverName string) ([]pkg.ToolSchema, error) {
	if m.ListToolsFunc != nil {
		return m.ListToolsFunc(ctx, serverName)
	}
	return nil, nil
}

func (m *MockClient) ExecuteTool(ctx context.Context, serverName string, toolName string, params json.RawMessage) (pkg.ToolResult, error) {
	if m.ExecuteToolFunc != nil {
		return m.ExecuteToolFunc(ctx, serverName, toolName, params)
	}
	return pkg.ToolResult{Success: true, Data: json.RawMessage(`{}`)}, nil
}

func (m *MockClient) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// compile-time checks
var _ pkg.MCPClient = (*MockClient)(nil)

// suppress unused import warnings
var _ = context.Background
var _ = json.Marshal
