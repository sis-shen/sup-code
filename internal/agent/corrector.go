package agent

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    "time"

    "github.com/supcode/supcode/pkg"
)

// SelfCorrector implements pkg.SelfCorrector with exponential backoff retry.
type SelfCorrector struct {
    maxRetries  int
    backoffBase time.Duration
}

// NewSelfCorrector creates a new SelfCorrector with the given max retries and base backoff.
func NewSelfCorrector(maxRetries int, backoffBase time.Duration) *SelfCorrector {
    return &SelfCorrector{
        maxRetries:  maxRetries,
        backoffBase: backoffBase,
    }
}

// DefaultSelfCorrector creates a SelfCorrector with default settings (3 retries, 1s base backoff).
func DefaultSelfCorrector() *SelfCorrector {
    return &SelfCorrector{
        maxRetries:  3,
        backoffBase: time.Second,
    }
}

// Retry determines whether to retry based on the error type.
// Returns modified parameters for retry, or an error if the error is non-retryable.
func (c *SelfCorrector) Retry(ctx context.Context, toolName string, params json.RawMessage, prevErr error) (json.RawMessage, error) {
    if !c.IsRetryable(prevErr) {
        return nil, fmt.Errorf("non-retryable error: %w", prevErr)
    }

    // Check context
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }

    // Return modified params (same params, the correction is in the retry)
    return params, nil
}

// MaxRetries returns the maximum number of retry attempts.
func (c *SelfCorrector) MaxRetries() int {
    return c.maxRetries
}

// BackoffDuration returns the backoff duration for a given attempt (1-indexed).
func (c *SelfCorrector) BackoffDuration(attempt int) time.Duration {
    if attempt < 1 {
        attempt = 1
    }
    // Exponential: base * 2^(attempt-1)
    backoff := c.backoffBase
    for i := 1; i < attempt; i++ {
        backoff *= 2
    }
    return backoff
}

// WaitAndRetry blocks until it's time to retry, respecting context cancellation.
func (c *SelfCorrector) WaitAndRetry(ctx context.Context, attempt int) error {
    duration := c.BackoffDuration(attempt)
    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-time.After(duration):
        return nil
    }
}

// isRetryable checks if the error can be recovered by retrying.
func (c *SelfCorrector) IsRetryable(err error) bool {
    if err == nil {
        return false
    }

    // Structured retryable errors
    if _, ok := err.(*pkg.ErrRetryable); ok {
        return true
    }

    // Unavailable LLM can be retried
    if _, ok := err.(*pkg.ErrLLMUnavailable); ok {
        return true
    }

    // Tool not found is not retryable (tool doesn't exist)
    if _, ok := err.(*pkg.ErrToolNotFound); ok {
        return false
    }

    // Permission denied is not retryable
    if _, ok := err.(*pkg.ErrPermissionDenied); ok {
        return false
    }

    // Context exceeded is not retryable
    if _, ok := err.(*pkg.ErrContextExceeded); ok {
        return false
    }

    // String-based heuristics for transient errors
    errStr := err.Error()
    transientSignals := []string{
        "timeout",
        "temporary",
        "try again",
        "rate limit",
        "too many requests",
        "server error",
        "internal error",
        "service unavailable",
        "connection refused",
        "connection reset",
        "deadline exceeded",
        "unexpected status 5",
    }

    for _, signal := range transientSignals {
        if strings.Contains(strings.ToLower(errStr), signal) {
            return true
        }
    }

    return false
}

// Compile-time interface check
var _ pkg.SelfCorrector = (*SelfCorrector)(nil)
