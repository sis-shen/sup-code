package pkg

import (
	"context"
	"encoding/json"
)

// ─────────────────────────────────────────────────────────────
// 4. 交互层接口（Interaction Layer）
// ─────────────────────────────────────────────────────────────

// InteractionService 交互层服务接口
// 引擎层通过此接口与用户交互，不直接依赖 Bubble Tea 或终端细节
type InteractionService interface {
	// StreamResponse 将 LLM 的流式响应推送给用户
	// sessionID: 当前会话 ID，用于多会话场景路由
	// stream: LLM 返回的事件流，交互层负责渲染
	StreamResponse(ctx context.Context, sessionID string, stream <-chan StreamEvent) error

	// RequestConfirmation 请求用户确认（权限弹窗等）
	// 返回用户的选择：true=同意, false=拒绝
	RequestConfirmation(ctx context.Context, sessionID string, prompt ConfirmPrompt) (bool, error)

	// ReadInput 阻塞等待用户输入一行文本
	ReadInput(ctx context.Context, sessionID string) (string, error)

	// Notify 向用户发送一条通知（非阻塞）
	Notify(ctx context.Context, sessionID string, level NotifyLevel, message string)

	// HandleCommand 处理 / 开头的斜杠命令
	// 返回命令是否被处理及处理结果
	HandleCommand(ctx context.Context, sessionID string, input string) (CommandResult, error)
}

// SessionManager 多轮对话会话管理器
type SessionManager interface {
	// Create 创建新会话
	Create(ctx context.Context, title string) (*Session, error)

	// Get 获取指定会话
	Get(ctx context.Context, sessionID string) (*Session, error)

	// List 列出所有会话
	List(ctx context.Context) ([]*Session, error)

	// AppendMessage 向会话追加一条消息
	AppendMessage(ctx context.Context, sessionID string, msg Message) error

	// SetState 更新会话状态
	SetState(ctx context.Context, sessionID string, state LoopState) error

	// SetPlan 更新会话计划
	SetPlan(ctx context.Context, sessionID string, plan Plan) error

	// Delete 删除会话
	Delete(ctx context.Context, sessionID string) error

	// Close 关闭会话（持久化并释放内存缓存）
	Close(ctx context.Context, sessionID string) error
}

// ─────────────────────────────────────────────────────────────
// 5. 引擎层接口（Engine Layer）
// ─────────────────────────────────────────────────────────────

// Agent 引擎层主入口
// cmd/supcode/main.go 持有此接口实例，启动交互循环
type Agent interface {
	// Run 执行一次完整的 Agent Loop
	// input: 用户输入的自然语言指令
	// 返回最终结果和可能的错误
	Run(ctx context.Context, sessionID string, input string) (*AgentResult, error)

	// RunPlan 先出计划，等待确认后再执行（Plan Mode）
	RunPlan(ctx context.Context, sessionID string, input string) (*AgentResult, error)

	// ApprovePlan 用户确认计划后继续执行
	ApprovePlan(ctx context.Context, sessionID string) (*AgentResult, error)

	// GetSession 获取当前会话状态
	GetSession(ctx context.Context, sessionID string) (*Session, error)
}

// LLMClient 大语言模型客户端抽象
// 每个供应商（OpenAI / Claude / DeepSeek / 本地模型）独立实现
type LLMClient interface {
	// Chat 发起对话，返回流式事件 channel
	// systemPrompt: 系统提示词
	// messages: 对话历史
	// tools: 可用的工具 Schema 列表（供 function calling）
	Chat(ctx context.Context, systemPrompt string, messages []Message, tools []ToolSchema) (<-chan StreamEvent, error)

	// Models 返回可用模型列表
	Models(ctx context.Context) ([]ModelInfo, error)

	// ProviderName 返回供应商名称
	ProviderName() string
}

// Planner 任务规划器
type Planner interface {
	// Plan 根据用户输入生成任务计划
	// 内部调用 LLM，返回结构化 Plan
	Plan(ctx context.Context, systemPrompt string, messages []Message, tools []ToolSchema) (*Plan, error)
}

