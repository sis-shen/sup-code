package tui

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supcode/supcode/pkg"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test_sessions.json")
	sm, err := NewSessionManager(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sm.CloseAll(); os.Remove(dbPath) })
	return NewService(sm)
}

func TestServiceNewAndSessionManager(t *testing.T) {
	svc := newTestService(t)
	require.NotNil(t, svc)
	require.NotNil(t, svc.SessionManager())
}

func TestServiceHandleCommandNew(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	result, err := svc.HandleCommand(ctx, "session-1", "/new")
	require.NoError(t, err)
	assert.True(t, result.Handled)
	assert.True(t, result.NewSession)
}

func TestServiceHandleCommandExit(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	result, err := svc.HandleCommand(ctx, "session-1", "/exit")
	require.NoError(t, err)
	assert.True(t, result.Handled)
	assert.False(t, result.NewSession)
}

func TestServiceHandleCommandHelp(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	result, err := svc.HandleCommand(ctx, "session-1", "/help")
	require.NoError(t, err)
	assert.True(t, result.Handled)
	assert.Contains(t, result.Message, "/new")
	assert.Contains(t, result.Message, "/exit")
}

func TestServiceHandleCommandUnknown(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	result, err := svc.HandleCommand(ctx, "session-1", "/unknown")
	require.NoError(t, err)
	assert.False(t, result.Handled)
}

func TestServiceHandleCommandNonSlash(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	result, err := svc.HandleCommand(ctx, "session-1", "just a normal message")
	require.NoError(t, err)
	assert.False(t, result.Handled)
}

func TestServiceHandleCommandEmpty(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	result, err := svc.HandleCommand(ctx, "session-1", "   ")
	require.NoError(t, err)
	assert.False(t, result.Handled)
}

func TestServiceNotify(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	notified := false
	svc.SetNotifyHandler(func(sessionID string, level pkg.NotifyLevel, message string) {
		notified = true
		assert.Equal(t, "test-session", sessionID)
		assert.Equal(t, pkg.NotifyInfo, level)
		assert.Equal(t, "test message", message)
	})

	svc.Notify(ctx, "test-session", pkg.NotifyInfo, "test message")
	assert.True(t, notified)
}

func TestServiceSubmitInputAndConfirmation(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// Test confirmation submission
	confirmCh := make(chan bool, 1)
	svc.mu.Lock()
	svc.confirmChannels["test-session"] = confirmCh
	svc.mu.Unlock()

	svc.SubmitConfirmation("test-session", true)
	result := <-confirmCh
	assert.True(t, result)

	// Test input submission
	inputCh := make(chan string, 1)
	svc.mu.Lock()
	svc.inputChannels["test-session"] = inputCh
	svc.mu.Unlock()

	svc.SubmitInput("test-session", "hello")
	input := <-inputCh
	assert.Equal(t, "hello", input)

	_ = ctx
}

func TestServiceSetCommandHandler(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	called := false
	svc.SetCommandHandler(func(ctx context.Context, sessionID, input string) (pkg.CommandResult, error) {
		called = true
		return pkg.CommandResult{Handled: true, Message: "custom handled"}, nil
	})

	result, err := svc.HandleCommand(ctx, "session-1", "/custom")
	require.NoError(t, err)
	assert.True(t, result.Handled)
	assert.True(t, called)
}

func TestServiceSessionManager(t *testing.T) {
	svc := newTestService(t)
	sm := svc.SessionManager()
	require.NotNil(t, sm)
	require.NotEmpty(t, sm.DBPath())
}

func TestServiceHandleCommandNoneInput(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	result, err := svc.HandleCommand(ctx, "s1", "/")
	require.NoError(t, err)
	assert.False(t, result.Handled)
}