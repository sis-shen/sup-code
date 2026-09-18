package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/pkg"
)

func TestSSEReadSSE_MultipleEvents(t *testing.T) {
	s := &sseTransport{
		pending: make(map[int]chan jsonRPCResponse),
	}

	data := "event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"protocolVersion\":\"2024-11-05\"}}\n\nevent: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{\"tools\":[]}}\n\n"
	respCh1 := make(chan jsonRPCResponse, 1)
	respCh2 := make(chan jsonRPCResponse, 1)
	s.pending[1] = respCh1
	s.pending[2] = respCh2

	go s.readSSE(io.NopCloser(bytes.NewReader([]byte(data))))

	r1 := <-respCh1
	assert.Contains(t, string(r1.Result), "2024-11-05")
	r2 := <-respCh2
	assert.Contains(t, string(r2.Result), "tools")
}

func TestSSEReadSSE_CommentLines(t *testing.T) {
	s := &sseTransport{
		pending: make(map[int]chan jsonRPCResponse),
	}

	data := ": this is a comment\nevent: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":5,\"result\":{\"ok\":true}}\n\n"
	respCh := make(chan jsonRPCResponse, 1)
	s.pending[5] = respCh

	go s.readSSE(io.NopCloser(bytes.NewReader([]byte(data))))

	resp := <-respCh
	assert.Contains(t, string(resp.Result), "ok")
}

func TestSSEReadSSE_EventID(t *testing.T) {
	s := &sseTransport{
		pending: make(map[int]chan jsonRPCResponse),
	}

	data := "id: event-42\nevent: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":3,\"result\":{\"value\":\"test\"}}\n\n"
	respCh := make(chan jsonRPCResponse, 1)
	s.pending[3] = respCh

	go s.readSSE(io.NopCloser(bytes.NewReader([]byte(data))))

	resp := <-respCh
	assert.Contains(t, string(resp.Result), "test")
	assert.Equal(t, "event-42", s.sessionID)
}

func TestMCPToolAdapter_WithMockClient(t *testing.T) {
	client := &MockClient{
		ExecuteToolFunc: func(ctx context.Context, serverName, toolName string, params json.RawMessage) (pkg.ToolResult, error) {
			return pkg.ToolResult{Success: true, Data: json.RawMessage(`{"echoed":true}`)}, nil
		},
	}

	adapter := &MCPToolAdapter{
		serverName: "mcp-test",
		toolSchema: pkg.ToolSchema{
			Name:        "echo",
			Description: "Echo tool",
			Parameters:  json.RawMessage(`{"type":"object"}`),
		},
		client: client,
	}

	assert.Equal(t, "echo", adapter.Name())
	result, err := adapter.Execute(context.Background(), json.RawMessage(`{"msg":"hi"}`))
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestClientDefaultConfig(t *testing.T) {
	assert.Equal(t, "2024-11-05", protocolVersion)
}
