package builtin

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/pkg"
)

type mockPermissionEngine struct {
	mu            sync.Mutex
	loggedActions []logEntry
}

type logEntry struct {
	action   pkg.Action
	decision pkg.Decision
	result   string
}

func (m *mockPermissionEngine) Check(ctx context.Context, action pkg.Action) (pkg.Decision, error) {
	return pkg.DecisionAllow, nil
}
func (m *mockPermissionEngine) AddRule(rule pkg.PermissionRule) error { return nil }
func (m *mockPermissionEngine) RemoveRule(ruleID string) error        { return nil }
func (m *mockPermissionEngine) ListRules() []pkg.PermissionRule       { return nil }
func (m *mockPermissionEngine) LogAction(ctx context.Context, action pkg.Action, decision pkg.Decision, result string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loggedActions = append(m.loggedActions, logEntry{action, decision, result})
	return nil
}

func (m *mockPermissionEngine) actions() []logEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]logEntry(nil), m.loggedActions...)
}

func (m *mockPermissionEngine) actionCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.loggedActions)
}

func TestAuditHook_Basic(t *testing.T) {
	mockEng := &mockPermissionEngine{}
	hook := NewAuditHook(mockEng)

	assert.Equal(t, "audit_log", hook.Name())

	params := json.RawMessage(`{"path":"/tmp/test.txt"}`)
	result, err := hook.BeforeTool(context.Background(), "write_file", params)
	require.NoError(t, err)
	assert.Equal(t, params, result)

	err = hook.AfterTool(context.Background(), "write_file", params, pkg.ToolResult{Success: true, Data: json.RawMessage(`{}`)})
	require.NoError(t, err)

	assert.Eventually(t, func() bool { return mockEng.actionCount() == 1 }, time.Second, 10*time.Millisecond)
}

func TestAuditHook_LogsFailedTool(t *testing.T) {
	mockEng := &mockPermissionEngine{}
	hook := NewAuditHook(mockEng)

	err := hook.AfterTool(context.Background(), "bash", json.RawMessage(`{"command":"bad cmd"}`), pkg.ToolResult{Success: false, Error: "command not found"})
	require.NoError(t, err)

	assert.Eventually(t, func() bool { return mockEng.actionCount() == 1 }, time.Second, 10*time.Millisecond)

	entry := mockEng.actions()[0]
	assert.Equal(t, pkg.DecisionDeny, entry.decision)
	assert.Contains(t, entry.result, "command not found")
}

func TestAuditHook_TruncatesParams(t *testing.T) {
	mockEng := &mockPermissionEngine{}
	hook := NewAuditHook(mockEng)

	longParams := make([]byte, 2000)
	for i := range longParams {
		longParams[i] = 'a'
	}

	err := hook.AfterTool(context.Background(), "write_file", json.RawMessage(longParams), pkg.ToolResult{Success: true})
	require.NoError(t, err)

	assert.Eventually(t, func() bool { return mockEng.actionCount() == 1 }, time.Second, 10*time.Millisecond)

	entry := mockEng.actions()[0]
	assert.LessOrEqual(t, len(entry.action.Params), 1024)
}
