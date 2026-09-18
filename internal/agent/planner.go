package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/supcode/supcode/pkg"
)

// Planner implements pkg.Planner by calling an LLM to generate a structured plan.
type Planner struct {
	llmClient pkg.LLMClient
}

// NewPlanner creates a new Planner with the given LLM client.
func NewPlanner(llmClient pkg.LLMClient) *Planner {
	return &Planner{llmClient: llmClient}
}

// Plan generates a structured plan by calling the LLM.
// It sends the system prompt, message history, and tool schemas to the LLM,
// and expects a JSON response with a plan structure.
func (p *Planner) Plan(ctx context.Context, systemPrompt string, messages []pkg.Message, tools []pkg.ToolSchema) (*pkg.Plan, error) {
	planPrompt := `You are a task planner. Your job is to break down the user's request into sequential steps.

Given the user's request, create a plan with:
1. A clear goal statement
2. A series of steps, each with a unique ID and description

Respond with a JSON object in this exact format:
{
    "goal": "Brief goal description",
    "steps": [
        {"id": "step-1", "description": "Description of step 1", "tool_hint": "optional_tool_name"},
        {"id": "step-2", "description": "Description of step 2", "tool_hint": "optional_tool_name"}
    ]
}

If the request doesn't need tools, create a single step with description "Process the request".
If there are available tools, use tool_hint to suggest which tool might be appropriate.`

	fullPrompt := systemPrompt
	if fullPrompt != "" {
		fullPrompt = systemPrompt + "\n\n---\n\n" + planPrompt
	} else {
		fullPrompt = planPrompt
	}

	eventCh, err := p.llmClient.Chat(ctx, fullPrompt, messages, tools)
	if err != nil {
		return nil, fmt.Errorf("llm chat: %w", err)
	}

	var textParts []string
	var toolEvents []pkg.StreamEvent
	for event := range eventCh {
		switch event.Type {
		case "text_delta":
			textParts = append(textParts, event.Delta)
		case "tool_call":
			toolEvents = append(toolEvents, event)
		case "done":
			// OK
		case "error":
			return nil, fmt.Errorf("llm error: %s", event.Error)
		}
	}

	// Try to extract JSON from the text response
	text := strings.Join(textParts, "")

	// If there's a tool call with a structured output, use that
	if len(toolEvents) > 0 {
		// The tool call might contain plan data in its arguments
		for _, te := range toolEvents {
			if te.ToolCall != nil {
				plan, err := parsePlanFromJSON(te.ToolCall.Params)
				if err == nil {
					return plan, nil
				}
			}
		}
	}

	// Try to find and parse JSON in the text response
	plan, err := parsePlanFromText(text)
	if err != nil {
		return nil, fmt.Errorf("parse plan: %w", err)
	}

	return plan, nil
}

// parsePlanFromJSON parses a Plan from a JSON message.
func parsePlanFromJSON(data []byte) (*pkg.Plan, error) {
	var plan pkg.Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, err
	}

	// Validate basic structure
	if len(plan.Steps) == 0 {
		return nil, fmt.Errorf("plan has no steps")
	}

	// Set default statuses
	for i := range plan.Steps {
		if plan.Steps[i].ID == "" {
			plan.Steps[i].ID = fmt.Sprintf("step-%d", i+1)
		}
		if plan.Steps[i].Status == "" {
			plan.Steps[i].Status = "pending"
		}
	}

	return &plan, nil
}

// parsePlanFromText extracts a JSON plan from text that may contain markdown.
func parsePlanFromText(text string) (*pkg.Plan, error) {
	// Try to find JSON block in markdown
	if idx := strings.Index(text, "```json"); idx >= 0 {
		end := strings.Index(text[idx+7:], "```")
		if end >= 0 {
			text = text[idx+7 : idx+7+end]
		}
	} else if idx := strings.Index(text, "```"); idx >= 0 {
		end := strings.Index(text[idx+3:], "```")
		if end >= 0 {
			text = text[idx+3 : idx+3+end]
		}
	}

	// Try to find a JSON object
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		text = text[start : end+1]
	}

	return parsePlanFromJSON([]byte(text))
}

// Compile-time interface check
var _ pkg.Planner = (*Planner)(nil)
