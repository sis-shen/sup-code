// Package mcp exposes the v1 MCP client (internal/mcp) as the plugin-mcp
// Cordis leaf plugin. It only assembles the v1 implementation and registers it
// under pkg.ServiceMCP; the MCP protocol logic is reused unchanged.
//
// The plugin depends on the tools service so that the MCP Bridge can later
// register discovered MCP tools with the shared registry. It does not connect
// to any server at Apply time: connections are runtime/config-driven.
package mcp

import (
	"github.com/supcode/supcode/core"
	mcpclient "github.com/supcode/supcode/internal/mcp"
	"github.com/supcode/supcode/pkg"
)

const (
	pluginName    = "plugin-mcp"
	pluginVersion = "2.0.0"
)

// Options configures the MCP plugin. It is intentionally empty for now; it
// exists so the constructor shape can grow without breaking callers.
type Options struct{}

// Plugin returns the plugin-mcp leaf plugin. It injects the tool registry and
// provides a fresh v1 MCP client under pkg.ServiceMCP. Cleanup closes the
// client when the plugin scope is disposed.
func Plugin(_ Options) core.Plugin {
	return core.Plugin{
		Name:     pluginName,
		Inject:   []string{pkg.ServiceTools},
		Provides: []string{pkg.ServiceMCP},
		Apply: func(ctx *core.Context) error {
			client := mcpclient.NewClient()
			core.Provide(ctx, pkg.ServiceMCP, client)

			return ctx.Effect(func() (core.Disposer, error) {
				return func() error { return client.Close() }, nil
			})
		},
	}
}

// Bridge re-exports the v1 MCP Bridge so callers can register the tools
// discovered on connected MCP servers with the shared ToolRegistry without
// importing internal/mcp directly.
func Bridge(client pkg.MCPClient, registry pkg.ToolRegistry) *mcpclient.Bridge {
	return mcpclient.NewBridge(client, registry)
}

// Manifest returns the declarative metadata for plugin-mcp. It mirrors
// sup.plugin.json.
func Manifest() core.Manifest {
	return core.Manifest{
		Name:        pluginName,
		Version:     pluginVersion,
		Description: "MCP protocol client exposed as the mcp service",
		Inject:      []string{pkg.ServiceTools},
		Provides:    []string{pkg.ServiceMCP},
		Entry:       "builtin:mcp",
	}
}
