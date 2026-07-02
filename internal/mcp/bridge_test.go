package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supcode/supcode/pkg"
)

type mockRegistry struct {
	mu    sync.Mutex
	tools map[string]pkg.Tool
}

func newMockRegistry() *mockRegistry {
	return &mockRegistry{tools: make(map[string]pkg.Tool)}
}

func (r *mockRegistry) Register(tool pkg.Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := tool.Name()
	if name == "" { return fmt.Errorf("empty name") }
	if _, exists := r.tools[name]; exists { return fmt.Errorf("already exists: %s", name) }
	r.tools[name] = tool
	return nil
}
func (r *mockRegistry) Unregister(name string) error          { return nil }
func (r *mockRegistry) Get(name string) (pkg.Tool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tools[name]
	if !ok { return nil, fmt.Errorf("not found: %s", name) }
	return t, nil
}
func (r *mockRegistry) List() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	names := make([]string, 0, len(r.tools))
	for n := range r.tools { names = append(names, n) }
	return names
}
func (r *mockRegistry) ListSchemas() []pkg.ToolSchema { return nil }
func (r *mockRegistry) Execute(ctx context.Context, name string, params json.RawMessage) (pkg.ToolResult, error) { return pkg.ToolResult{}, nil }
func (r *mockRegistry) RegisterHook(hook pkg.ToolHook) error   { return nil }
func (r *mockRegistry) UnregisterHook(hookName string) error   { return nil }

func TestBridge_New(t *testing.T) {
	client := NewClient()
	registry := newMockRegistry()
	bridge := NewBridge(client, registry)
	assert.NotNil(t, bridge)
}

func TestBridge_RegisterAllTools_NoServers(t *testing.T) {
	client := NewClient()
	registry := newMockRegistry()
	bridge := NewBridge(client, registry)
	err := bridge.RegisterAllTools(context.Background())
	require.NoError(t, err)
	assert.Empty(t, registry.List())
}

func TestMCPToolAdapter_Basic(t *testing.T) {
	client := NewClient()
	schema := pkg.ToolSchema{Name: "test_tool", Description: "A test tool", Parameters: json.RawMessage(`{"type":"object"}`)}
	adapter := &MCPToolAdapter{serverName: "test_server", toolSchema: schema, client: client}
	assert.Equal(t, "test_tool", adapter.Name())
	assert.Equal(t, "A test tool", adapter.Description())
	assert.Equal(t, schema, adapter.Schema())
}

func TestMCPToolAdapter_Execute_UnknownServer(t *testing.T) {
	client := NewClient()
	adapter := &MCPToolAdapter{serverName: "nonexistent", toolSchema: pkg.ToolSchema{Name: "tool"}, client: client}
	result, err := adapter.Execute(context.Background(), json.RawMessage(`{}`))
	require.Error(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "not found")
}

func TestBridge_RegisterAllTools_AfterConnect(t *testing.T) {
	client := NewClient()
	registry := newMockRegistry()
	bridge := NewBridge(client, registry)
	_ = client.Connect(context.Background(), pkg.MCPServerConfig{Name: "failing_server", Command: "echo"})
	err := bridge.RegisterAllTools(context.Background())
	t.Logf("RegisterAllTools result: %v", err)
}

func TestMCPToolAdapter_SchemaPassthrough(t *testing.T) {
	params := json.RawMessage(`{"type":"object"}`)
	schema := pkg.ToolSchema{Name: "read_file", Description: "Read", Parameters: params}
	adapter := &MCPToolAdapter{serverName: "fs", toolSchema: schema}
	assert.Equal(t, params, adapter.Schema().Parameters)
}
