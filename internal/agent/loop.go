package agent

import (
    "context"
    "fmt"
    "log"
    "strings"
    "sync"
    "time"

    "github.com/supcode/supcode/pkg"
)

// LoopState represents the current state of the Agent loop.
type LoopState string

const (
    StateIdle      LoopState = "idle"
    StatePlanning  LoopState = "planning"
    StateActing    LoopState = "acting"
    StateObserving LoopState = "observing"
    StateCompleted LoopState = "completed"
    StateError     LoopState = "error"
)

// Agent implements the pkg.Agent interface using a ReAct state machine loop.
type Agent struct {
    llmClient      pkg.LLMClient
    planner        pkg.Planner
    toolSelector   pkg.ToolSelector
    toolRegistry   pkg.ToolRegistry
    selfCorrector  pkg.SelfCorrector
    contextManager pkg.ContextManager
    maxIterations  int
    sessionManager pkg.SessionManager

    mu    sync.RWMutex
    state LoopState
    plan  *pkg.Plan
}

// AgentConfig holds the configuration for creating a new Agent.
type AgentConfig struct {
    LLMClient      pkg.LLMClient
    ToolRegistry   pkg.ToolRegistry
    ContextManager pkg.ContextManager
    SessionManager pkg.SessionManager
    MaxIterations  int
}

// NewAgent creates a new Agent with the given configuration.
func NewAgent(cfg AgentConfig) (*Agent, error) {
    if cfg.MaxIterations <= 0 {
        cfg.MaxIterations = 25
    }
    agent := &Agent{
        llmClient:      cfg.LLMClient,
        toolRegistry:   cfg.ToolRegistry,
        contextManager: cfg.ContextManager,
        sessionManager: cfg.SessionManager,
        maxIterations:  cfg.MaxIterations,
        selfCorrector:  DefaultSelfCorrector(),
        state:          StateIdle,
    }
    agent.planner = NewPlanner(cfg.LLMClient)
    agent.toolSelector = NewToolSelector(cfg.LLMClient)
    return agent, nil
}

// Run executes a complete Agent loop for the given input.
func (a *Agent) Run(ctx context.Context, sessionID string, input string) (*pkg.AgentResult, error) {
    defer a.setState(StateIdle)

    // 1. Build context
    systemPrompt, messages, err := a.contextManager.BuildContext(ctx, sessionID)
    if err != nil {
        a.setState(StateError)
        return &pkg.AgentResult{Error: fmt.Sprintf("build context: %v", err)}, fmt.Errorf("build context: %w", err)
    }

    // Append user message
    userMsg := pkg.Message{
        Role:      pkg.RoleUser,
        Content:   input,
        Timestamp: time.Now(),
    }
    if err := a.contextManager.AppendMessage(ctx, sessionID, userMsg); err != nil {
        a.setState(StateError)
        return &pkg.AgentResult{Error: fmt.Sprintf("append message: %v", err)}, err
    }

    // 2. Get available tools
    toolSchemas := a.toolRegistry.ListSchemas()

    // 3. Generate plan
    a.setState(StatePlanning)
    plan, err := a.planner.Plan(ctx, systemPrompt, append(messages, userMsg), toolSchemas)
    if err != nil {
        a.setState(StateError)
        return &pkg.AgentResult{Error: fmt.Sprintf("plan failed: %v", err)}, fmt.Errorf("plan: %w", err)
    }
    if len(plan.Steps) == 0 {
        a.setState(StateError)
        return &pkg.AgentResult{Error: "plan has no steps"}, fmt.Errorf("empty plan")
    }

    a.mu.Lock()
    a.plan = plan
    a.mu.Unlock()
    if a.sessionManager != nil {
        a.sessionManager.SetPlan(ctx, sessionID, *plan)
        a.sessionManager.SetState(ctx, sessionID, pkg.StatePlanning)
    }

    // 4. Execute plan steps
    a.setState(StateActing)
    var stepResults []string

    for stepIndex, step := range plan.Steps {
        select {
        case <-ctx.Done():
            a.setState(StateError)
            return &pkg.AgentResult{
                Plan:    plan,
                Summary: fmt.Sprintf("execution cancelled after %d/%d steps", stepIndex+1, len(plan.Steps)),
                Error:   ctx.Err().Error(),
            }, ctx.Err()
        default:
        }

        if stepIndex >= a.maxIterations {
            a.setState(StateCompleted)
            return &pkg.AgentResult{
                Plan:    plan,
                Summary: fmt.Sprintf("completed %d/%d steps (max iterations reached)", stepIndex, len(plan.Steps)),
            }, nil
        }

        result, err := a.executeStep(ctx, sessionID, systemPrompt, step, toolSchemas, stepIndex+1)
        if err != nil {
            if isRetryableErr(err) {
                corrected, retryErr := a.retryStep(ctx, sessionID, step, toolSchemas, stepIndex+1, err)
                if retryErr != nil {
                    a.setState(StateError)
                    return &pkg.AgentResult{
                        Plan:    plan,
                        Summary: fmt.Sprintf("step %d failed after retries: %v", stepIndex+1, retryErr),
                        Error:   retryErr.Error(),
                    }, retryErr
                }
                stepResults = append(stepResults, corrected)
                continue
            }
            a.setState(StateError)
            return &pkg.AgentResult{
                Plan:    plan,
                Summary: fmt.Sprintf("step %d failed: %v", stepIndex+1, err),
                Error:   err.Error(),
            }, err
        }
        stepResults = append(stepResults, result)
    }

    // 5. Build summary
    a.setState(StateObserving)
    summary := a.buildSummary(plan, stepResults)
    a.setState(StateCompleted)
    if a.sessionManager != nil {
        a.sessionManager.SetState(ctx, sessionID, pkg.StateCompleted)
    }
    return &pkg.AgentResult{
        Plan:    plan,
        Summary: summary,
    }, nil
}

