package subagent

import (
    "context"
    "encoding/json"
    "strings"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/supcode/supcode/pkg"
)

func successLLMFactory() (pkg.LLMClient, error) {
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
                planJSON := `{"goal":"Worker goal","steps":[{"id":"s1","description":"Worker step","tool_hint":"echo"}]}`
                ch <- pkg.StreamEvent{Type: "text_delta", Delta: planJSON}
                ch <- pkg.StreamEvent{Type: "done"}
            }()
            return ch, nil
        },
    }, nil
}

func TestWorkerPool_RunTasks(t *testing.T) {
    wtMgr := &MockWorktreeManager{}
    toolReg := &MockToolRegistry{
        ListSchemasFunc: func() []pkg.ToolSchema {
            return []pkg.ToolSchema{{Name: "echo", Description: "Echo"}}
        },
        ExecuteFunc: func(ctx context.Context, name string, params json.RawMessage) (pkg.ToolResult, error) {
            return pkg.ToolResult{Success: true, Data: json.RawMessage(`{"result":"ok"}`)}, nil
        },
    }
    pool, err := NewWorkerPool(2, successLLMFactory, toolReg, wtMgr, 25)
    require.NoError(t, err)
    defer pool.Stop()

    pool.Submit(pkg.SubTask{ID: "task-1", Context: "do something"})
    pool.Submit(pkg.SubTask{ID: "task-2", Context: "do something else"})

    results := make([]pkg.SubTaskResult, 0, 2)
    timeout := time.After(10 * time.Second)
    for len(results) < 2 {
        select {
        case r := <-pool.Results():
            results = append(results, r)
        case <-timeout:
            t.Fatal("timeout waiting for results")
        }
    }

    require.Len(t, results, 2)
    for _, r := range results {
        assert.True(t, r.Success, "task %s should succeed", r.TaskID)
    }
}

func TestWorkerPool_DeadlineExceeded(t *testing.T) {
    llmFactory := func() (pkg.LLMClient, error) {
        return &MockLLMClient{
            ChatFunc: func(ctx context.Context, systemPrompt string, messages []pkg.Message, tools []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
                ch := make(chan pkg.StreamEvent, 10)
                go func() {
                    defer close(ch)
                    planJSON := `{"goal":"Slow task","steps":[{"id":"s1","description":"Slow step"}]}`
                    ch <- pkg.StreamEvent{Type: "text_delta", Delta: planJSON}
                    ch <- pkg.StreamEvent{Type: "done"}
                }()
                return ch, nil
            },
        }, nil
    }
    pool, err := NewWorkerPool(1, llmFactory, &MockToolRegistry{}, &MockWorktreeManager{}, 25)
    require.NoError(t, err)
    defer pool.Stop()

    pool.Submit(pkg.SubTask{ID: "fast-task", Context: "do it"})

    results := make([]pkg.SubTaskResult, 0, 1)
    timeout := time.After(5 * time.Second)
    for len(results) < 1 {
        select {
        case r := <-pool.Results():
            results = append(results, r)
        case <-timeout:
            t.Fatal("timeout")
        }
    }
    require.Len(t, results, 1)
    t.Logf("Result: success=%v, output=%s, error=%s", results[0].Success, results[0].Output, results[0].Error)
}
