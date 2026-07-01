package agent

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"

    "github.com/supcode/supcode/pkg"
)

// ToolSelector implements pkg.ToolSelector by calling an LLM to select tools.
type ToolSelector struct {
    llmClient pkg.LLMClient
}

// NewToolSelector creates a new ToolSelector with the given LLM client.
func NewToolSelector(llmClient pkg.LLMClient) *ToolSelector {
    return &ToolSelector{llmClient: llmClient}
}

// Select uses the LLM to select the most appropriate tool for a given step.
// It returns the tool name and parameters as a JSON raw message.
func (s *ToolSelector) Select(ctx context.Context, step pkg.PlanItem, availableTools []pkg.ToolSchema) (string, json.RawMessage, error) {
    if len(availableTools) == 0 {
        return "", nil, fmt.Errorf("no tools available for step: %s", step.Description)
    }

    selectionPrompt := fmt.Sprintf(
        `Select the most appropriate tool for this task step.

Step description: %s
Step ID: %s
Tool hint: %s

Available tools:
%s

Respond with a JSON object selecting the tool and parameters:
{
    "tool": "tool_name",
    "parameters": { ... tool-specific parameters ... }
}

Choose only from the available tools listed above.`,
        step.Description,
        step.ID,
        step.ToolHint,
        formatToolList(availableTools),
    )

    // Use function calling by passing tool schemas
    eventCh, err := s.llmClient.Chat(ctx, selectionPrompt, nil, availableTools)
    if err != nil {
        return "", nil, fmt.Errorf("llm chat: %w", err)
    }

    var textParts []string
    var selectedTool string
    var selectedParams json.RawMessage

    for event := range eventCh {
        switch event.Type {
        case "tool_call":
            if event.ToolCall != nil && selectedTool == "" {
                selectedTool = event.ToolCall.Name
                selectedParams = event.ToolCall.Params
            }
        case "text_delta":
            textParts = append(textParts, event.Delta)
        case "done":
            // OK
        case "error":
            return "", nil, fmt.Errorf("llm error: %s", event.Error)
        }
    }

    if selectedTool != "" {
        return selectedTool, selectedParams, nil
    }

    // Fallback: try to parse JSON from text response
    text := strings.Join(textParts, "")
    toolName, params, err := parseSelectionFromText(text)
    if err != nil {
        return "", nil, fmt.Errorf("parse selection: %w", err)
    }

    return toolName, params, nil
}

// formatToolList formats tool schemas into a readable string for the prompt.
func formatToolList(tools []pkg.ToolSchema) string {
    var b strings.Builder
    for i, t := range tools {
        b.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, t.Name, t.Description))
    }
    return b.String()
}

// parseSelectionFromText tries to extract tool selection JSON from text.
func parseSelectionFromText(text string) (string, json.RawMessage, error) {
    // Try to find JSON in markdown
    if idx := strings.Index(text, "```json"); idx >= 0 {
        end := strings.Index(text[idx+7:], "```")
        if end >= 0 {
            text = text[idx+7 : idx+7+end]
        }
    }

    start := strings.Index(text, "{")
    end := strings.LastIndex(text, "}")
    if start >= 0 && end > start {
        text = text[start : end+1]
    }

    var result struct {
		Tool       string          `json:"tool"`
		Parameters json.RawMessage `json:"parameters"`
    }
    if err := json.Unmarshal([]byte(text), &result); err != nil {
        return "", nil, err
    }

    if result.Tool == "" {
        return "", nil, fmt.Errorf("no tool selected")
    }

    return result.Tool, result.Parameters, nil
}

// Compile-time interface check
var _ pkg.ToolSelector = (*ToolSelector)(nil)
