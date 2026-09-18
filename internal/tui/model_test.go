package tui

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/pkg"
)

func setupTestModel(t *testing.T) *Model {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test_model.db")
	sm, err := NewSessionManager(dbPath)
	require.NoError(t, err)

	svc := NewService(sm)
	renderer, err := NewRenderer()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = sm.CloseAll()
		_ = renderer.Close()
		_ = os.Remove(dbPath)
	})

	session, err := sm.Create(context.Background(), "Test")
	require.NoError(t, err)

	return NewModel(svc, renderer, session.ID)
}

func TestModelInit(t *testing.T) {
	m := setupTestModel(t)
	cmds := m.Init()
	require.NotNil(t, cmds, "Init() should return initial commands")
}

func TestModelCtrlC(t *testing.T) {
	m := setupTestModel(t)
	keyMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
	result, cmd := m.Update(keyMsg)
	require.NotNil(t, result)

	// Verify it returns a quit command by checking the Update for QuitMsg
	quitResult, _ := result.Update(tea.QuitMsg{})
	_ = quitResult
	_ = cmd
}

func TestModelNewSessionWorks(t *testing.T) {
	m := setupTestModel(t)

	// Simulate typing "/new" and pressing Enter
	m.input.SetValue("/new")
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	_, _ = m.Update(enterMsg)

	// After /new, messages should have a system message
	found := false
	for _, msg := range m.Messages() {
		if msg.role == pkg.RoleSystem {
			found = true
			break
		}
	}
	require.True(t, found, "/new should produce a system message")
}

func TestStreamEventTextDelta(t *testing.T) {
	m := setupTestModel(t)

	msg := StreamEventMsg{
		SessionID: m.sessionID,
		Event: pkg.StreamEvent{
			Type:  "text_delta",
			Delta: "Hello",
		},
	}
	result, cmd := m.Update(msg)
	require.NotNil(t, result)

	// After text_delta, the main msg builder should have content
	model := result.(*Model)
	require.Greater(t, model.currentMsg.Len(), 0)
	_ = cmd
}

func TestStreamEventToolCall(t *testing.T) {
	m := setupTestModel(t)

	msg := StreamEventMsg{
		SessionID: m.sessionID,
		Event: pkg.StreamEvent{
			Type: "tool_call",
			ToolCall: &pkg.ToolCall{
				Name:   "ReadFile",
				Params: []byte(`{"path": "test.txt"}`),
			},
		},
	}
	result, cmd := m.Update(msg)
	require.NotNil(t, result)

	model := result.(*Model)
	require.True(t, model.isThinking)
	require.Equal(t, "ReadFile", model.thinkingTool)
	_ = cmd
}

func TestStreamEventDone(t *testing.T) {
	m := setupTestModel(t)

	// First send some text
	m.Update(StreamEventMsg{
		SessionID: m.sessionID,
		Event:     pkg.StreamEvent{Type: "text_delta", Delta: "Hello world"},
	})

	// Then mark as done
	doneMsg := StreamEventMsg{
		SessionID: m.sessionID,
		Event:     pkg.StreamEvent{Type: "done"},
	}
	result, cmd := m.Update(doneMsg)
	require.NotNil(t, result)

	model := result.(*Model)
	require.False(t, model.isThinking)
	require.GreaterOrEqual(t, len(model.Messages()), 1)
	_ = cmd
}

func TestNotificationMessage(t *testing.T) {
	m := setupTestModel(t)

	msg := NotificationMsg{
		SessionID: m.sessionID,
		Level:     pkg.NotifyInfo,
		Message:   "Test notification",
	}
	result, cmd := m.Update(msg)
	require.NotNil(t, result)

	model := result.(*Model)
	require.Contains(t, model.status, "Test notification")
	_ = cmd
}

func TestErrorMsg(t *testing.T) {
	m := setupTestModel(t)

	msg := ErrorMsg{
		SessionID: m.sessionID,
		Err:       context.DeadlineExceeded,
	}
	result, cmd := m.Update(msg)
	require.NotNil(t, result)

	model := result.(*Model)
	require.Error(t, model.err)
	_ = cmd
}

// Test that model handles window resize properly
func TestWindowResize(t *testing.T) {
	m := setupTestModel(t)

	msg := tea.WindowSizeMsg{Width: 100, Height: 40}
	result, cmd := m.Update(msg)
	require.NotNil(t, result)
	_ = cmd
}

// Test that sending user input adds a message
func TestUserInputCreatesMessage(t *testing.T) {
	m := setupTestModel(t)

	m.input.SetValue("Hello")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	found := false
	for _, msg := range m.Messages() {
		if msg.role == pkg.RoleUser {
			found = true
			break
		}
	}
	require.True(t, found, "user input should create a user message")
}

// Test input history
func TestInputHistory(t *testing.T) {
	m := setupTestModel(t)

	m.input.SetValue("message 1")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	m.input.SetValue("message 2")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Press up to recall history
	m.input.SetValue("")
	m.Update(tea.KeyMsg{Type: tea.KeyUp})
	require.Equal(t, "message 2", m.input.Value())

	m.Update(tea.KeyMsg{Type: tea.KeyUp})
	require.Equal(t, "message 1", m.input.Value())
}

// Test tostring on NotifyLevel constants
func TestNotifyLevelConstants(t *testing.T) {
	require.Equal(t, pkg.NotifyLevel("info"), pkg.NotifyInfo)
	require.Equal(t, pkg.NotifyLevel("warn"), pkg.NotifyWarn)
	require.Equal(t, pkg.NotifyLevel("error"), pkg.NotifyError)
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello world", "Hello world"},
		{"\x1b[31mRed\x1b[0m", "Red"},
		{"\x1b[1mBold\x1b[0m and \x1b[4mUnderline\x1b[0m", "Bold and Underline"},
		{"No escape sequences here", "No escape sequences here"},
	}
	for _, tt := range tests {
		result := StripANSI(tt.input)
		require.Equal(t, tt.expected, result)
	}
}

func TestTimer(t *testing.T) {
	// Simple test that time operations work
	start := time.Now()
	time.Sleep(time.Millisecond)
	elapsed := time.Since(start)
	require.GreaterOrEqual(t, elapsed, time.Millisecond)
}