// executeStep executes a single step of the plan.
func (a *Agent) executeStep(ctx context.Context, sessionID, systemPrompt string, step pkg.PlanItem, tools []pkg.ToolSchema, stepNum int) (string, error) {
    toolName, params, err := a.toolSelector.Select(ctx, step, tools)
    if err != nil {
        return "", fmt.Errorf("select tool for step %d: %w", stepNum, err)
    }

    a.setState(StateActing)
    result, err := a.toolRegistry.Execute(ctx, toolName, params)
    if err != nil {
        return "", fmt.Errorf("execute tool %s: %w", toolName, err)
    }

    toolResultMsg := pkg.Message{
        Role:      pkg.RoleTool,
        Content:   fmt.Sprintf("Tool %s result: %s", toolName, string(result.Data)),
        ToolID:    toolName,
        Timestamp: time.Now(),
    }
    if err := a.contextManager.AppendMessage(ctx, sessionID, toolResultMsg); err != nil {
        log.Printf("append tool result message: %v", err)
    }

    a.setState(StateObserving)
    if !result.Success {
        errMsg := result.Error
        if errMsg == "" {
            errMsg = fmt.Sprintf("tool %s returned failure", toolName)
        }
        return "", fmt.Errorf("tool %s failed: %s", toolName, errMsg)
    }
    return fmt.Sprintf("Step %d (%s): completed successfully", stepNum, step.Description), nil
}

// retryStep retries a failed step using the SelfCorrector.
func (a *Agent) retryStep(ctx context.Context, sessionID string, step pkg.PlanItem, tools []pkg.ToolSchema, stepNum int, originalErr error) (string, error) {
    maxRetries := a.selfCorrector.MaxRetries()
    if maxRetries <= 0 {
        maxRetries = 3
    }
    lastErr := originalErr

    for attempt := 1; attempt <= maxRetries; attempt++ {
        if err := retryBackoff(ctx, attempt); err != nil {
            return "", err
        }

        toolName, params, err := a.toolSelector.Select(ctx, step, tools)
        if err != nil {
            lastErr = fmt.Errorf("select tool (attempt %d): %w", attempt, err)
            continue
        }

        result, execErr := a.toolRegistry.Execute(ctx, toolName, params)
        if execErr != nil {
            if isRetryableErr(execErr) {
                lastErr = execErr
                continue
            }
            return "", fmt.Errorf("non-retryable error after %d attempts: %w", attempt, execErr)
        }

        if !result.Success {
            lastErr = fmt.Errorf("tool %s returned failure (attempt %d): %s", toolName, attempt, result.Error)
            continue
        }
        return fmt.Sprintf("Step %d: recovered after %d retries", stepNum, attempt), nil
    }
    return "", fmt.Errorf("step %d failed after %d retries: %w", stepNum, maxRetries, lastErr)
}

