package agent

import (
    "context"
    "encoding/json"
    "sync"

    "github.com/supcode/supcode/pkg"
)

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

type MockToolRegistry struct {
    mu                sync.Mutex
    RegisterFunc      func(tool pkg.Tool) error
    UnregisterFunc    func(name string) error
    GetFunc           func(name string) (pkg.Tool, error)
    ListFunc          func() []string
    ListSchemasFunc   func() []pkg.ToolSchema
    ExecuteFunc       func(ctx context.Context, name string, params json.RawMessage) (pkg.ToolResult, error)
    RegisterHookFunc  func(hook pkg.ToolHook) error
    UnregisterHookFunc func(hookName string) error
}

func (m *MockToolRegistry) Register(tool pkg.Tool) error {
    if m.RegisterFunc != nil { return m.RegisterFunc(tool) }
    return nil
}
func (m *MockToolRegistry) Unregister(name string) error {
    if m.UnregisterFunc != nil { return m.UnregisterFunc(name) }
    return nil
}
func (m *MockToolRegistry) Get(name string) (pkg.Tool, error) {
    if m.GetFunc != nil { return m.GetFunc(name) }
    return nil, &pkg.ErrToolNotFound{ToolName: name}
}
func (m *MockToolRegistry) List() []string {
    if m.ListFunc != nil { return m.ListFunc() }
    return nil
}
func (m *MockToolRegistry) ListSchemas() []pkg.ToolSchema {
    if m.ListSchemasFunc != nil { return m.ListSchemasFunc() }
    return nil
}
func (m *MockToolRegistry) Execute(ctx context.Context, name string, params json.RawMessage) (pkg.ToolResult, error) {
    if m.ExecuteFunc != nil { return m.ExecuteFunc(ctx, name, params) }
    return pkg.ToolResult{Success: true}, nil
}
func (m *MockToolRegistry) RegisterHook(hook pkg.ToolHook) error {
    if m.RegisterHookFunc != nil { return m.RegisterHookFunc(hook) }
    return nil
}
func (m *MockToolRegistry) UnregisterHook(hookName string) error {
    if m.UnregisterHookFunc != nil { return m.UnregisterHookFunc(hookName) }
    return nil
}

type MockContextManager struct {
    BuildContextFunc   func(ctx context.Context, sessionID string) (string, []pkg.Message, error)
    AppendMessageFunc  func(ctx context.Context, sessionID string, msg pkg.Message) error
    TokenCountFunc     func(ctx context.Context, sessionID string) (int, error)
    ShouldCompressFunc func(ctx context.Context, sessionID string) (bool, error)
    CompressFunc       func(ctx context.Context, sessionID string) error
    GetMemoryCardsFunc func(ctx context.Context, sessionID string) ([]pkg.MemoryCard, error)
    ClearFunc          func(ctx context.Context, sessionID string) error
}

func (m *MockContextManager) BuildContext(ctx context.Context, sessionID string) (string, []pkg.Message, error) {
    if m.BuildContextFunc != nil { return m.BuildContextFunc(ctx, sessionID) }
    return "", nil, nil
}
func (m *MockContextManager) AppendMessage(ctx context.Context, sessionID string, msg pkg.Message) error {
    if m.AppendMessageFunc != nil { return m.AppendMessageFunc(ctx, sessionID, msg) }
    return nil
}
func (m *MockContextManager) TokenCount(ctx context.Context, sessionID string) (int, error) {
    if m.TokenCountFunc != nil { return m.TokenCountFunc(ctx, sessionID) }
    return 0, nil
}
func (m *MockContextManager) ShouldCompress(ctx context.Context, sessionID string) (bool, error) {
    if m.ShouldCompressFunc != nil { return m.ShouldCompressFunc(ctx, sessionID) }
    return false, nil
}
func (m *MockContextManager) Compress(ctx context.Context, sessionID string) error {
    if m.CompressFunc != nil { return m.CompressFunc(ctx, sessionID) }
    return nil
}
func (m *MockContextManager) GetMemoryCards(ctx context.Context, sessionID string) ([]pkg.MemoryCard, error) {
    if m.GetMemoryCardsFunc != nil { return m.GetMemoryCardsFunc(ctx, sessionID) }
    return nil, nil
}
func (m *MockContextManager) Clear(ctx context.Context, sessionID string) error {
    if m.ClearFunc != nil { return m.ClearFunc(ctx, sessionID) }
    return nil
}

var _ pkg.LLMClient = (*MockLLMClient)(nil)
var _ pkg.ToolRegistry = (*MockToolRegistry)(nil)
var _ pkg.ContextManager = (*MockContextManager)(nil)
