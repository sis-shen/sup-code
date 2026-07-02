package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supcode/supcode/pkg"
)

// ─── SSE transport via mock MCP server ──────────────────────────

func TestSSETransport_ConnectViaMockServer(t *testing.T) {
	mockSrv := NewMockMCPServer(t, nil, protocolVersion)
	defer mockSrv.Stop()

	cfg := pkg.MCPServerConfig{
		Name:      "mock_mcp",
		Transport: "sse",
		Command:   mockSrv.URL(),
	}

	client := NewClient()
	err := client.Connect(context.Background(), cfg)
	require.NoError(t, err)
	defer client.Close()

	assert.Contains(t, client.ListServers(), "mock_mcp")
}

func TestClient_ListToolsViaMockServer(t *testing.T) {
	tools := []pkg.ToolSchema{
		{Name: "echo", Description: "Echo input", Parameters: json.RawMessage(`{"type":"object"}`)},
		{Name: "read", Description: "Read file", Parameters: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}}}`)},
	}
	mockSrv := NewMockMCPServer(t, tools, protocolVersion)
	defer mockSrv.Stop()

	cfg := pkg.MCPServerConfig{Name: "mcp", Transport: "sse", Command: mockSrv.URL()}
	client := NewClient()
	require.NoError(t, client.Connect(context.Background(), cfg))
	defer client.Close()

	result, err := client.ListTools(context.Background(), "mcp")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "echo", result[0].Name)
	assert.Equal(t, "read", result[1].Name)
}

func TestClient_ExecuteToolViaMockServer(t *testing.T) {
	mockSrv := NewMockMCPServer(t, nil, protocolVersion)
	defer mockSrv.Stop()

	cfg := pkg.MCPServerConfig{Name: "mcp", Transport: "sse", Command: mockSrv.URL()}
	client := NewClient()
	require.NoError(t, client.Connect(context.Background(), cfg))
	defer client.Close()

	params, _ := json.Marshal(map[string]string{"msg": "hello"})
	result, err := client.ExecuteTool(context.Background(), "mcp", "echo", params)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, string(result.Data), "mock result")
}

func TestClient_DisconnectViaMockServer(t *testing.T) {
	mockSrv := NewMockMCPServer(t, nil, protocolVersion)
	defer mockSrv.Stop()

	cfg := pkg.MCPServerConfig{Name: "mcp", Transport: "sse", Command: mockSrv.URL()}
	client := NewClient()
	require.NoError(t, client.Connect(context.Background(), cfg))

	err := client.Disconnect(context.Background(), "mcp")
	require.NoError(t, err)
	assert.NotContains(t, client.ListServers(), "mcp")
}

func TestClient_Connect_ProtocolVersionMismatch(t *testing.T) {
	mockSrv := NewMockMCPServer(t, nil, "1.0.0") // different version
	defer mockSrv.Stop()

	cfg := pkg.MCPServerConfig{Name: "mcp", Transport: "sse", Command: mockSrv.URL()}
	client := NewClient()
	err := client.Connect(context.Background(), cfg)
	// Connection succeeds because we don't enforce version matching (just note it)
	require.NoError(t, err)
	defer client.Close()
}

func TestClient_ExecuteTool_NotConnected(t *testing.T) {
	client := NewClient()
	_, err := client.ExecuteTool(context.Background(), "nonexistent", "tool", json.RawMessage(`{}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ─── Bridge via mock MCP server ──────────────────────────────────

func TestBridge_RegisterAllToolsViaMockServer(t *testing.T) {
	tools := []pkg.ToolSchema{
		{Name: "read", Description: "Read file", Parameters: json.RawMessage(`{"type":"object"}`)},
		{Name: "write", Description: "Write file", Parameters: json.RawMessage(`{"type":"object"}`)},
	}
	mockSrv := NewMockMCPServer(t, tools, protocolVersion)
	defer mockSrv.Stop()

	cfg := pkg.MCPServerConfig{Name: "bridge_mcp", Transport: "sse", Command: mockSrv.URL()}
	client := NewClient()
	require.NoError(t, client.Connect(context.Background(), cfg))
	defer client.Close()

	registry := newMockRegistry()
	bridge := NewBridge(client, registry)
	err := bridge.RegisterAllTools(context.Background())
	require.NoError(t, err)

	toolNames := registry.List()
	assert.ElementsMatch(t, []string{"read", "write"}, toolNames)
}

func TestBridge_ExecuteViaMockServer(t *testing.T) {
	mockSrv := NewMockMCPServer(t, nil, protocolVersion)
	defer mockSrv.Stop()

	cfg := pkg.MCPServerConfig{Name: "bmcp", Transport: "sse", Command: mockSrv.URL()}
	client := NewClient()
	require.NoError(t, client.Connect(context.Background(), cfg))
	defer client.Close()

	registry := newMockRegistry()
	bridge := NewBridge(client, registry)
	_ = bridge.RegisterAllTools(context.Background())

	// Create adapter manually for testing
	adapter := &MCPToolAdapter{
		serverName: "bmcp",
		toolSchema: pkg.ToolSchema{Name: "echo", Description: "Echo", Parameters: json.RawMessage(`{"type":"object"}`)},
		client:     client,
	}

	result, err := adapter.Execute(context.Background(), json.RawMessage(`{"msg":"hello"}`))
	require.NoError(t, err)
	assert.True(t, result.Success)
}

// ─── Stdio transport via PowerShell MCP server ───────────────────