// ToolSelector 工具选择器（可替换策略）
type ToolSelector interface {
	// Select 根据步骤描述选择最合适的工具
	// 返回工具名和参数
	Select(ctx context.Context, step PlanItem, availableTools []ToolSchema) (string, json.RawMessage, error)
}

// SelfCorrector 自我修正器
type SelfCorrector interface {
	// Retry 根据错误信息决定重试策略
	// 返回修正后的参数，或返回错误表示不可恢复
	Retry(ctx context.Context, toolName string, params json.RawMessage, prevErr error) (json.RawMessage, error)

	// MaxRetries 返回最大重试次数
	MaxRetries() int
}

// SubAgentOrchestrator 子 Agent 编排器（Phase 3）
type SubAgentOrchestrator interface {
	// Dispatch 将子任务分发给 SubAgent 并行执行
	// tasks: 子任务列表
	// 返回每个子任务的结果
	Dispatch(ctx context.Context, tasks []SubTask) ([]SubTaskResult, error)

	// CancelAll 取消所有正在运行的 SubAgent
	CancelAll()
}

// ─────────────────────────────────────────────────────────────
// 6. 工具层接口（Tool Layer）
// ─────────────────────────────────────────────────────────────

// Tool 工具接口
// 内置工具、MCP 外部工具、Skill 工具均实现此接口
type Tool interface {
	// Name 工具名，唯一标识
	Name() string

	// Description 工具描述，供 LLM 理解用途
	Description() string

	// Schema 工具的参数 JSON Schema，供 LLM function calling
	Schema() ToolSchema

	// Execute 执行工具
	Execute(ctx context.Context, params json.RawMessage) (ToolResult, error)
}

// ToolRegistry 工具注册中心
// 引擎层持有此接口，通过它发现和调用工具
type ToolRegistry interface {
	// Register 注册一个工具
	Register(tool Tool) error

	// Unregister 移除一个工具
	Unregister(name string) error

	// Get 按名称获取工具
	Get(name string) (Tool, error)

	// List 列出所有已注册的工具名
	List() []string

	// ListSchemas 列出所有工具的 Schema（供 LLM function calling）
	ListSchemas() []ToolSchema

	// Execute 按名称执行工具（会经过 Hook 链和权限检查）
	Execute(ctx context.Context, name string, params json.RawMessage) (ToolResult, error)

	// RegisterHook 注册一个全局钩子
	RegisterHook(hook ToolHook) error

	// UnregisterHook 移除一个钩子
	UnregisterHook(hookName string) error
}

// ToolHook 工具钩子
type ToolHook interface {
	// Name 钩子名
	Name() string

	// BeforeTool 工具执行前调用
	// 返回 nil 继续执行；返回 error 中断执行
	BeforeTool(ctx context.Context, toolName string, params json.RawMessage) (json.RawMessage, error)

	// AfterTool 工具执行后调用
	AfterTool(ctx context.Context, toolName string, params json.RawMessage, result ToolResult) error
}

// MCPClient MCP 协议客户端（Phase 3）
type MCPClient interface {
	// Connect 连接一个 MCP 服务
	Connect(ctx context.Context, config MCPServerConfig) error

	// Disconnect 断开一个 MCP 服务
	Disconnect(ctx context.Context, serverName string) error

	// ListServers 列出已连接的 MCP 服务
	ListServers() []string

	// ListTools 列出指定 MCP 服务的工具
	ListTools(ctx context.Context, serverName string) ([]ToolSchema, error)

	// ExecuteTool 执行 MCP 工具
	ExecuteTool(ctx context.Context, serverName string, toolName string, params json.RawMessage) (ToolResult, error)

	// Close 断开所有连接
	Close() error
}

// ─────────────────────────────────────────────────────────────
// 7. 记忆层接口（Memory Layer）
// ─────────────────────────────────────────────────────────────

