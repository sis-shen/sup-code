package tui

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supcode/supcode/pkg"
)

func setupTestService(t *testing.T) *Service {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test_svc_s.json")
	sm, err := NewSessionManager(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sm.CloseAll(); os.Remove(dbPath) })
	return NewService(sm)
}

func TestService_StreamResponse_TextDelta(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()
	stream := make(chan pkg.StreamEvent, 2)

	// Start StreamResponse in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- svc.StreamResponse(ctx, "session-1", stream)
	}()

	// Send events
	stream <- pkg.StreamEvent{Type: "text_delta", Delta: "Hello"}
	stream <- pkg.StreamEvent{Type: "done"}
	close(stream)

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for StreamResponse")
	}
}

func TestService_StreamResponse_ToolCall(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()
	stream := make(chan pkg.StreamEvent, 2)

	errCh := make(chan error, 1)
	go func() {
		errCh <- svc.StreamResponse(ctx, "session-2", stream)
	}()

	stream <- pkg.StreamEvent{
		Type: "tool_call",
		ToolCall: &pkg.ToolCall{
			ID:   "call-1",
			Name: "ReadFile",
		},
	}
	close(stream)

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestService_StreamResponse_Error(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()
	stream := make(chan pkg.StreamEvent, 2)

	errCh := make(chan error, 1)
	go func() {
		errCh <- svc.StreamResponse(ctx, "session-3", stream)
	}()

	stream <- pkg.StreamEvent{Type: "error", Error: "test error"}
	stream <- pkg.StreamEvent{Type: "done"}
	close(stream)

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestService_StreamResponse_ClosedChannel(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()
	stream := make(chan pkg.StreamEvent)

	errCh := make(chan error, 1)
	go func() {
		errCh <- svc.StreamResponse(ctx, "session-4", stream)
	}()

	close(stream)

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestService_StreamResponse_Cancel(t *testing.T) {
	svc := setupTestService(t)
	ctx, cancel := context.WithCancel(context.Background())
	stream := make(chan pkg.StreamEvent)

	errCh := make(chan error, 1)
	go func() {
		errCh <- svc.StreamResponse(ctx, "session-cancel", stream)
	}()

	cancel()

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestService_RequestConfirmation_Accepted(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	resultCh := make(chan bool, 1)
	errCh := make(chan error, 1)
	go func() {
		ok, err := svc.RequestConfirmation(ctx, "session-conf-1", pkg.ConfirmPrompt{
			Title:   "Test",
			Message: "Allow?",
		})
		errCh <- err
		resultCh <- ok
	}()

	// Give goroutine time to register its channel
	time.Sleep(50 * time.Millisecond)

	svc.SubmitConfirmation("session-conf-1", true)

	select {
	case ok := <-resultCh:
		assert.True(t, ok)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
	require.NoError(t, <-errCh)
}

func TestService_RequestConfirmation_Rejected(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	resultCh := make(chan bool, 1)
	errCh := make(chan error, 1)
	go func() {
		ok, err := svc.RequestConfirmation(ctx, "session-conf-2", pkg.ConfirmPrompt{
			Title:   "Test",
			Message: "Deny?",
		})
		errCh <- err
		resultCh <- ok
	}()

	time.Sleep(50 * time.Millisecond)
	svc.SubmitConfirmation("session-conf-2", false)

	select {
	case ok := <-resultCh:
		assert.False(t, ok)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
	require.NoError(t, <-errCh)
}

func TestService_ReadInput_Basic(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	inputCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		input, err := svc.ReadInput(ctx, "session-in-1")
		errCh <- err
		inputCh <- input
	}()

	time.Sleep(50 * time.Millisecond)
	svc.SubmitInput("session-in-1", "hello world")

	select {
	case input := <-inputCh:
		assert.Equal(t, "hello world", input)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
	require.NoError(t, <-errCh)
}

func TestService_ReadInput_Cancel(t *testing.T) {
	svc := setupTestService(t)
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		_, err := svc.ReadInput(ctx, "session-in-cancel")
		errCh <- err
	}()

	cancel()

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}