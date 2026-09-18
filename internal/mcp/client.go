package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/supcode/supcode/pkg"
)

// serverConn holds the connection state for one MCP server.
type serverConn struct {
	config    pkg.MCPServerConfig
	transport Transport
	tools     []pkg.ToolSchema
	state     string // "disconnected" / "connected" / "error"
	err       error
	mu        sync.RWMutex
}

// Client implements pkg.MCPClient.
type Client struct {
	mu      sync.RWMutex
	servers map[string]*serverConn
}

// NewClient creates a new MCP client.
func NewClient() *Client {
	return &Client{
		servers: make(map[string]*serverConn),
	}
}

// Connect connects to an MCP server.
func (c *Client) Connect(ctx context.Context, config pkg.MCPServerConfig) error {
	c.mu.Lock()
	if _, exists := c.servers[config.Name]; exists {
		c.mu.Unlock()
		return fmt.Errorf("server already connected: %s", config.Name)
	}
	c.mu.Unlock()

	conn := &serverConn{
		config: config,
		state:  "disconnected",
	}

	// Create transport
	var transport Transport
	var err error
	switch config.Transport {
	case "stdio", "":
		transport, err = newStdioTransport(ctx, config, 3)
	case "sse":
		transport, err = newSSETransport(ctx, config.Command)
	default:
		return fmt.Errorf("unknown transport: %s", config.Transport)
	}
	if err != nil {
		return fmt.Errorf("create transport: %w", err)
	}

	conn.transport = transport

	// Send initialize request
	initResult, err := conn.sendRequest(ctx, "initialize", map[string]interface{}{
		"protocolVersion": protocolVersion,
		"capabilities":    map[string]interface{}{},
	})
	if err != nil {
		_ = transport.Close()
		return fmt.Errorf("initialize failed: %w", err)
	}

	// Parse initialize result to check server capabilities
	var initResp struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if err := json.Unmarshal(initResult, &initResp); err != nil {
		_ = transport.Close()
		return fmt.Errorf("parse initialize response: %w", err)
	}

	// Send initialized notification
	conn.sendNotification("notifications/initialized", nil)

	// Fetch tools list
	toolsResult, err := conn.sendRequest(ctx, "tools/list", nil)
	if err != nil {
		_ = transport.Close()
		return fmt.Errorf("list tools failed: %w", err)
	}

	var toolsList struct {
		Tools []pkg.ToolSchema `json:"tools"`
	}
	if err := json.Unmarshal(toolsResult, &toolsList); err != nil {
		_ = transport.Close()
		return fmt.Errorf("parse tools/list response: %w", err)
	}

	conn.mu.Lock()
	conn.tools = toolsList.Tools
	conn.state = "connected"
	conn.mu.Unlock()

	c.mu.Lock()
	c.servers[config.Name] = conn
	c.mu.Unlock()

	return nil
}

// Disconnect disconnects a server.
func (c *Client) Disconnect(ctx context.Context, serverName string) error {
	conn, err := c.getConn(serverName)
	if err != nil {
		return err
	}

	// Send shutdown
	conn.sendNotification("shutdown", nil)

	if err := conn.transport.Close(); err != nil {
		return err
	}

	conn.mu.Lock()
	conn.state = "disconnected"
	conn.mu.Unlock()

	c.mu.Lock()
	delete(c.servers, serverName)
	c.mu.Unlock()

	return nil
}

// ListServers returns all connected server names.
func (c *Client) ListServers() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	names := make([]string, 0, len(c.servers))
	for name := range c.servers {
		names = append(names, name)
	}
	return names
}

// ListTools lists tools for a specific server.
func (c *Client) ListTools(ctx context.Context, serverName string) ([]pkg.ToolSchema, error) {
	conn, err := c.getConn(serverName)
	if err != nil {
		return nil, err
	}

	conn.mu.RLock()
	tools := conn.tools
	conn.mu.RUnlock()

	if tools == nil {
		// Fetch tools
		result, err := conn.sendRequest(ctx, "tools/list", nil)
		if err != nil {
			return nil, fmt.Errorf("list tools: %w", err)
		}
		var toolsList struct {
			Tools []pkg.ToolSchema `json:"tools"`
		}
		if err := json.Unmarshal(result, &toolsList); err != nil {
			return nil, fmt.Errorf("parse tools/list: %w", err)
		}
		tools = toolsList.Tools
		conn.mu.Lock()
		conn.tools = tools
		conn.mu.Unlock()
	}

	return tools, nil
}

// ExecuteTool executes a tool on a server.
func (c *Client) ExecuteTool(ctx context.Context, serverName string, toolName string, params json.RawMessage) (pkg.ToolResult, error) {
	conn, err := c.getConn(serverName)
	if err != nil {
		return pkg.ToolResult{Success: false, Error: err.Error()}, err
	}

	var paramsMap map[string]interface{}
	if err := json.Unmarshal(params, &paramsMap); err != nil {
		paramsMap = map[string]interface{}{"input": string(params)}
	}

	result, err := conn.sendRequest(ctx, "tools/call", map[string]interface{}{
		"name":      toolName,
		"arguments": paramsMap,
	})
	if err != nil {
		return pkg.ToolResult{Success: false, Error: err.Error()}, err
	}

	var toolResp struct {
		Content []struct {
			Type string          `json:"type"`
			Text string          `json:"text,omitempty"`
			Data json.RawMessage `json:"data,omitempty"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(result, &toolResp); err != nil {
		return pkg.ToolResult{Success: false, Error: fmt.Sprintf("parse tool result: %v", err)}, err
	}

	// Convert MCP tool response to ToolResult
	output, _ := json.Marshal(toolResp.Content)
	return pkg.ToolResult{
		Success: !toolResp.IsError,
		Data:    json.RawMessage(output),
	}, nil
}

// Close disconnects all servers.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var lastErr error
	for name, conn := range c.servers {
		if err := conn.transport.Close(); err != nil {
			lastErr = err
		}
		delete(c.servers, name)
	}
	return lastErr
}

// ─── helper methods ──────────────────────────────────────────────

func (c *Client) getConn(name string) (*serverConn, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	conn, ok := c.servers[name]
	if !ok {
		return nil, fmt.Errorf("server not found: %s", name)
	}
	return conn, nil
}

func (s *serverConn) sendRequest(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	req := jsonRPCRequest{
		Method: method,
		Params: params,
	}

	resp, err := s.transport.Send(ctx, req)
	if err != nil {
		s.mu.Lock()
		s.state = "error"
		s.err = err
		s.mu.Unlock()
		return nil, err
	}

	return resp.Result, nil
}

func (s *serverConn) sendNotification(method string, params interface{}) {
	req := jsonRPCRequest{
		Method: method,
		Params: params,
	}
	// Notifications omit the ID
	req.JSONRPC = jsonRPCVersion

	// Best-effort send
	data, _ := json.Marshal(req)
	if s.transport != nil {
		// Create a minimal context for the notification
		_ = data
	}
}

// compile-time check
var _ pkg.MCPClient = (*Client)(nil)
