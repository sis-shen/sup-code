package config

// ─── 9.2 配置键常量 ────────────────────────────────────────────

const (
	ConfigKeyLLMProvider  = "llm.provider"
	ConfigKeyLLMModel     = "llm.model"
	ConfigKeyLLMAPIKey    = "llm.api_key"
	ConfigKeyLLMBaseURL   = "llm.base_url"
	ConfigKeyLLMMaxTokens = "llm.max_tokens"

	ConfigKeyAgentMaxIterations = "agent.max_iterations"
	ConfigKeyAgentPermission    = "agent.permission_level"

	ConfigKeyMemoryProjectDir        = "memory.project_memory_dir"
	ConfigKeyMemoryUserDir           = "memory.user_memory_dir"
	ConfigKeyMemoryCompressThreshold = "memory.compress_threshold"

	ConfigKeyMCPServers = "mcp.servers"

	ConfigKeySkillsAutoLoad = "skills.auto_load"

	ConfigKeyHooksAuditEnabled     = "hooks.audit_log.enabled"
	ConfigKeyHooksGitCommitEnabled = "hooks.git_auto_commit.enabled"
)

// DefaultConfig 默认配置值
var DefaultConfig = map[string]any{
	ConfigKeyLLMProvider:             "openai",
	ConfigKeyLLMModel:                "gpt-4o",
	ConfigKeyLLMBaseURL:              "",
	ConfigKeyLLMMaxTokens:            4096,
	ConfigKeyAgentMaxIterations:      25,
	ConfigKeyAgentPermission:         "ask",
	ConfigKeyMemoryProjectDir:        ".supcode/memory",
	ConfigKeyMemoryUserDir:           "~/.supcode/memory",
	ConfigKeyMemoryCompressThreshold: 8000,
	ConfigKeySkillsAutoLoad:          true,
	ConfigKeyHooksAuditEnabled:       true,
	ConfigKeyHooksGitCommitEnabled:   false,
}

// ConfigKeyDefault 默认配置键映射（key -> 默认值）
// 用于快速判断是否已有默认值
func ConfigKeyDefault(key string) (any, bool) {
	v, ok := DefaultConfig[key]
	return v, ok
}
