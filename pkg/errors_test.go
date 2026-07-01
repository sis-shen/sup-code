package pkg

import (
	"errors"
	"testing"
	"time"
)

// TestErrRetryable 测试 ErrRetryable
func TestErrRetryable(t *testing.T) {
	cause := errors.New("rate limited")
	err := &ErrRetryable{Cause: cause, RetryAfter: 5 * time.Second}

	msg := err.Error()
	if msg == "" {
		t.Fatal("Error() returned empty string")
	}

	if !contains(msg, "retryable") {
		t.Errorf("Error() = %q, should contain 'retryable'", msg)
	}

	if !errors.Is(err, cause) {
		t.Error("errors.Is should unwrap to cause")
	}

	var target *ErrRetryable
	if !errors.As(err, &target) {
		t.Error("errors.As should match *ErrRetryable")
	}
}

func TestErrPermissionDenied(t *testing.T) {
	err := &ErrPermissionDenied{
		Action: Action{Type: "command", Target: "rm -rf /"},
		Rule:   "deny-rm",
	}

	msg := err.Error()
	if !contains(msg, "permission denied") || !contains(msg, "command") || !contains(msg, "deny-rm") {
		t.Errorf("Error() = %q, should contain relevant fields", msg)
	}
}

func TestErrToolNotFound(t *testing.T) {
	err := &ErrToolNotFound{ToolName: "nonexistent_tool"}

	msg := err.Error()
	if !contains(msg, "tool not found") || !contains(msg, "nonexistent_tool") {
		t.Errorf("Error() = %q, should contain tool name", msg)
	}
}

func TestErrContextExceeded(t *testing.T) {
	err := &ErrContextExceeded{Current: 10000, Limit: 8000}

	msg := err.Error()
	if !contains(msg, "context exceeded") || !contains(msg, "10000") || !contains(msg, "8000") {
		t.Errorf("Error() = %q, should contain token counts", msg)
	}
}

func TestErrLLMUnavailable(t *testing.T) {
	cause := errors.New("connection timeout")
	err := &ErrLLMUnavailable{Provider: "openai", Cause: cause}

	msg := err.Error()
	if !contains(msg, "llm unavailable") || !contains(msg, "openai") || !contains(msg, "connection timeout") {
		t.Errorf("Error() = %q, should contain provider and cause", msg)
	}

	if !errors.Is(err, cause) {
		t.Error("errors.Is should unwrap to cause")
	}

	var target *ErrLLMUnavailable
	if !errors.As(err, &target) {
		t.Error("errors.As should match *ErrLLMUnavailable")
	}
}

func TestErrWrapping(t *testing.T) {
	base := &ErrToolNotFound{ToolName: "missing"}

	withW := &ErrRetryable{Cause: base, RetryAfter: 1 * time.Second}
	if !errors.Is(withW, base) {
		t.Error("errors.Is should follow %w chain through ErrRetryable.Unwrap")
	}

	var target *ErrToolNotFound
	if !errors.As(withW, &target) {
		t.Error("errors.As should follow %w chain through ErrRetryable.Unwrap")
	}
	if target.ToolName != "missing" {
		t.Errorf("unwrapped ToolName = %q, want %q", target.ToolName, "missing")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
