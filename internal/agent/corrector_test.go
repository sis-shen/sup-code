package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/supcode/supcode/pkg"
)

func TestCorrector_RetryWithRetryable(t *testing.T) {
	c := NewSelfCorrector(3, time.Second)
	params, err := c.Retry(context.Background(), "tool", json.RawMessage(`{"key":"val"}`), &pkg.ErrRetryable{
		Cause: fmt.Errorf("rate limited"),
	})
	assert.NoError(t, err)
	assert.NotNil(t, params)
	assert.Contains(t, string(params), "key")
}

func TestCorrector_RetryWithPermissionDenied(t *testing.T) {
	c := NewSelfCorrector(3, time.Second)
	params, err := c.Retry(context.Background(), "tool", json.RawMessage(`{}`), &pkg.ErrPermissionDenied{
		Action: pkg.Action{Type: "file_write", Target: "/tmp/test"},
		Rule:   "deny-all",
	})
	assert.Error(t, err)
	assert.Nil(t, params)
	assert.Contains(t, err.Error(), "non-retryable")
}

func TestCorrector_MaxRetries(t *testing.T) {
	c := NewSelfCorrector(5, time.Second)
	assert.Equal(t, 5, c.MaxRetries())

	def := DefaultSelfCorrector()
	assert.Equal(t, 3, def.MaxRetries())
}

func TestCorrector_BackoffDuration(t *testing.T) {
	c := NewSelfCorrector(3, 500*time.Millisecond)
	assert.Equal(t, 500*time.Millisecond, c.BackoffDuration(1))
	assert.Equal(t, time.Second, c.BackoffDuration(2))
	assert.Equal(t, 2*time.Second, c.BackoffDuration(3))
}
