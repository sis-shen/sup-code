package subagent

import (
    "context"
    "encoding/json"
    "sync"

    "github.com/supcode/supcode/pkg"
)

// MockLLMClient implements pkg.LLMClient for testing.
type MockLLMClient struct {
    mu           sync.Mutex
    ChatFunc     func(ctx context.Context, systemPrompt string, messages []pkg.Message, tools []pkg.ToolSchema) (<-chan pkg.StreamEvent, error)
    ModelsFunc   func(ctx context.Context) ([]pkg.ModelInfo, error)
    ProviderFunc func() string
}

func (m *MockLLMClient) Chat(ctx context.Context, systemPrompt string, messages []pkg.Message, tools []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    if m.ChatFunc != nil { return m.ChatFunc(ctx, systemPrompt, messages, tools) }
    ch := make(chan pkg.StreamEvent, 1)
    ch <- pkg.StreamEvent{Type: "done"}
    close(ch)
    return ch, nil
}
func (m *MockLLMClient) Models(ctx context.Context) ([]pkg.ModelInfo, error) {
    if m.ModelsFunc != nil { return m.ModelsFunc(ctx) }
    return nil, nil
}
func (m *MockLLMClient) ProviderName() string {
    if m.ProviderFunc != nil { return m.ProviderFunc() }
    return "mock"
}

// MockToolRegistry implements pkg.ToolRegistry for testing.
type MockToolRegistry struct {
    mu              sync.Mutex
    ListSchemasFunc func() []pkg.ToolSchema
    ExecuteFunc     func(ctx context.Context, name string, params json.RawMessage) (pkg.ToolResult, error)
}

func (m *MockToolRegistry) Register(tool pkg.Tool) error { return nil }
func (m *MockToolRegistry) Unregister(name string) error { return nil }
func (m *MockToolRegistry) Get(name string) (pkg.Tool, error) { return nil, &pkg.ErrToolNotFound{ToolName: name} }
func (m *MockToolRegistry) List() []string { return nil }
func (m *MockToolRegistry) ListSchemas() []pkg.ToolSchema {
    if m.ListSchemasFunc != nil { return m.ListSchemasFunc() }
    return nil
}
func (m *MockToolRegistry) Execute(ctx context.Context, name string, params json.RawMessage) (pkg.ToolResult, error) {
    if m.ExecuteFunc != nil { return m.ExecuteFunc(ctx, name, params) }
    return pkg.ToolResult{Success: true}, nil
}
func (m *MockToolRegistry) RegisterHook(hook pkg.ToolHook) error { return nil }
func (m *MockToolRegistry) UnregisterHook(hookName string) error { return nil }

// MockWorktreeManager implements pkg.WorktreeManager for testing.
type MockWorktreeManager struct {
    mu            sync.Mutex
    CreateFunc    func(ctx context.Context, agentID string, baseBranch string) (string, error)
    MergeFunc     func(ctx context.Context, agentID string) error
    AbandonFunc   func(ctx context.Context, agentID string) error
    ListActiveFunc func(ctx context.Context) ([]pkg.WorktreeInfo, error)
    CleanupFunc   func(ctx context.Context) error
}

func (m *MockWorktreeManager) Create(ctx context.Context, agentID string, baseBranch string) (string, error) {
    if m.CreateFunc != nil { return m.CreateFunc(ctx, agentID, baseBranch) }
    return "/tmp/worktree/" + agentID, nil
}
func (m *MockWorktreeManager) Merge(ctx context.Context, agentID string) error {
    if m.MergeFunc != nil { return m.MergeFunc(ctx, agentID) }
    return nil
}
func (m *MockWorktreeManager) Abandon(ctx context.Context, agentID string) error {
    if m.AbandonFunc != nil { return m.AbandonFunc(ctx, agentID) }
    return nil
}
func (m *MockWorktreeManager) ListActive(ctx context.Context) ([]pkg.WorktreeInfo, error) {
    if m.ListActiveFunc != nil { return m.ListActiveFunc(ctx) }
    return nil, nil
}
func (m *MockWorktreeManager) Cleanup(ctx context.Context) error {
    if m.CleanupFunc != nil { return m.CleanupFunc(ctx) }
    return nil
}

// MockSubAgentOrchestrator implements pkg.SubAgentOrchestrator for testing.
type MockSubAgentOrchestrator struct {
    DispatchFunc func(ctx context.Context, tasks []pkg.SubTask) ([]pkg.SubTaskResult, error)
    CancelAllFunc func()
}

func (m *MockSubAgentOrchestrator) Dispatch(ctx context.Context, tasks []pkg.SubTask) ([]pkg.SubTaskResult, error) {
    if m.DispatchFunc != nil { return m.DispatchFunc(ctx, tasks) }
    results := make([]pkg.SubTaskResult, len(tasks))
    for i, t := range tasks {
        results[i] = pkg.SubTaskResult{TaskID: t.ID, Success: true, Output: "mock result"}
    }
    return results, nil
}
func (m *MockSubAgentOrchestrator) CancelAll() {
    if m.CancelAllFunc != nil { m.CancelAllFunc() }
}
