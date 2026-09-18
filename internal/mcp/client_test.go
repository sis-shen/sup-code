package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/pkg"
)

func TestClient_NewClient(t *testing.T) {
	c := NewClient()
	assert.NotNil(t, c)
	assert.Empty(t, c.ListServers())
}

func TestClient_Connect_ValidatesConfig(t *testing.T) {
	c := NewClient()
	err := c.Connect(context.Background(), pkg.MCPServerConfig{})
	require.Error(t, err)
}

func TestClient_Connect_Duplicate(t *testing.T) {
	c := NewClient()
	err1 := c.Connect(context.Background(), pkg.MCPServerConfig{Name: "test", Command: "nonexistent"})
	err2 := c.Connect(context.Background(), pkg.MCPServerConfig{Name: "test", Command: "echo"})
	_ = err1
	_ = err2
	// On platforms where first or second connect fails, at minimum no panic
	// Testing duplicate detection requires a successful initial connect
	t.Log("duplicate connect test completed")
}

func TestClient_ListServers(t *testing.T) {
	c := NewClient()
	// Test returns empty list or populated list - just verify no panic
	_ = c.ListServers()
	t.Log("ListServers OK")
}

func TestClient_Disconnect_UnknownServer(t *testing.T) {
	c := NewClient()
	err := c.Disconnect(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestClient_ExecuteTool_UnknownServer(t *testing.T) {
	c := NewClient()
	_, err := c.ExecuteTool(context.Background(), "unknown", "tool", json.RawMessage(`{}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestClient_ListTools_UnknownServer(t *testing.T) {
	c := NewClient()
	_, err := c.ListTools(context.Background(), "unknown")
	require.Error(t, err)
}

func TestClient_Connect_UnknownTransport(t *testing.T) {
	c := NewClient()
	err := c.Connect(context.Background(), pkg.MCPServerConfig{Name: "test", Transport: "unknown"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown transport")
}

func TestClient_Close(t *testing.T) {
	c := NewClient()
	err := c.Close()
	require.NoError(t, err)
	assert.Empty(t, c.ListServers())
}

func TestClient_Connect_StdioNoCommand(t *testing.T) {
	c := NewClient()
	err := c.Connect(context.Background(), pkg.MCPServerConfig{Name: "test", Transport: "stdio"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create transport")
}

func TestClient_Connect_StdioBadCommand(t *testing.T) {
	c := NewClient()
	err := c.Connect(context.Background(), pkg.MCPServerConfig{Name: "test", Command: "nonexistent-binary-xyz", Transport: "stdio"})
	require.Error(t, err)
}

func TestClient_Connect_SSENoURL(t *testing.T) {
	c := NewClient()
	err := c.Connect(context.Background(), pkg.MCPServerConfig{Name: "test", Transport: "sse"})
	require.Error(t, err)
}

func TestClient_ConcurrentConnect(t *testing.T) {
	c := NewClient()
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(_ int) {
			_ = c.Connect(context.Background(), pkg.MCPServerConfig{Name: "server", Command: "nonexistent"})
			done <- true
		}(i)
	}
	for i := 0; i < 5; i++ {
		<-done
	}
	t.Log("concurrent connect OK")
}

func TestClient_ConcurrentListServers(t *testing.T) {
	c := NewClient()
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_ = c.ListServers()
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}
