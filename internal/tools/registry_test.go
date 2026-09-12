package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supcode/supcode/pkg"
)

func TestRegistry_Register_Get(t *testing.T) {
	permEng := &MockPermissionEngine{CheckFunc: func(ctx context.Context, action pkg.Action) (pkg.Decision, error) {
		return pkg.DecisionAllow, nil
	}}
	r := NewRegistry(permEng)

	tool := &MockTool{
		NameFunc: func() string { return "test_tool" },
	}

	err := r.Register(tool)
	require.NoError(t, err)

	err = r.Register(tool)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	got, err := r.Get("test_tool")
	require.NoError(t, err)
	assert.Equal(t, tool, got)

	_, err = r.Get("nonexistent")
	require.Error(t, err)
	assert.IsType(t, &pkg.ErrToolNotFound{}, err)
}

func TestRegistry_Register_NilTool(t *testing.T) {
	r := NewRegistry(nil)
	err := r.Register(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil tool")
}

func TestRegistry_Register_EmptyName(t *testing.T) {
	r := NewRegistry(nil)
	tool := &MockTool{NameFunc: func() string { return "" }}
	err := r.Register(tool)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestRegistry_Unregister(t *testing.T) {
	r := NewRegistry(nil)
	tool := &MockTool{NameFunc: func() string { return "test_tool" }}
	_ = r.Register(tool)

	err := r.Unregister("test_tool")
	require.NoError(t, err)

	err = r.Unregister("nonexistent")
	require.Error(t, err)
	assert.IsType(t, &pkg.ErrToolNotFound{}, err)
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry(nil)
	assert.Empty(t, r.List())

	_ = r.Register(&MockTool{NameFunc: func() string { return "b" }})
	_ = r.Register(&MockTool{NameFunc: func() string { return "a" }})

	names := r.List()
	assert.Equal(t, []string{"a", "b"}, names)
}

func TestRegistry_ListSchemas(t *testing.T) {
	r := NewRegistry(nil)
	_ = r.Register(&MockTool{
		NameFunc: func() string { return "test" },
		SchemaFunc: func() pkg.ToolSchema {
			return pkg.ToolSchema{Name: "test", Parameters: json.RawMessage(`{"type":"object"}`)}
		},
	})

	schemas := r.ListSchemas()
	assert.Len(t, schemas, 1)
	assert.Equal(t, "test", schemas[0].Name)
}

func TestRegistry_Execute_Success(t *testing.T) {
	permEng := &MockPermissionEngine{CheckFunc: func(ctx context.Context, action pkg.Action) (pkg.Decision, error) {
		assert.Equal(t, "tool", action.Type)
		assert.Equal(t, "test_tool", action.ToolName)
		return pkg.DecisionAllow, nil
	}}
	r := NewRegistry(permEng)

	executed := false
	tool := &MockTool{
		NameFunc: func() string { return "test_tool" },
		ExecuteFunc: func(ctx context.Context, params json.RawMessage) (pkg.ToolResult, error) {
			executed = true
			return pkg.ToolResult{Success: true, Data: json.RawMessage(`{"result":"ok"}`)}, nil
		},
	}
	_ = r.Register(tool)

	result, err := r.Execute(context.Background(), "test_tool", json.RawMessage(`{"param":"value"}`))
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, executed)
}

func TestRegistry_Execute_PermissionDenied(t *testing.T) {
	permEng := &MockPermissionEngine{CheckFunc: func(ctx context.Context, action pkg.Action) (pkg.Decision, error) {
		return pkg.DecisionDeny, nil
	}}
	r := NewRegistry(permEng)

	tool := &MockTool{
		NameFunc: func() string { return "test_tool" },
	}
	_ = r.Register(tool)

	_, err := r.Execute(context.Background(), "test_tool", json.RawMessage(`{}`))
	require.Error(t, err)
	assert.IsType(t, &pkg.ErrPermissionDenied{}, err)
}

func TestRegistry_Execute_ToolNotFound(t *testing.T) {
	r := NewRegistry(nil)
	_, err := r.Execute(context.Background(), "nonexistent", json.RawMessage(`{}`))
	require.Error(t, err)
	assert.IsType(t, &pkg.ErrToolNotFound{}, err)
}

func TestRegistry_Execute_Hooks(t *testing.T) {
	permEng := &MockPermissionEngine{CheckFunc: func(ctx context.Context, action pkg.Action) (pkg.Decision, error) {
		return pkg.DecisionAllow, nil
	}}
	r := NewRegistry(permEng)

	var beforeCalled, afterCalled bool
	hook := &MockToolHook{
		NameFunc: func() string { return "test_hook" },
		BeforeToolFunc: func(ctx context.Context, toolName string, params json.RawMessage) (json.RawMessage, error) {
			beforeCalled = true
			assert.Equal(t, "test_tool", toolName)
			return nil, nil
		},
		AfterToolFunc: func(ctx context.Context, toolName string, params json.RawMessage, result pkg.ToolResult) error {
			afterCalled = true
			return nil
		},
	}

	_ = r.RegisterHook(hook)

	tool := &MockTool{
		NameFunc: func() string { return "test_tool" },
		ExecuteFunc: func(ctx context.Context, params json.RawMessage) (pkg.ToolResult, error) {
			return pkg.ToolResult{Success: true, Data: json.RawMessage(`{}`)}, nil
		},
	}
	_ = r.Register(tool)

	_, err := r.Execute(context.Background(), "test_tool", json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.True(t, beforeCalled)
	assert.True(t, afterCalled)
}

func TestRegistry_RegisterHook_Duplicate(t *testing.T) {
	r := NewRegistry(nil)
	_ = r.RegisterHook(&MockToolHook{NameFunc: func() string { return "hook1" }})
	err := r.RegisterHook(&MockToolHook{NameFunc: func() string { return "hook1" }})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestRegistry_UnregisterHook(t *testing.T) {
	r := NewRegistry(nil)
	_ = r.RegisterHook(&MockToolHook{NameFunc: func() string { return "hook1" }})
	err := r.UnregisterHook("hook1")
	require.NoError(t, err)

	err = r.UnregisterHook("hook1")
	require.Error(t, err)
}

func TestRegistry_ContextCancellation(t *testing.T) {
	r := NewRegistry(nil)
	tool := &MockTool{
		NameFunc: func() string { return "slow_tool" },
		ExecuteFunc: func(ctx context.Context, params json.RawMessage) (pkg.ToolResult, error) {
			<-ctx.Done()
			return pkg.ToolResult{Success: false, Error: "canceled"}, ctx.Err()
		},
	}
	_ = r.Register(tool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := r.Execute(ctx, "slow_tool", json.RawMessage(`{}`))
	require.Error(t, err)
}

func TestRegistry_Execute_BeforeToolHook_TimesOut(t *testing.T) {
	r := NewRegistryWithHookTimeout(nil, 50*time.Millisecond)

	hook := &MockToolHook{
		NameFunc: func() string { return "slow_hook" },
		BeforeToolFunc: func(ctx context.Context, toolName string, params json.RawMessage) (json.RawMessage, error) {
			<-ctx.Done() // 模拟挂死的钩子
			return nil, ctx.Err()
		},
	}
	_ = r.RegisterHook(hook)

	executed := false
	_ = r.Register(&MockTool{
		NameFunc: func() string { return "test_tool" },
		ExecuteFunc: func(ctx context.Context, params json.RawMessage) (pkg.ToolResult, error) {
			executed = true
			return pkg.ToolResult{Success: true}, nil
		},
	})

	done := make(chan struct{})
	var execErr error
	go func() {
		_, execErr = r.Execute(context.Background(), "test_tool", json.RawMessage(`{}`))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Execute blocked on a hung hook; per-hook timeout not enforced")
	}
	require.Error(t, execErr)
	assert.False(t, executed, "tool must not run when its BeforeTool hook times out")
}

func TestRegistry_Execute_ContextCancelled_DuringHookChain(t *testing.T) {
	r := NewRegistry(nil)

	hookCalled := false
	_ = r.RegisterHook(&MockToolHook{
		NameFunc: func() string { return "h" },
		BeforeToolFunc: func(ctx context.Context, toolName string, params json.RawMessage) (json.RawMessage, error) {
			hookCalled = true
			return nil, nil
		},
	})

	executed := false
	_ = r.Register(&MockTool{
		NameFunc: func() string { return "test_tool" },
		ExecuteFunc: func(ctx context.Context, params json.RawMessage) (pkg.ToolResult, error) {
			executed = true
			return pkg.ToolResult{Success: true}, nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := r.Execute(ctx, "test_tool", json.RawMessage(`{}`))
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.False(t, hookCalled, "hook chain should abort on cancelled context")
	assert.False(t, executed, "tool must not run on cancelled context")
}

func TestRegistry_Execute_AfterToolHookError_DoesNotAffectResult(t *testing.T) {
	r := NewRegistry(nil)
	_ = r.RegisterHook(&MockToolHook{
		NameFunc: func() string { return "bad_after" },
		AfterToolFunc: func(ctx context.Context, toolName string, params json.RawMessage, result pkg.ToolResult) error {
			return errors.New("after hook boom")
		},
	})
	_ = r.Register(&MockTool{
		NameFunc: func() string { return "test_tool" },
		ExecuteFunc: func(ctx context.Context, params json.RawMessage) (pkg.ToolResult, error) {
			return pkg.ToolResult{Success: true, Data: json.RawMessage(`{"ok":true}`)}, nil
		},
	})

	result, err := r.Execute(context.Background(), "test_tool", json.RawMessage(`{}`))
	require.NoError(t, err, "AfterTool hook errors must not propagate to the caller")
	assert.True(t, result.Success)
}

func TestNewRegistryWithHookTimeout_DefaultOnNonPositive(t *testing.T) {
	r := NewRegistryWithHookTimeout(nil, 0)
	assert.Equal(t, defaultHookTimeout, r.hookTimeout)
}

type MockPermissionEngine struct {
	CheckFunc      func(ctx context.Context, action pkg.Action) (pkg.Decision, error)
	AddRuleFunc    func(rule pkg.PermissionRule) error
	RemoveRuleFunc func(ruleID string) error
	ListRulesFunc  func() []pkg.PermissionRule
	LogActionFunc  func(ctx context.Context, action pkg.Action, decision pkg.Decision, result string) error
}

func (m *MockPermissionEngine) Check(ctx context.Context, action pkg.Action) (pkg.Decision, error) {
	if m.CheckFunc != nil {
		return m.CheckFunc(ctx, action)
	}
	return pkg.DecisionAllow, nil
}
func (m *MockPermissionEngine) AddRule(rule pkg.PermissionRule) error {
	if m.AddRuleFunc != nil {
		return m.AddRuleFunc(rule)
	}
	return nil
}
func (m *MockPermissionEngine) RemoveRule(ruleID string) error {
	if m.RemoveRuleFunc != nil {
		return m.RemoveRuleFunc(ruleID)
	}
	return nil
}
func (m *MockPermissionEngine) ListRules() []pkg.PermissionRule {
	if m.ListRulesFunc != nil {
		return m.ListRulesFunc()
	}
	return nil
}
func (m *MockPermissionEngine) LogAction(ctx context.Context, action pkg.Action, decision pkg.Decision, result string) error {
	if m.LogActionFunc != nil {
		return m.LogActionFunc(ctx, action, decision, result)
	}
	return nil
}

type MockToolHook struct {
	NameFunc       func() string
	BeforeToolFunc func(ctx context.Context, toolName string, params json.RawMessage) (json.RawMessage, error)
	AfterToolFunc  func(ctx context.Context, toolName string, params json.RawMessage, result pkg.ToolResult) error
}

func (m *MockToolHook) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return ""
}
func (m *MockToolHook) BeforeTool(ctx context.Context, toolName string, params json.RawMessage) (json.RawMessage, error) {
	if m.BeforeToolFunc != nil {
		return m.BeforeToolFunc(ctx, toolName, params)
	}
	return params, nil
}
func (m *MockToolHook) AfterTool(ctx context.Context, toolName string, params json.RawMessage, result pkg.ToolResult) error {
	if m.AfterToolFunc != nil {
		return m.AfterToolFunc(ctx, toolName, params, result)
	}
	return nil
}

var (
	_ pkg.PermissionEngine = (*MockPermissionEngine)(nil)
	_ pkg.ToolHook         = (*MockToolHook)(nil)
)

func init() { _ = errors.New }
