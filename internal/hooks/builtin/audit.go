package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supcode/supcode/pkg"
)

// AuditHook is a post-tool-call hook that records audit logs.
type AuditHook struct {
	permEng pkg.PermissionEngine
}

// NewAuditHook creates an AuditHook using the given PermissionEngine.
func NewAuditHook(permEng pkg.PermissionEngine) *AuditHook {
	return &AuditHook{permEng: permEng}
}

func (h *AuditHook) Name() string { return "audit_log" }

// BeforeTool is a no-op for the audit hook.
func (h *AuditHook) BeforeTool(ctx context.Context, toolName string, params json.RawMessage) (json.RawMessage, error) {
	return params, nil
}

// AfterTool records the tool call asynchronously via PermissionEngine.LogAction.
func (h *AuditHook) AfterTool(ctx context.Context, toolName string, params json.RawMessage, result pkg.ToolResult) error {
	action := pkg.Action{
		Type:     "tool",
		ToolName: toolName,
		Params:   truncateString(string(params), 1024),
	}

	decision := pkg.DecisionAllow
	if !result.Success {
		decision = pkg.DecisionDeny
	}

	resultSummary := truncateString(result.Error, 500)
	if result.Success {
		resultSummary = "ok"
	}

	// Async: do not block tool return
	go func() {
		_ = h.permEng.LogAction(ctx, action, decision, resultSummary)
	}()

	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// Compile-time check
var _ pkg.ToolHook = (*AuditHook)(nil)

// Ensure fmt is used
var _ = fmt.Sprintf
