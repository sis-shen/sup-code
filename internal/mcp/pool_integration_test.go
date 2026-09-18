package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/internal/tools"
	"github.com/supcode/supcode/internal/tools/bash"
	"github.com/supcode/supcode/pkg"
)

// TestBridge_UnifiedToolPool_WithRealRegistry 是本任务最关键的验收证据：
// MCP 工具通过与内置工具完全相同的 Register/Execute 路径进入同一个
// tools.Registry，证明 MCP 没有独立的第二套工具调度逻辑。
func TestBridge_UnifiedToolPool_WithRealRegistry(t *testing.T) {
	mockSrv := NewMockMCPServer(t, []pkg.ToolSchema{
		{Name: "mcp_echo", Description: "echo via mcp", Parameters: json.RawMessage(`{"type":"object"}`)},
	}, protocolVersion)
	defer mockSrv.Stop()

	reg := tools.NewRegistry(nil)
	require.NoError(t, reg.Register(&bash.Tool{}))

	client := NewClient()
	require.NoError(t, client.Connect(context.Background(),
		pkg.MCPServerConfig{Name: "mock", Transport: "sse", Command: mockSrv.URL()}))
	defer func() { _ = client.Close() }()

	require.NoError(t, NewBridge(client, reg).RegisterAllTools(context.Background()))

	names := reg.List()
	assert.Contains(t, names, "bash", "内置工具应仍在同一个 Registry 中")
	assert.Contains(t, names, "mcp_echo", "MCP 工具应与内置工具共存于同一个 Registry")

	result, err := reg.Execute(context.Background(), "mcp_echo", json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.True(t, result.Success, "通过 Registry.Execute 统一入口应能调用到 MCP 工具")
	assert.Contains(t, string(result.Data), "mock result")
}

// TestConnectConfigured_GracefulDegradation 验证 ConnectConfigured：
// 单个 server 失败不中断其余 server 的连接尝试。
func TestConnectConfigured_GracefulDegradation(t *testing.T) {
	good := NewMockMCPServer(t, nil, protocolVersion)
	defer good.Stop()

	servers := []pkg.MCPServerConfig{
		{Name: "bad", Transport: "stdio", Command: "nonexistent-binary-xyz"},
		{Name: "good", Transport: "sse", Command: good.URL()},
	}

	client := NewClient()
	defer func() { _ = client.Close() }()

	results := ConnectConfigured(context.Background(), client, servers)
	require.Len(t, results, 2)

	byName := map[string]ConnectResult{}
	for _, r := range results {
		byName[r.ServerName] = r
	}
	assert.Error(t, byName["bad"].Err, "坏 server 应记录失败")
	assert.NoError(t, byName["good"].Err, "坏 server 不应阻止好的 server 连接")
	assert.Contains(t, client.ListServers(), "good")
}
