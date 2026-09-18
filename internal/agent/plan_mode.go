package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/supcode/supcode/pkg"
)

// RunPlan generates a plan using the LLM and returns it without executing.
// The plan is persisted to the session and the state transitions to WaitingApproval.
func (a *Agent) RunPlan(ctx context.Context, sessionID string, input string) (*pkg.AgentResult, error) {
	defer a.setState(StateIdle)

	// Build context
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

	// Get available tools
	toolSchemas := a.toolRegistry.ListSchemas()

	// Generate plan
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

	// Persist plan to agent and session
	a.mu.Lock()
	a.plan = plan
	a.mu.Unlock()
	if a.sessionManager != nil {
		if err := a.sessionManager.SetPlan(ctx, sessionID, *plan); err != nil {
			log.Printf("set plan: %v", err)
		}
		if err := a.sessionManager.SetState(ctx, sessionID, pkg.StateWaitingApproval); err != nil {
			log.Printf("set state waiting approval: %v", err)
		}
	}

	a.setState(StateWaitingApproval)
	return &pkg.AgentResult{
		Plan:    plan,
		Summary: fmt.Sprintf("Plan generated with %d steps, waiting for approval", len(plan.Steps)),
	}, nil
}

// ApprovePlan executes a previously generated plan.
// It loads the plan from the session and runs each step sequentially.
func (a *Agent) ApprovePlan(ctx context.Context, sessionID string) (*pkg.AgentResult, error) {
	defer a.setState(StateIdle)

	// Load plan from session
	var plan *pkg.Plan
	if a.sessionManager != nil {
		session, err := a.sessionManager.Get(ctx, sessionID)
		if err != nil {
			a.setState(StateError)
			return &pkg.AgentResult{Error: fmt.Sprintf("get session: %v", err)}, fmt.Errorf("get session: %w", err)
		}
		plan = session.Plan
	} else {
		// Fallback: use in-memory plan
		a.mu.RLock()
		if a.plan != nil {
			p := *a.plan
			plan = &p
		}
		a.mu.RUnlock()
	}
	if plan == nil {
		a.setState(StateError)
		return &pkg.AgentResult{Error: "no plan to execute"}, fmt.Errorf("no plan to execute")
	}
	if len(plan.Steps) == 0 {
		a.setState(StateCompleted)
		return &pkg.AgentResult{Summary: "Plan has no steps, nothing to execute"}, nil
	}

	// Build context for execution
	_, _, err := a.contextManager.BuildContext(ctx, sessionID)
	if err != nil {
		a.setState(StateError)
		return &pkg.AgentResult{Error: fmt.Sprintf("build context: %v", err)}, fmt.Errorf("build context: %w", err)
	}

	toolSchemas := a.toolRegistry.ListSchemas()
	a.setState(StateActing)
	if a.sessionManager != nil {
		if err := a.sessionManager.SetState(ctx, sessionID, pkg.StateActing); err != nil {
			log.Printf("set state acting: %v", err)
		}
	}

	// Execute each step
	var stepResults []string
	for stepIndex, step := range plan.Steps {
		select {
		case <-ctx.Done():
			a.setState(StateError)
			return &pkg.AgentResult{
				Plan:    plan,
				Summary: fmt.Sprintf("execution canceled after step %d/%d", stepIndex, len(plan.Steps)),
				Error:   ctx.Err().Error(),
			}, ctx.Err()
		default:
		}

		if stepIndex >= a.maxIterations {
			a.setState(StateCompleted)
			return &pkg.AgentResult{
				Plan:    plan,
				Summary: fmt.Sprintf("completed %d/%d steps (max iterations)", stepIndex, len(plan.Steps)),
			}, nil
		}

		result, err := a.executeStep(ctx, sessionID, step, toolSchemas, stepIndex+1)
		if err != nil {
			if isRetryableErr(err) {
				result, retryErr := a.retryStep(ctx, step, toolSchemas, stepIndex+1, err)
				if retryErr != nil {
					a.setState(StateError)
					return &pkg.AgentResult{
						Plan:    plan,
						Summary: fmt.Sprintf("step %d failed after retries: %v", stepIndex+1, retryErr),
						Error:   retryErr.Error(),
					}, retryErr
				}
				stepResults = append(stepResults, result)
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

	// Complete
	summary := a.buildSummary(plan, stepResults)
	a.setState(StateCompleted)
	if a.sessionManager != nil {
		if err := a.sessionManager.SetState(ctx, sessionID, pkg.StateCompleted); err != nil {
			log.Printf("set state completed: %v", err)
		}
	}
	return &pkg.AgentResult{
		Plan:    plan,
		Summary: summary,
	}, nil
}
