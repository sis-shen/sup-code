package agent

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

func mockPlanLlmClient() *MockLLMClient {
    return &MockLLMClient{
        ChatFunc: func(ctx context.Context, systemPrompt string, messages []pkg.Message, tools []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
            ch := make(chan pkg.StreamEvent, 10)
            go func() {
                defer close(ch)
                if strings.Contains(systemPrompt, "Select the most appropriate tool") {
                    ch <- pkg.StreamEvent{Type: "tool_call", ToolCall: &pkg.ToolCall{
                        ID: "call1", Name: "echo", Params: json.RawMessage(`{}`),
                    }}
                    ch <- pkg.StreamEvent{Type: "done"}
                    return
                }
                planJSON := `{"goal":"Plan mode goal","steps":[{"id":"step-1","description":"Step 1 description","tool_hint":"echo"}]}`
                ch <- pkg.StreamEvent{Type: "text_delta", Delta: planJSON}
                ch <- pkg.StreamEvent{Type: "done"}
            }()
            return ch, nil
        },
    }
}

func TestRunPlan_GeneratesPlan(t *testing.T) {
    agent, err := NewAgent(AgentConfig{
        LLMClient: mockPlanLlmClient(), ToolRegistry: mockToolRegistry(),
        ContextManager: mockContextManager(), MaxIterations: 10,
    })
    require.NoError(t, err)

    result, err := agent.RunPlan(context.Background(), "test-plan-session", "do something")
    require.NoError(t, err)
    require.NotNil(t, result)
    assert.NotNil(t, result.Plan)
    assert.Len(t, result.Plan.Steps, 1)
    assert.Equal(t, "Plan mode goal", result.Plan.Goal)
    assert.Contains(t, result.Summary, "waiting for approval")
    // State should be WaitingApproval during execution, then Idle after defer
    assert.Equal(t, StateIdle, agent.getState())
}

func TestRunPlan_EmptyPlan(t *testing.T) {
    llm := &MockLLMClient{
        ChatFunc: func(ctx context.Context, systemPrompt string, messages []pkg.Message, tools []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
            ch := make(chan pkg.StreamEvent, 10)
            go func() {
                defer close(ch)
                ch <- pkg.StreamEvent{Type: "text_delta", Delta: `{"goal":"Empty","steps":[]}`}
                ch <- pkg.StreamEvent{Type: "done"}
            }()
            return ch, nil
        },
    }
    agent, err := NewAgent(AgentConfig{
        LLMClient: llm, ToolRegistry: mockToolRegistry(),
        ContextManager: mockContextManager(), MaxIterations: 10,
    })
    require.NoError(t, err)
    result, err := agent.RunPlan(context.Background(), "test-empty", "do nothing")
    require.Error(t, err)
    require.NotNil(t, result)
    assert.Contains(t, result.Error, "no steps")
}

func TestRunPlan_PlannerError(t *testing.T) {
    llm := &MockLLMClient{
        ChatFunc: func(ctx context.Context, systemPrompt string, messages []pkg.Message, tools []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
            ch := make(chan pkg.StreamEvent, 10)
            go func() {
                defer close(ch)
                ch <- pkg.StreamEvent{Type: "error", Error: "model unavailable"}
            }()
            return ch, nil
        },
    }
    agent, err := NewAgent(AgentConfig{
        LLMClient: llm, ToolRegistry: mockToolRegistry(),
        ContextManager: mockContextManager(), MaxIterations: 10,
    })
    require.NoError(t, err)
    result, err := agent.RunPlan(context.Background(), "test-err", "do thing")
    require.Error(t, err)
    assert.Contains(t, result.Error, "plan failed")
}

func TestApprovePlan_ExecutesSteps(t *testing.T) {
    plan := &pkg.Plan{
        Goal: "Test approval",
        Steps: []pkg.PlanItem{
            {ID: "s1", Description: "Step 1", ToolHint: "echo"},
        },
    }
    agent, err := NewAgent(AgentConfig{
        LLMClient: mockPlanLlmClient(), ToolRegistry: mockToolRegistry(),
        ContextManager: mockContextManager(), MaxIterations: 10,
    })
    require.NoError(t, err)
    agent.plan = plan

    result, err := agent.ApprovePlan(context.Background(), "test-approve")
    require.NoError(t, err)
    require.NotNil(t, result)
    assert.Contains(t, result.Summary, "completed")
    assert.NotNil(t, result.Plan)
}

