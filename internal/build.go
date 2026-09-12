package internal

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/supcode/supcode/internal/agent"
	"github.com/supcode/supcode/internal/config"
	"github.com/supcode/supcode/internal/contextmgr"
	"github.com/supcode/supcode/internal/hooks/builtin"
	"github.com/supcode/supcode/internal/llm"
	"github.com/supcode/supcode/internal/mcp"
	"github.com/supcode/supcode/internal/permission"
	"github.com/supcode/supcode/internal/skill"
	"github.com/supcode/supcode/internal/tools"
	"github.com/supcode/supcode/internal/tools/bash"
	"github.com/supcode/supcode/internal/tools/editfile"
	"github.com/supcode/supcode/internal/tools/glob"
	"github.com/supcode/supcode/internal/tools/grep"
	"github.com/supcode/supcode/internal/tools/readfile"
	"github.com/supcode/supcode/internal/tools/skilltool"
	"github.com/supcode/supcode/internal/tools/writefile"
	"github.com/supcode/supcode/pkg"
)

// BuildAgent wires all layers and returns a ready-to-use Agent.
func BuildAgent(cfg pkg.Config) (pkg.Agent, error) {
	permEng := permission.New()

	reg := tools.NewRegistry(permEng)
	for _, tool := range []pkg.Tool{
		&bash.Tool{},
		&readfile.Tool{},
		&writefile.Tool{},
		&editfile.Tool{},
		&glob.Tool{},
		&grep.Tool{},
	} {
		if err := reg.Register(tool); err != nil {
			return nil, err
		}
	}

	// 接入 Skill：把技能库包装为标准 pkg.Tool 注册进同一个 Tool Pool，
	// 由 LLM 通过 function calling 自主决定何时 list/load 技能。
	builtinDir, userDir, projectDir := skill.GetSkillDirs()
	skillLoader := skill.NewSkillLoader(builtinDir, userDir, projectDir)
	if err := reg.Register(skilltool.New(skillLoader)); err != nil {
		return nil, err
	}

	// 接入 MCP：读取 mcp.servers 配置，连接外部 MCP server，
	// 并通过 Bridge 将其工具适配为 pkg.Tool 注册进与内置工具相同的 Tool Pool。
	// 单个 server 连接失败不阻塞启动（优雅降级），仅记录告警。
	mcpClient := connectMCP(cfg, reg)

	// 接入内置 Hook：使审计日志与 Git 自动提交真正参与每次工具调用。
	// 默认策略保守：审计默认开，Git 自动提交默认关，避免误伤用户仓库。
	if cfg.GetBool(config.ConfigKeyHooksAuditEnabled) {
		if err := reg.RegisterHook(builtin.NewAuditHook(permEng)); err != nil {
			return nil, fmt.Errorf("register audit hook: %w", err)
		}
	}
	if cfg.GetBool(config.ConfigKeyHooksGitCommitEnabled) {
		if err := reg.RegisterHook(builtin.NewGitCommitHook(true)); err != nil {
			return nil, fmt.Errorf("register git commit hook: %w", err)
		}
	}

	llmCfg := llm.Config{
		Provider:  cfg.GetString(config.ConfigKeyLLMProvider),
		APIKey:    cfg.GetString(config.ConfigKeyLLMAPIKey),
		BaseURL:   cfg.GetString(config.ConfigKeyLLMBaseURL),
		Model:     cfg.GetString(config.ConfigKeyLLMModel),
		MaxTokens: cfg.GetInt(config.ConfigKeyLLMMaxTokens),
	}
	llmClient, err := llm.NewLLMClient(llmCfg)
	if err != nil {
		return nil, err
	}

	ctxMgr := contextmgr.NewManager()

	a, err := agent.NewAgent(agent.AgentConfig{
		LLMClient:      llmClient,
		ToolRegistry:   reg,
		ContextManager: ctxMgr,
	})
	if err != nil {
		if mcpClient != nil {
			_ = mcpClient.Close()
		}
		return nil, err
	}

	if mcpClient != nil {
		return &agentWithCleanup{Agent: a, mcpClient: mcpClient}, nil
	}
	return a, nil
}

// connectMCP 读取 mcp.servers 配置并连接 MCP server，将其工具注册进 reg。
// 返回已创建的 MCPClient（无配置时为 nil），供调用方在关闭时释放连接。
func connectMCP(cfg pkg.Config, reg pkg.ToolRegistry) pkg.MCPClient {
	var servers []pkg.MCPServerConfig
	if err := cfg.UnmarshalKey(config.ConfigKeyMCPServers, &servers); err != nil {
		slog.Warn("parse mcp.servers config failed, skipping mcp tools", "error", err)
		return nil
	}
	if len(servers) == 0 {
		return nil
	}

	client := mcp.NewClient()
	for _, r := range mcp.ConnectConfigured(context.Background(), client, servers) {
		if r.Err != nil {
			slog.Warn("mcp server connect failed, skipping its tools", "server", r.ServerName, "error", r.Err)
		}
	}

	if err := mcp.NewBridge(client, reg).RegisterAllTools(context.Background()); err != nil {
		slog.Warn("some mcp tools failed to register", "error", err)
	}

	return client
}

// agentWithCleanup 在 Agent 之上挂载 MCP 连接的释放逻辑。
type agentWithCleanup struct {
	pkg.Agent
	mcpClient pkg.MCPClient
}

// Close 关闭底层 MCP 连接。
func (a *agentWithCleanup) Close() error {
	if a.mcpClient != nil {
		return a.mcpClient.Close()
	}
	return nil
}