// ContextManager 上下文管理器
// 引擎层在每次 LLM 调用前通过此接口获取构建好的上下文
type ContextManager interface {
	// BuildContext 构建送给 LLM 的完整消息列表
	// 包含：系统提示词 + 记忆卡片 + 近期完整消息
	BuildContext(ctx context.Context, sessionID string) (systemPrompt string, messages []Message, err error)

	// AppendMessage 追加一条消息并更新 Token 计数
	AppendMessage(ctx context.Context, sessionID string, msg Message) error

	// TokenCount 返回当前会话的 Token 估算数
	TokenCount(ctx context.Context, sessionID string) (int, error)

	// ShouldCompress 判断是否需要触发压缩
	ShouldCompress(ctx context.Context, sessionID string) (bool, error)

	// Compress 触发上下文压缩（异步）
	// 返回压缩是否成功启动
	Compress(ctx context.Context, sessionID string) error

	// GetMemoryCards 获取当前会话的记忆卡片
	GetMemoryCards(ctx context.Context, sessionID string) ([]MemoryCard, error)

	// Clear 清除会话上下文
	Clear(ctx context.Context, sessionID string) error
}

// MemoryStore 长期记忆存储接口
type MemoryStore interface {
	// Save 保存一条记忆
	Save(ctx context.Context, entry MemoryEntry) error

	// Search 语义检索相关记忆
	Search(ctx context.Context, query string, scope string, limit int) ([]MemoryEntry, error)

	// GetByCategory 按分类获取记忆
	GetByCategory(ctx context.Context, scope string, category string) ([]MemoryEntry, error)

	// Delete 删除一条记忆
	Delete(ctx context.Context, id string) error

	// UpdateEmbedding 更新一条记忆的嵌入向量
	UpdateEmbedding(ctx context.Context, id string, embedding []float32) error

	// Close 关闭存储连接
	Close() error
}

// EmbeddingProvider 嵌入向量生成器（可替换后端）
type EmbeddingProvider interface {
	// Generate 为文本生成嵌入向量
	Generate(ctx context.Context, text string) ([]float32, error)

	// BatchGenerate 批量生成嵌入向量
	BatchGenerate(ctx context.Context, texts []string) ([][]float32, error)
}

// ─────────────────────────────────────────────────────────────
// 8. 安全层接口（Security Layer）
// ─────────────────────────────────────────────────────────────

// PermissionEngine 权限引擎
// 嵌入在 ToolRegistry.Execute 的调用链中
type PermissionEngine interface {
	// Check 检查操作是否允许
	// 返回 allow / ask（需回调交互层确认）/ deny
	Check(ctx context.Context, action Action) (Decision, error)

	// AddRule 动态添加权限规则
	AddRule(rule PermissionRule) error

	// RemoveRule 移除权限规则
	RemoveRule(ruleID string) error

	// ListRules 列出所有规则
	ListRules() []PermissionRule

	// LogAction 记录操作审计日志
	LogAction(ctx context.Context, action Action, decision Decision, result string) error
}

// ConfirmCallback 确认回调函数签名
// 安全层不直接依赖交互层，而是通过此回调
// 引擎层在初始化时注入，内部调用 InteractionService.RequestConfirmation
type ConfirmCallback func(ctx context.Context, sessionID string, prompt ConfirmPrompt) (bool, error)

// WorktreeManager Git Worktree 管理器（Phase 3）
type WorktreeManager interface {
	// Create 为指定 Agent 创建独立 Worktree
	Create(ctx context.Context, agentID string, baseBranch string) (worktreePath string, err error)

	// Merge 合并 Worktree 的变更回主分支
	Merge(ctx context.Context, agentID string) error

	// Abandon 放弃 Worktree 的变更并清理
	Abandon(ctx context.Context, agentID string) error

	// ListActive 列出所有活跃的 Worktree
	ListActive(ctx context.Context) ([]WorktreeInfo, error)

	// Cleanup 清理所有已合并或废弃的 Worktree
	Cleanup(ctx context.Context) error
}

// ─────────────────────────────────────────────────────────────
// 9. 配置接口
// ─────────────────────────────────────────────────────────────

// Config 配置管理器
type Config interface {
	// Load 加载配置（合并所有来源）
	Load() error

	// Get 获取指定 key 的值
	Get(key string) any

	// GetString/GetInt/GetBool 类型安全的取值
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool

	// Set 设置配置值（运行时覆盖）
	Set(key string, value any) error

	// Save 持久化当前配置
	Save() error

	// AllSettings 返回所有配置的 map
	AllSettings() map[string]any
}
