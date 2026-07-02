package mcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supcode/supcode/pkg"
)

func TestJSONRPCError(t *testing.T) {
	err := &jsonRPCError{Code: -32000, Message: "test error"}
	assert.Equal(t, "JSON-RPC error -32000: test error", err.Error())

	err2 := &jsonRPCError{Code: 0, Message: ""}
	assert.Contains(t, err2.Error(), "error 0")
}

func TestClientSendRequest_Disconnected(t *testing.T) {
	c := NewClient()
	_, err := c.ExecuteTool(context.Background(), "nonexistent", "tool", nil)
	require.Error(t, err)
}

func TestStdioSend_AfterProcessCrash(t *testing.T) {
	transport, err := newStdioTransport(context.Background(), pkg.MCPServerConfig{
		Name: "quick_exit", Command: "cmd", Args: []string{"/c", "exit 0"},
	}, 0)
	if err != nil {
		t.Skip("transport init failed:", err)
	}
	defer transport.Close()

	_, err = transport.Send(context.Background(), jsonRPCRequest{Method: "test"})
	require.Error(t, err)
}

func TestNewStdioTransport_RestartWithRetry(t *testing.T) {
	transport, err := newStdioTransport(context.Background(), pkg.MCPServerConfig{
		Name: "test_restart", Command: "cmd", Args: []string{"/c", "exit 1"},
	}, 3)
	if err != nil {
		t.Skip("transport init failed:", err)
	}
	defer transport.Close()

	_, err = transport.Send(context.Background(), jsonRPCRequest{Method: "test"})
	require.Error(t, err)
}

func TestStdioProcessCrashDetection(t *testing.T) {
	transport, err := newStdioTransport(context.Background(), pkg.MCPServerConfig{
		Name: "immediate_exit", Command: "cmd", Args: []string{"/c", "exit 5"},
	}, 0)
	if err != nil {
		t.Skip("transport init failed:", err)
	}
	defer transport.Close()

	_, err = transport.Send(context.Background(), jsonRPCRequest{ID: 42, Method: "test"})
	require.Error(t, err)
}