// buildSummary creates a summary string from the plan and step results.
func (a *Agent) buildSummary(plan *pkg.Plan, results []string) string {
    if plan == nil {
        return "No plan was executed"
    }
    summary := fmt.Sprintf("Goal: %s\n", plan.Goal)
    summary += fmt.Sprintf("Steps completed: %d/%d\n", len(results), len(plan.Steps))
    for _, r := range results {
        summary += "- " + r + "\n"
    }
    return summary
}

// RunPlan implements pkg.Agent.RunPlan (stub for Phase 2).
func (a *Agent) RunPlan(ctx context.Context, sessionID string, input string) (*pkg.AgentResult, error) {
    return nil, fmt.Errorf("RunPlan not implemented in MVP")
}

// ApprovePlan implements pkg.Agent.ApprovePlan (stub for Phase 2).
func (a *Agent) ApprovePlan(ctx context.Context, sessionID string) (*pkg.AgentResult, error) {
    return nil, fmt.Errorf("ApprovePlan not implemented in MVP")
}

// GetSession implements pkg.Agent.GetSession.
func (a *Agent) GetSession(ctx context.Context, sessionID string) (*pkg.Session, error) {
    if a.sessionManager != nil {
        return a.sessionManager.Get(ctx, sessionID)
    }
    return nil, fmt.Errorf("session manager not configured")
}

// getState returns the current loop state.
func (a *Agent) getState() LoopState {
    a.mu.RLock()
    defer a.mu.RUnlock()
    return a.state
}

// setState updates the current loop state.
func (a *Agent) setState(state LoopState) {
    a.mu.Lock()
    defer a.mu.Unlock()
    a.state = state
}

// Planner returns the agent's planner (for testing).
func (a *Agent) Planner() pkg.Planner { return a.planner }

// ToolSelector returns the agent's tool selector (for testing).
func (a *Agent) ToolSelector() pkg.ToolSelector { return a.toolSelector }

// SelfCorrector returns the agent's self-corrector (for testing).
func (a *Agent) SelfCorrector() pkg.SelfCorrector { return a.selfCorrector }

// isRetryableErr checks if the error is retryable.
func isRetryableErr(err error) bool {
    if err == nil { return false }
    if _, ok := err.(*pkg.ErrRetryable); ok { return true }
    if _, ok := err.(*pkg.ErrLLMUnavailable); ok { return true }
    if _, ok := err.(*pkg.ErrToolNotFound); ok { return false }
    if _, ok := err.(*pkg.ErrPermissionDenied); ok { return false }
    if _, ok := err.(*pkg.ErrContextExceeded); ok { return false }
    errStr := err.Error()
    signals := []string{"timeout", "temporary", "try again", "rate limit",
        "too many requests", "server error", "internal error",
        "service unavailable", "connection refused", "connection reset",
        "deadline exceeded", "unexpected status 5"}
    for _, s := range signals {
        if strings.Contains(strings.ToLower(errStr), s) { return true }
    }
    return false
}

// retryBackoff waits with exponential backoff for the given attempt.
func retryBackoff(ctx context.Context, attempt int) error {
    d := 100 * time.Millisecond
    for i := 1; i < attempt; i++ { d *= 2 }
    select {
    case <-ctx.Done(): return ctx.Err()
    case <-time.After(d): return nil
    }
}

// Compile-time interface check
var _ pkg.Agent = (*Agent)(nil)
