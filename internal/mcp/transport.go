package mcp

import (
	"github.com/supcode/supcode/pkg"
	"context"
	"encoding/json"
	"fmt"
)

// ─── JSON-RPC 2.0 types ──────────────────────────────────────────

const jsonRPCVersion = "2.0"

// jsonRPCRequest is a JSON-RPC 2.0 request.
type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// jsonRPCResponse is a JSON-RPC 2.0 response.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

// jsonRPCError is a JSON-RPC 2.0 error.
type jsonRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *jsonRPCError) Error() string {
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

// ─── Transport interface ──────────────────────────────────────────

// Transport abstracts the communication channel to an MCP server.
type Transport interface {
	// Send sends a JSON-RPC request and waits for the response.
	Send(ctx context.Context, req jsonRPCRequest) (jsonRPCResponse, error)
	// Close shuts down the transport.
	Close() error
}

// ─── MCP protocol constants ──────────────────────────────────────

const protocolVersion = "2024-11-05"

// defaultServerConfig is used when connecting via SSE without explicit config.
var defaultServerConfig = pkg.MCPServerConfig{Transport: "stdio"}

// compile-time check
var _ Transport = (*stdioTransport)(nil)
var _ Transport = (*sseTransport)(nil)