func TestApprovePlan_NoPlan(t *testing.T) {
    agent, err := NewAgent(AgentConfig{
        LLMClient: &MockLLMClient{}, ToolRegistry: &MockToolRegistry{},
        ContextManager: mockContextManager(), MaxIterations: 10,
    })
    require.NoError(t, err)
    result, err := agent.ApprovePlan(context.Background(), "test-no-plan")
    require.Error(t, err)
    require.NotNil(t, result)
    assert.Contains(t, result.Error, "no plan")
}

func TestApprovePlan_EmptyPlan(t *testing.T) {
    agent, err := NewAgent(AgentConfig{
        LLMClient: &MockLLMClient{}, ToolRegistry: &MockToolRegistry{},
        ContextManager: mockContextManager(), MaxIterations: 10,
    })
    require.NoError(t, err)
    agent.plan = &pkg.Plan{Goal: "Empty", Steps: []pkg.PlanItem{}}
    result, err := agent.ApprovePlan(context.Background(), "test-empty-plan")
    require.NoError(t, err)
    require.NotNil(t, result)
    assert.Contains(t, result.Summary, "nothing to execute")
}

func TestApprovePlan_StepFailure(t *testing.T) {
    execAttempts := 0
    toolReg := &MockToolRegistry{
        ListSchemasFunc: func() []pkg.ToolSchema {
            return []pkg.ToolSchema{{Name: "fragile", Description: "Fragile"}}
        },
        ExecuteFunc: func(ctx context.Context, name string, params json.RawMessage) (pkg.ToolResult, error) {
            execAttempts++
            return pkg.ToolResult{Success: false, Error: "non-retryable error"}, nil
        },
    }
    selectorLLM := &MockLLMClient{
        ChatFunc: func(ctx context.Context, systemPrompt string, messages []pkg.Message, tools []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
            ch := make(chan pkg.StreamEvent, 10)
            go func() {
                defer close(ch)
                ch <- pkg.StreamEvent{Type: "tool_call", ToolCall: &pkg.ToolCall{
                    ID: "call1", Name: "fragile", Params: json.RawMessage(`{}`),
                }}
                ch <- pkg.StreamEvent{Type: "done"}
            }()
            return ch, nil
        },
    }
    llm := mockPlanLlmClient()
    agent := &Agent{
        llmClient: llm, toolRegistry: toolReg, contextManager: mockContextManager(),
        maxIterations: 10, selfCorrector: NewSelfCorrector(1, 10*time.Millisecond), state: StateIdle,
    }
    agent.planner = NewPlanner(llm)
    agent.toolSelector = NewToolSelector(selectorLLM)
    agent.plan = &pkg.Plan{
        Goal: "Fail test",
        Steps: []pkg.PlanItem{
            {ID: "s1", Description: "Fragile step", ToolHint: "fragile"},
        },
    }

    result, err := agent.ApprovePlan(context.Background(), "test-fail")
    require.Error(t, err)
    require.NotNil(t, result)
    assert.Contains(t, result.Error, "failed")
}

func TestApprovePlan_WithSessionManager(t *testing.T) {
    plan := &pkg.Plan{
        Goal: "Session test",
        Steps: []pkg.PlanItem{
            {ID: "s1", Description: "Step 1", ToolHint: "echo"},
        },
    }
    sm := &MockSessionManager{
        GetFunc: func(ctx context.Context, sessionID string) (*pkg.Session, error) {
            return &pkg.Session{ID: sessionID, Plan: plan}, nil
        },
        SetStateFunc: func(ctx context.Context, sessionID string, state pkg.LoopState) error {
            return nil
        },
        SetPlanFunc: func(ctx context.Context, sessionID string, p pkg.Plan) error {
            return nil
        },
    }
    agent, err := NewAgent(AgentConfig{
        LLMClient: mockPlanLlmClient(), ToolRegistry: mockToolRegistry(),
        ContextManager: mockContextManager(), SessionManager: sm, MaxIterations: 10,
    })
    require.NoError(t, err)
    result, err := agent.RunPlan(context.Background(), "test-sm", "do with sm")
    require.NoError(t, err)
    require.NotNil(t, result)
    assert.NotNil(t, result.Plan)
    assert.Contains(t, result.Summary, "waiting for approval")
}
