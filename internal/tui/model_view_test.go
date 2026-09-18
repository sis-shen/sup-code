package tui

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/pkg"
)

func setupTestModelReady(t *testing.T) *Model {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test_mv.json")
	sm, err := NewSessionManager(dbPath)
	require.NoError(t, err)
	svc := NewService(sm)
	r, err := NewRenderer()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sm.CloseAll(); _ = r.Close(); _ = os.Remove(dbPath) })
	session, err := sm.Create(context.Background(), "Test")
	require.NoError(t, err)
	m := NewModel(svc, r, session.ID)
	// Simulate window resize to set ready state
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	return m
}

func TestModel_View_Initializing(t *testing.T) {
	// Model without ready state returns "Initializing..."
	dbPath := filepath.Join(t.TempDir(), "test_vi.json")
	sm, err := NewSessionManager(dbPath)
	require.NoError(t, err)
	svc := NewService(sm)
	r, err := NewRenderer()
	require.NoError(t, err)
	defer func() { _ = r.Close() }()
	defer func() { _ = sm.CloseAll() }()
	session, err := sm.Create(context.Background(), "Test")
	require.NoError(t, err)
	m := NewModel(svc, r, session.ID)
	view := m.View()
	assert.Contains(t, StripANSI(view), "Initializing")
}

func TestModel_View_Ready(t *testing.T) {
	m := setupTestModelReady(t)
	view := m.View()
	require.NotEmpty(t, view, "View should return non-empty string when ready")
	assert.NotContains(t, StripANSI(view), "Initializing")
}

func TestModel_View_WithMessages(t *testing.T) {
	m := setupTestModelReady(t)
	m.addMessage(pkg.RoleUser, "hello from test")
	view := m.View()
	require.NotEmpty(t, view)
	clean := StripANSI(view)
	assert.Contains(t, clean, "hello from test")
}

func TestModel_View_ConfirmMode(t *testing.T) {
	m := setupTestModelReady(t)
	m.viewMode = modeConfirm
	m.input.SetValue("y")
	view := m.View()
	require.NotEmpty(t, view)
	clean := StripANSI(view)
	assert.Contains(t, clean, "Confirm")
}

func TestModel_View_Error(t *testing.T) {
	m := setupTestModelReady(t)
	m.err = context.DeadlineExceeded
	view := m.View()
	require.NotEmpty(t, view)
	clean := StripANSI(view)
	assert.Contains(t, clean, "Error")
}

func TestModel_SessionID(t *testing.T) {
	m := setupTestModelReady(t)
	id := m.SessionID()
	require.NotEmpty(t, id)
}

func TestModel_Messages(t *testing.T) {
	m := setupTestModelReady(t)
	msgs := m.Messages()
	require.NotNil(t, msgs)
	assert.Empty(t, msgs)

	m.addMessage(pkg.RoleUser, "test")
	msgs = m.Messages()
	assert.Len(t, msgs, 1)
}

func TestModel_Update_WindowSize(t *testing.T) {
	m := setupTestModelReady(t)
	result, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
	require.NotNil(t, result)
	_ = cmd // cmd may be nil for window size messages
}

func TestModel_Update_StreamEventTextDelta(t *testing.T) {
	m := setupTestModelReady(t)
	msg := StreamEventMsg{
		SessionID: m.sessionID,
		Event:     pkg.StreamEvent{Type: "text_delta", Delta: "Hello streaming world"},
	}
	result, cmd := m.Update(msg)
	require.NotNil(t, result)
	model := result.(*Model)
	assert.Greater(t, model.currentMsg.Len(), 0)
	_ = cmd
}

func TestModel_Update_StreamEventToolCall(t *testing.T) {
	m := setupTestModelReady(t)
	msg := StreamEventMsg{
		SessionID: m.sessionID,
		Event: pkg.StreamEvent{
			Type: "tool_call",
			ToolCall: &pkg.ToolCall{
				Name:   "Bash",
				Params: json.RawMessage(`{"command":"ls -la"}`),
			},
		},
	}
	result, cmd := m.Update(msg)
	require.NotNil(t, result)
	model := result.(*Model)
	assert.True(t, model.isThinking)
	assert.Equal(t, "Bash", model.thinkingTool)
	_ = cmd
}

func TestModel_Update_StreamEventDone(t *testing.T) {
	m := setupTestModelReady(t)
	// Send some text first
	m.Update(StreamEventMsg{
		SessionID: m.sessionID,
		Event:     pkg.StreamEvent{Type: "text_delta", Delta: "Final message"},
	})
	// Mark as done
	result, cmd := m.Update(StreamEventMsg{
		SessionID: m.sessionID,
		Event:     pkg.StreamEvent{Type: "done"},
	})
	require.NotNil(t, result)
	model := result.(*Model)
	assert.False(t, model.isThinking)
	assert.Equal(t, "Ready", model.status)
	_ = cmd
}

func TestModel_Update_StreamEventError(t *testing.T) {
	m := setupTestModelReady(t)
	msg := StreamEventMsg{
		SessionID: m.sessionID,
		Event:     pkg.StreamEvent{Type: "error", Error: "API error"},
	}
	result, cmd := m.Update(msg)
	require.NotNil(t, result)
	model := result.(*Model)
	assert.False(t, model.isThinking)
	assert.Contains(t, model.status, "Error")
	_ = cmd
}

func TestModel_Update_Notification(t *testing.T) {
	m := setupTestModelReady(t)
	msg := NotificationMsg{
		SessionID: m.sessionID,
		Level:     pkg.NotifyInfo,
		Message:   "Test notification",
	}
	result, cmd := m.Update(msg)
	require.NotNil(t, result)
	model := result.(*Model)
	assert.Contains(t, model.status, "Test notification")
	_ = cmd
}

func TestModel_Update_ConfirmResult(t *testing.T) {
	m := setupTestModelReady(t)

	// Set up a confirm channel
	confirmDone := make(chan bool, 1)
	m.confirmDone = confirmDone

	result, cmd := m.Update(ConfirmResultMsg{
		SessionID: m.sessionID,
		Approved:  true,
	})
	require.NotNil(t, result)
	assert.Equal(t, modeNormal, m.viewMode)
	_ = cmd
}

func TestModel_Update_ConfirmResultRejected(t *testing.T) {
	m := setupTestModelReady(t)

	confirmDone := make(chan bool, 1)
	m.confirmDone = confirmDone

	result, cmd := m.Update(ConfirmResultMsg{
		SessionID: m.sessionID,
		Approved:  false,
	})
	require.NotNil(t, result)
	assert.Equal(t, modeNormal, m.viewMode)
	_ = cmd
}
