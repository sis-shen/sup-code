package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supcode/supcode/pkg"
)

// Bridge connects MCP servers to the ToolRegistry.
type Bridge struct {
	client   pkg.MCPClient
	registry pkg.ToolRegistry
}

// NewBridge creates a new Bridge.
func NewBridge(client pkg.MCPClient, registry pkg.ToolRegistry) *Bridge {
	return &Bridge{
		client:   client,
		registry: registry,
	}
}

// RegisterAllTools discovers all tools from connected MCP servers
// and registers them with the ToolRegistry.
func (b *Bridge) RegisterAllTools(ctx context.Context) error {
	servers := b.client.ListServers()
	var lastErr error

	for _, serverName := range servers {
		tools, err := b.client.ListTools(ctx, serverName)
		if err != nil {
			lastErr = err
			continue
		}

		for _, toolSchema := range tools {
			adapter := &MCPToolAdapter{
				serverName: serverName,
				toolSchema: toolSchema,
				client:     b.client,
			}
			if err := b.registry.Register(adapter); err != nil {
				lastErr = err
			}
		}
	}

	return lastErr
}

// MCPToolAdapter adapts an MCP tool to the Tool interface.
type MCPToolAdapter struct {
	serverName string
	toolSchema pkg.ToolSchema
	client   pkg.MCPClient
}

func (a *MCPToolAdapter) Name() string {
	return a.toolSchema.Name
}

func (a *MCPToolAdapter) Description() string {
	return a.toolSchema.Description
}

func (a *MCPToolAdapter) Schema() pkg.ToolSchema {
	return a.toolSchema
}

func (a *MCPToolAdapter) Execute(ctx context.Context, params json.RawMessage) (pkg.ToolResult, error) {
	return a.client.ExecuteTool(ctx, a.serverName, a.toolSchema.Name, params)
}

// compile-time check
var _ pkg.Tool = (*MCPToolAdapter)(nil)

// suppress unused import warning
var _ = fmt.Sprintf


