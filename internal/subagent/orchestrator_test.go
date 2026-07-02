package subagent

import (
    "context"
    "sync"
    "testing"
    "strings"
    "encoding/json"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/supcode/supcode/pkg"
)

func successFactory() (pkg.LLMClient, error) {
    return &MockLLMClient{
        ChatFunc: func(ctx context.Context, systemPrompt string, messages []pkg.Message, tools []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
            ch := make(chan pkg.StreamEvent, 10)
            go func() {
                defer close(ch)
                if strings.Contains(systemPrompt, "Select the most appropriate tool") {
                    ch <- pkg.StreamEvent{Type: "tool_call", ToolCall: &pkg.ToolCall{ID: "call1", Name: "echo", Params: json.RawMessage(`{}`)}}
                    ch <- pkg.StreamEvent{Type: "done"}
                    return
                }
                planJSON := `{"goal":"SubTask goal","steps":[{"id":"s1","description":"SubTask step","tool_hint":"echo"}]}`
                ch <- pkg.StreamEvent{Type: "text_delta", Delta: planJSON}
                ch <- pkg.StreamEvent{Type: "done"}
            }()
            return ch, nil
        },
    }, nil
}

func TestNewOrchestrator(t *testing.T) {
    o := NewOrchestrator(successFactory, &MockToolRegistry{}, &MockWorktreeManager{}, 2)
    require.NotNil(t, o)
    assert.Equal(t, 2, o.maxWorkers)
}

func TestNewOrchestrator_DefaultWorkers(t *testing.T) {
    o := NewOrchestrator(successFactory, &MockToolRegistry{}, &MockWorktreeManager{}, 0)
    assert.Equal(t, 3, o.maxWorkers)
}

func TestDispatch_AllSuccess(t *testing.T) {
    wtMgr := &MockWorktreeManager{}
    o := NewOrchestrator(successFactory, &MockToolRegistry{
        ListSchemasFunc: func() []pkg.ToolSchema {
            return []pkg.ToolSchema{{Name: "echo", Description: "Echoes input"}}
        },
        ExecuteFunc: func(ctx context.Context, name string, params json.RawMessage) (pkg.ToolResult, error) {
            return pkg.ToolResult{Success: true, Data: json.RawMessage(`{"result":"ok"}`)}, nil
        },
    }, wtMgr, 2)
    tasks := []pkg.SubTask{
        {ID: "task-1", Description: "First task", Context: "do something 1"},
        {ID: "task-2", Description: "Second task", Context: "do something 2"},
    }
    results, err := o.Dispatch(context.Background(), tasks)
    require.NoError(t, err)
    require.Len(t, results, 2)
    assert.True(t, results[0].Success)
    assert.True(t, results[1].Success)
}

func TestDispatch_PartialFailure(t *testing.T) {
    callCount := 0
    llmFactory := func() (pkg.LLMClient, error) {
        callCount++
        if callCount == 1 {
            return &MockLLMClient{
                ChatFunc: func(ctx context.Context, systemPrompt string, messages []pkg.Message, tools []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
                    ch := make(chan pkg.StreamEvent, 10)
                    go func() {
                        defer close(ch)
                        ch <- pkg.StreamEvent{Type: "error", Error: "worker 1 failed"}
                    }()
                    return ch, nil
                },
            }, nil
        }
        return successFactory()
    }
    o := NewOrchestrator(llmFactory, &MockToolRegistry{
        ListSchemasFunc: func() []pkg.ToolSchema {
            return []pkg.ToolSchema{{Name: "echo", Description: "Echo"}}
        },
        ExecuteFunc: func(ctx context.Context, name string, params json.RawMessage) (pkg.ToolResult, error) {
            return pkg.ToolResult{Success: true}, nil
        },
    }, &MockWorktreeManager{}, 2)
    tasks := []pkg.SubTask{
        {ID: "task-1", Context: "fail"},
        {ID: "task-2", Context: "succeed"},
    }
    results, err := o.Dispatch(context.Background(), tasks)
    require.Error(t, err)
    require.Len(t, results, 2)
    hasSuccess := false
    for _, r := range results {
        if r.Success { hasSuccess = true }
    }
    assert.True(t, hasSuccess, "at least one task should succeed")
}

func TestDispatch_CancelAll(t *testing.T) {
    wtMgr := &MockWorktreeManager{}
    o := NewOrchestrator(successFactory, &MockToolRegistry{}, wtMgr, 2)
    tasks := []pkg.SubTask{
        {ID: "task-1", Context: "do something"},
        {ID: "task-2", Context: "do something"},
    }
    ctx, cancel := context.WithCancel(context.Background())
    cancel()
    _, _ = o.Dispatch(ctx, tasks)
}

func TestDispatch_WorktreeIsolation(t *testing.T) {
    var mu sync.Mutex
    createdWorktrees := make(map[string]bool)
    wtMgr := &MockWorktreeManager{
        CreateFunc: func(ctx context.Context, agentID string, baseBranch string) (string, error) {
            mu.Lock()
            createdWorktrees[agentID] = true
            mu.Unlock()
            return "/tmp/wt/" + agentID, nil
        },
        MergeFunc: func(ctx context.Context, agentID string) error { return nil },
    }
    o := NewOrchestrator(successFactory, &MockToolRegistry{
        ListSchemasFunc: func() []pkg.ToolSchema {
            return []pkg.ToolSchema{{Name: "echo", Description: "Echoes input"}}
        },
        ExecuteFunc: func(ctx context.Context, name string, params json.RawMessage) (pkg.ToolResult, error) {
            return pkg.ToolResult{Success: true}, nil
        },
    }, wtMgr, 2)
    tasks := []pkg.SubTask{
        {ID: "task-a", Context: "do a"},
        {ID: "task-b", Context: "do b"},
    }
    results, err := o.Dispatch(context.Background(), tasks)
    require.NoError(t, err)
    require.Len(t, results, 2)
    mu.Lock()
    assert.True(t, createdWorktrees["task-a"])
    assert.True(t, createdWorktrees["task-b"])
    mu.Unlock()
}

func TestMockSubAgentOrchestrator(t *testing.T) {
    m := &MockSubAgentOrchestrator{}
    results, err := m.Dispatch(context.Background(), []pkg.SubTask{
        {ID: "t1", Context: "test"},
    })
    require.NoError(t, err)
    require.Len(t, results, 1)
    assert.True(t, results[0].Success)
    assert.Equal(t, "mock result", results[0].Output)
    m.CancelAll()
}

func TestDispatch_CancelAllAfterStart(t *testing.T) {
    o := NewOrchestrator(successFactory, &MockToolRegistry{}, &MockWorktreeManager{}, 2)
    ctx, cancel := context.WithCancel(context.Background())
    done := make(chan bool)
    go func() {
        o.Dispatch(ctx, []pkg.SubTask{{ID: "task-1", Context: "do"}})
        close(done)
    }()
    cancel()
    <-done
}

func TestCancelAll_Idempotent(t *testing.T) {
    o := NewOrchestrator(successFactory, &MockToolRegistry{}, &MockWorktreeManager{}, 2)
    o.CancelAll()
    o.CancelAll()
}

func TestSimpleSessionManager_GetErrors(t *testing.T) {
    sm := &simpleSessionManager{}
    _, err := sm.Get(context.Background(), "nonexistent")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "not found")
}