func TestStdioTransport_WithPowerShellMCPServer(t *testing.T) {
	scriptDir := t.TempDir()
	scriptPath := filepath.Join(scriptDir, "mcp_server.ps1")
	script := `try {
    $line = [Console]::In.ReadLine()
    while ($line -ne $null) {
        $req = $line | ConvertFrom-Json
        if ($req.method -eq "initialize") {
            $jsonResp = '{"jsonrpc":"2.0","id":' + $req.id + ',"result":{"protocolVersion":"2024-11-05"}}'
        } elseif ($req.method -eq "tools/list") {
            $jsonResp = '{"jsonrpc":"2.0","id":' + $req.id + ',"result":{"tools":[]}}'
        } else {
            $jsonResp = '{"jsonrpc":"2.0","id":' + $req.id + ',"result":{"content":[{"type":"text","text":"ok"}]}}'
        }
        Write-Host $jsonResp
        $line = [Console]::In.ReadLine()
    }
} catch { exit 1 }`
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	transport, err := newStdioTransport(context.Background(), pkg.MCPServerConfig{
		Name:    "ps_mcp",
		Command: "powershell",
		Args:    []string{"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", scriptPath},
	}, 0)
	if err != nil {
		t.Skip("PowerShell MCP server not available:", err)
	}
	defer transport.Close()

	// Test initialize
	resp, err := transport.Send(context.Background(), jsonRPCRequest{
		ID:     1,
		Method: "initialize",
		Params: map[string]interface{}{"protocolVersion": "2024-11-05", "capabilities": map[string]interface{}{}},
	})
	require.NoError(t, err)
	assert.Contains(t, string(resp.Result), "2024-11-05")

	// Test tools/list
	resp, err = transport.Send(context.Background(), jsonRPCRequest{
		ID:     2,
		Method: "tools/list",
	})
	require.NoError(t, err)
	assert.Contains(t, string(resp.Result), "tools")

	// Test tools/call
	resp, err = transport.Send(context.Background(), jsonRPCRequest{
		ID:     3,
		Method: "tools/call",
		Params: map[string]interface{}{"name": "echo", "arguments": map[string]interface{}{}},
	})
	require.NoError(t, err)
	assert.Contains(t, string(resp.Result), "ok")
}

// ─── End-to-end: Client via stdio mock server ────────────────────

func TestClient_ConnectViaPowerShellStdio(t *testing.T) {
	scriptDir := t.TempDir()
	scriptPath := filepath.Join(scriptDir, "mcp_init.ps1")
	script := `try {
    $line = [Console]::In.ReadLine()
    while ($line -ne $null) {
        $req = $line | ConvertFrom-Json
        if ($req.method -eq "initialize") {
            $jsonResp = '{"jsonrpc":"2.0","id":' + $req.id + ',"result":{"protocolVersion":"2024-11-05"}}'
        } elseif ($req.method -eq "tools/list") {
            $jsonResp = '{"jsonrpc":"2.0","id":' + $req.id + ',"result":{"tools":[{"name":"hello","description":"Say hello","parameters":{"type":"object"}}]}}'
        } elseif ($req.method -eq "tools/call") {
            $jsonResp = '{"jsonrpc":"2.0","id":' + $req.id + ',"result":{"content":[{"type":"text","text":"Hello World!"}]}}'
        } else {
            $jsonResp = '{"jsonrpc":"2.0","id":' + $req.id + ',"result":{}}'
        }
        Write-Host $jsonResp
        $line = [Console]::In.ReadLine()
    }
} catch { exit 1 }`
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	client := NewClient()
	err := client.Connect(context.Background(), pkg.MCPServerConfig{
		Name:      "ps_mcp",
		Command:   "powershell",
		Args:      []string{"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", scriptPath},
		Transport: "stdio",
	})
	if err != nil {
		t.Skip("PowerShell MCP init failed:", err)
	}
	defer client.Close()

	require.Contains(t, client.ListServers(), "ps_mcp")

	tools, err := client.ListTools(context.Background(), "ps_mcp")
	require.NoError(t, err)
	require.Len(t, tools, 1)
	assert.Equal(t, "hello", tools[0].Name)

	result, err := client.ExecuteTool(context.Background(), "ps_mcp", "hello", json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, string(result.Data), "Hello World!")
}

func TestClient_FullLifecycleViaMockSSE(t *testing.T) {
	mockSrv := NewMockMCPServer(t, nil, protocolVersion)
	defer mockSrv.Stop()

	client := NewClient()
	cfg := pkg.MCPServerConfig{Name: "lifecycle", Transport: "sse", Command: mockSrv.URL()}

	// Connect
	require.NoError(t, client.Connect(context.Background(), cfg))

	// ListTools
	tools, err := client.ListTools(context.Background(), "lifecycle")
	require.NoError(t, err)

	// ExecuteTool
	result, err := client.ExecuteTool(context.Background(), "lifecycle", "test", json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.True(t, result.Success)
	_ = tools

	// Verify requests were received
	reqs := mockSrv.ReceivedRequests()
	methods := make([]string, len(reqs))
	for i, r := range reqs {
		methods[i] = r.Method
	}
	assert.Contains(t, methods, "initialize")
	assert.Contains(t, methods, "tools/list")
	assert.Contains(t, methods, "tools/call")

	// Disconnect
	require.NoError(t, client.Disconnect(context.Background(), "lifecycle"))
	assert.NotContains(t, client.ListServers(), "lifecycle")
}

// suppress unused imports
var _ = time.Second
var _ = os.PathSeparator

