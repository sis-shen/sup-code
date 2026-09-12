package mcp

import (
	"context"

	"github.com/supcode/supcode/pkg"
)

// ConnectResult 记录单个 MCP server 的连接结果。
type ConnectResult struct {
	ServerName string
	Err        error
}

// ConnectConfigured 依次连接给定的 MCP server 配置列表。
// 单个 server 连接失败不会中断其余 server 的连接（优雅降级），
// 失败信息通过返回的 ConnectResult 交给调用方记录日志。
func ConnectConfigured(ctx context.Context, client pkg.MCPClient, servers []pkg.MCPServerConfig) []ConnectResult {
	results := make([]ConnectResult, 0, len(servers))
	for _, s := range servers {
		err := client.Connect(ctx, s)
		results = append(results, ConnectResult{ServerName: s.Name, Err: err})
	}
	return results
}
