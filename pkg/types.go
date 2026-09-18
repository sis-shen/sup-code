package pkg

import (
	"encoding/json"
	"time"
)

// ─── 3.1 消息与角色 ─────────────────────────────────────────────

// Role 消息角色
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message 对话消息
type Message struct {
	Role      Role       `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	ToolID    string     `json:"tool_id,omitempty"` // role=tool 时关联的调用 ID
	Timestamp time.Time  `json:"timestamp"`
}

// ─── 3.2 工具调用 ───────────────────────────────────────────────

// ToolCall 工具调用请求
type ToolCall struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Params json.RawMessage `json:"params"`
}

// ToolResult 工具调用结果
type ToolResult struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// ─── 3.3 任务计划 ───────────────────────────────────────────────

// PlanItem 计划步骤
type PlanItem struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Status      string `json:"status"` // pending / in_progress / completed / skipped
	ToolHint    string `json:"tool_hint,omitempty"`
}

// Plan 任务计划
type Plan struct {
	Steps []PlanItem `json:"steps"`
	Goal  string     `json:"goal"`
}

// ─── 3.4 会话 ───────────────────────────────────────────────────

// LoopState Agent Loop 状态
type LoopState string

const (
	StateIdle            LoopState = "idle"
	StatePlanning        LoopState = "planning"
	StateWaitingApproval LoopState = "waiting_approval"
	StateActing          LoopState = "acting"
	StateObserving       LoopState = "observing"
	StateCompleted       LoopState = "completed"
	StateError           LoopState = "error"
)

// Session 会话
type Session struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Messages   []Message `json:"messages"`
	Plan       *Plan     `json:"plan,omitempty"`
	State      LoopState `json:"state"`
	TokenCount int       `json:"token_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ─── 3.5 权限决策 ───────────────────────────────────────────────

// Decision 权限决策结果
type Decision string

const (
	DecisionAllow Decision = "allow"
	DecisionAsk   Decision = "ask"
	DecisionDeny  Decision = "deny"
)

// Action 待审核的操作
type Action struct {
	Type     string `json:"type"`   // "command" / "file_write" / "file_delete" / "tool" / "mcp"
	Target   string `json:"target"` // 命令文本 / 文件路径 / 工具名
	ToolName string `json:"tool_name,omitempty"`
	Params   string `json:"params,omitempty"`
}

// ─── 3.6 StreamEvent ────────────────────────────────────────────

// StreamEvent LLM 返回的流式事件
type StreamEvent struct {
	Type     string    `json:"type"` // "text_delta" / "tool_call" / "done" / "error"
	Delta    string    `json:"delta,omitempty"`
	ToolCall *ToolCall `json:"tool_call,omitempty"`
	Error    string    `json:"error,omitempty"`
}

// ─── 4.1 交互层 DTO ─────────────────────────────────────────────

// ConfirmPrompt 确认提示
type ConfirmPrompt struct {
	Title        string `json:"title"`         // 简短标题
	Message      string `json:"message"`       // 详细描述
	ActionType   string `json:"action_type"`   // "file_write" / "command" / "tool" / "mcp"
	ActionDetail string `json:"action_detail"` // 具体操作描述
}

// NotifyLevel 通知级别
type NotifyLevel string

const (
	NotifyInfo  NotifyLevel = "info"
	NotifyWarn  NotifyLevel = "warn"
	NotifyError NotifyLevel = "error"
)

// CommandResult 命令处理结果
type CommandResult struct {
	Handled    bool   `json:"handled"`     // 是否被识别和处理
	Message    string `json:"message"`     // 反馈消息
	NewSession bool   `json:"new_session"` // 是否为创建新会话的命令
}

// ─── 5.1 Agent 层 DTO ───────────────────────────────────────────

// AgentResult Agent 执行结果
type AgentResult struct {
	Plan    *Plan  `json:"plan,omitempty"`  // 执行的计划
	Summary string `json:"summary"`         // 结果摘要
	Error   string `json:"error,omitempty"` // 如有错误
}

// ModelInfo 模型信息
type ModelInfo struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	MaxTokens      int    `json:"max_tokens"`
	SupportsVision bool   `json:"supports_vision"`
}

// ToolSchema 工具的参数 Schema（JSON Schema 格式，供 LLM function calling）
type ToolSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ─── 5.6 子任务 DTO ─────────────────────────────────────────────

// SubTask 子任务定义
type SubTask struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Context     string `json:"context"` // 传递给 SubAgent 的上下文
}

// SubTaskResult 子任务结果
type SubTaskResult struct {
	TaskID  string `json:"task_id"`
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

// ─── 7.1 记忆层 DTO ─────────────────────────────────────────────

// MemoryCard 记忆卡片（压缩后的对话摘要）
type MemoryCard struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`  // 摘要内容
	Category  string    `json:"category"` // "decision" / "error" / "preference" / "context"
	CreatedAt time.Time `json:"created_at"`
}

// MemoryEntry 记忆条目
type MemoryEntry struct {
	ID        string    `json:"id"`
	Scope     string    `json:"scope"`    // "project" / "user"
	Category  string    `json:"category"` // "structure" / "convention" / "preference" / "task"
	Content   string    `json:"content"`
	Embedding []float32 `json:"embedding,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ─── 8.1 安全层 DTO ─────────────────────────────────────────────

// PermissionRule 权限规则
type PermissionRule struct {
	ID       string   `json:"id"`
	Scope    string   `json:"scope"`    // "command" / "path" / "tool" / "mcp"
	Pattern  string   `json:"pattern"`  // 匹配模式（正则或前缀）
	Decision Decision `json:"decision"` // allow / ask / deny
	Priority int      `json:"priority"` // 优先级，数字越小越优先
}

// WorktreeInfo Worktree 信息
type WorktreeInfo struct {
	AgentID   string    `json:"agent_id"`
	Path      string    `json:"path"`
	Branch    string    `json:"branch"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"` // "active" / "merged" / "abandoned"
}

// ─── 6.4 MCP DTO ────────────────────────────────────────────────

// MCPServerConfig MCP 服务配置
type MCPServerConfig struct {
	Name      string            `yaml:"name" json:"name"`
	Command   string            `yaml:"command" json:"command"`
	Args      []string          `yaml:"args" json:"args"`
	Env       map[string]string `yaml:"env" json:"env"`
	Transport string            `yaml:"transport" json:"transport"` // "stdio" / "sse"
}
