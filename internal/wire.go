
package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/supcode/supcode/internal/agent"
	"github.com/supcode/supcode/internal/config"
	"github.com/supcode/supcode/internal/contextmgr"
	"github.com/supcode/supcode/internal/llm/openai"
	"github.com/supcode/supcode/internal/permission"
	"github.com/supcode/supcode/internal/tools"
	"github.com/supcode/supcode/internal/tools/bash"
	"github.com/supcode/supcode/internal/tools/editfile"
	"github.com/supcode/supcode/internal/tools/glob"
	"github.com/supcode/supcode/internal/tools/grep"
	"github.com/supcode/supcode/internal/tools/readfile"
	"github.com/supcode/supcode/internal/tools/writefile"
	"github.com/supcode/supcode/internal/tui"
	"github.com/supcode/supcode/pkg"
)

// SupCode 持有所有顶层组件的集成容器
type SupCode struct {
	Config     *config.Manager
	PermEngine *permission.Engine
	CtxMgr     *contextmgr.Manager
	ToolReg    *tools.Registry
	LLMClient  *openai.LLMClient
	Agent      *agent.Agent
	SessionMgr *tui.SessionManager
	Service    *tui.Service
}

// NewSupCode 创建并组装所有层的真实实现。
// configPath 可为空字符串以使用默认路径。
func NewSupCode(configPath string) (*SupCode, error) {
	// ── 1. 配置管理器 ──────────────────────────────────────
	cfg := config.NewManager()
	if configPath != "" {
		if err := cfg.Set("config", configPath); err != nil {
			return nil, fmt.Errorf("set config path: %w", err)
		}
	}
	if err := cfg.Load(); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// ── 2. 安全层：权限引擎 ────────────────────────────────
	permEng := permission.New()

	// ── 3. 记忆层：上下文管理器 ────────────────────────────
	ctxMgr := contextmgr.NewManager()

	// ── 4. 工具层：注册中心 + 六大内置工具 ────────────────
	toolReg := tools.NewRegistry(permEng)
	builtinTools := []pkg.Tool{
		&readfile.Tool{},
		&writefile.Tool{},
		&editfile.Tool{},
		&bash.Tool{},
		&glob.Tool{},
		&grep.Tool{},
	}
	for _, t := range builtinTools {
		if err := toolReg.Register(t); err != nil {
			return nil, fmt.Errorf("register tool %q: %w", t.Name(), err)
		}
	}

	// ── 5. 引擎层：LLM 客户端 ──────────────────────────────
	llmCfg := openai.Config{
		APIKey:    cfg.GetString(config.ConfigKeyLLMAPIKey),
		BaseURL:   cfg.GetString(config.ConfigKeyLLMBaseURL),
		Model:     cfg.GetString(config.ConfigKeyLLMModel),
		MaxTokens: cfg.GetInt(config.ConfigKeyLLMMaxTokens),
	}
	llmClient := openai.NewLLMClient(llmCfg)

	// ── 6. 交互层：会话管理器 + 服务 ──────────────────────
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir: %w", err)
	}
	dbPath := filepath.Join(homeDir, ".supcode", "sessions.json")
	sessionMgr, err := tui.NewSessionManager(dbPath)
	if err != nil {
		return nil, fmt.Errorf("session manager: %w", err)
	}
	service := tui.NewService(sessionMgr)

	// ── 7. 引擎层：Agent 主循环 ────────────────────────────
	agentCfg := agent.AgentConfig{
		LLMClient:      llmClient,
		ToolRegistry:   toolReg,
		ContextManager: ctxMgr,
		SessionManager: sessionMgr,
		MaxIterations:  cfg.GetInt(config.ConfigKeyAgentMaxIterations),
	}
	ag, err := agent.NewAgent(agentCfg)
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}

	return &SupCode{
		Config:     cfg,
		PermEngine: permEng,
		CtxMgr:     ctxMgr,
		ToolReg:    toolReg,
		LLMClient:  llmClient,
		Agent:      ag,
		SessionMgr: sessionMgr,
		Service:    service,
	}, nil
}

// Close 优雅关闭所有组件，释放资源
func (s *SupCode) Close() error {
	var firstErr error
	if s.SessionMgr != nil {
		if err := s.SessionMgr.CloseAll(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
