package pkg

import (
	"fmt"
	"time"
)

// ─── 11.1 错误类型 ──────────────────────────────────────────────

// ErrRetryable 可重试错误
type ErrRetryable struct {
	Cause      error
	RetryAfter time.Duration
}

func (e *ErrRetryable) Error() string {
	return fmt.Sprintf("retryable: %v (after %v)", e.Cause, e.RetryAfter)
}

func (e *ErrRetryable) Unwrap() error {
	return e.Cause
}

// ErrPermissionDenied 权限拒绝
type ErrPermissionDenied struct {
	Action Action
	Rule   string
}

func (e *ErrPermissionDenied) Error() string {
	return fmt.Sprintf("permission denied: %s by rule %s", e.Action.Type, e.Rule)
}

// ErrToolNotFound 工具不存在
type ErrToolNotFound struct {
	ToolName string
}

func (e *ErrToolNotFound) Error() string {
	return fmt.Sprintf("tool not found: %s", e.ToolName)
}

// ErrContextExceeded Token 超限
type ErrContextExceeded struct {
	Current int
	Limit   int
}

func (e *ErrContextExceeded) Error() string {
	return fmt.Sprintf("context exceeded: %d/%d", e.Current, e.Limit)
}

// ErrLLMUnavailable LLM 不可用
type ErrLLMUnavailable struct {
	Provider string
	Cause    error
}

func (e *ErrLLMUnavailable) Error() string {
	return fmt.Sprintf("llm unavailable [%s]: %v", e.Provider, e.Cause)
}

func (e *ErrLLMUnavailable) Unwrap() error {
	return e.Cause
}
