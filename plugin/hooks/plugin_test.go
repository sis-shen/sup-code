package hooks

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/core"
	"github.com/supcode/supcode/pkg"
)

type recordedAction struct {
	action   pkg.Action
	decision pkg.Decision
	result   string
}

// fakePermissionEngine implements pkg.PermissionEngine and records LogAction
// calls so tests can assert on the audit hook's asynchronous output.
type fakePermissionEngine struct {
	mu      sync.Mutex
	entries []recordedAction
}

func (f *fakePermissionEngine) Check(context.Context, pkg.Action) (pkg.Decision, error) {
	return pkg.DecisionAllow, nil
}

func (f *fakePermissionEngine) AddRule(pkg.PermissionRule) error { return nil }

func (f *fakePermissionEngine) RemoveRule(string) error { return nil }

func (f *fakePermissionEngine) ListRules() []pkg.PermissionRule { return nil }

func (f *fakePermissionEngine) LogAction(_ context.Context, action pkg.Action, decision pkg.Decision, result string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, recordedAction{action: action, decision: decision, result: result})
	return nil
}

func (f *fakePermissionEngine) actionCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.entries)
}

func (f *fakePermissionEngine) actions() []recordedAction {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]recordedAction(nil), f.entries...)
}

func loadFakePermission(t *testing.T, k *core.Kernel, eng pkg.PermissionEngine) {
	t.Helper()
	require.NoError(t, k.Load(k.Root(), core.Plugin{
		Name:     "fake-permission",
		Provides: []string{pkg.ServicePermission},
		Apply: func(ctx *core.Context) error {
			core.Provide(ctx, pkg.ServicePermission, eng)
			return nil
		},
	}))
}

func TestAuditPluginRecordsAction(t *testing.T) {
	k := core.New()
	eng := &fakePermissionEngine{}
	loadFakePermission(t, k, eng)

	require.NoError(t, k.Load(k.Root(), AuditPlugin(AuditOptions{})))

	inv := &pkg.ToolInvocation{
		Name:   "bash",
		Params: json.RawMessage(`{}`),
		Result: pkg.ToolResult{Success: true},
	}
	core.Emit(context.Background(), k, pkg.EventToolsResult, inv)

	require.Eventually(t, func() bool { return eng.actionCount() == 1 }, time.Second, 10*time.Millisecond)

	entry := eng.actions()[0]
	assert.Equal(t, "bash", entry.action.ToolName)
	assert.Equal(t, "tool", entry.action.Type)
	assert.Equal(t, pkg.DecisionAllow, entry.decision)
	assert.Equal(t, "ok", entry.result)
}

func TestAuditPluginMissingPermission(t *testing.T) {
	k := core.New()
	err := k.Load(k.Root(), AuditPlugin(AuditOptions{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing dependencies")
}

func TestAuditPluginManifest(t *testing.T) {
	m := Manifest()
	assert.Equal(t, "plugin-hooks-audit", m.Name)
	assert.Equal(t, []string{pkg.ServicePermission}, m.Inject)
	assert.Empty(t, m.Provides)
}

func TestGitPluginDisabledNoOp(t *testing.T) {
	k := core.New()
	require.NoError(t, k.Load(k.Root(), GitPlugin(false)))

	inv := &pkg.ToolInvocation{
		Name:   "write_file",
		Params: json.RawMessage(`{"path":"test.txt"}`),
		Result: pkg.ToolResult{Success: true},
	}
	got, err := core.Waterfall[*pkg.ToolInvocation, *pkg.ToolInvocation](context.Background(), k, pkg.EventToolsPostExecute, inv)
	require.NoError(t, err)
	assert.Same(t, inv, got)
}

func TestGitPluginEnabledNonMutationNoOp(t *testing.T) {
	k := core.New()
	require.NoError(t, k.Load(k.Root(), GitPlugin(true)))

	inv := &pkg.ToolInvocation{
		Name:   "bash",
		Params: json.RawMessage(`{"command":"echo hi"}`),
		Result: pkg.ToolResult{Success: true},
	}
	got, err := core.Waterfall[*pkg.ToolInvocation, *pkg.ToolInvocation](context.Background(), k, pkg.EventToolsPostExecute, inv)
	require.NoError(t, err)
	assert.Same(t, inv, got)
}

func TestGitPluginManifest(t *testing.T) {
	m := GitManifest()
	assert.Equal(t, "plugin-hooks-git", m.Name)
	assert.Empty(t, m.Inject)
	assert.Empty(t, m.Provides)
}
