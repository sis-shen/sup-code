package tui

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/core"
	tui "github.com/supcode/supcode/internal/tui"
	"github.com/supcode/supcode/pkg"
)

// fakeSessionManager is a minimal in-memory pkg.SessionManager used to satisfy
// the sessions dependency without loading plugin-session.
type fakeSessionManager struct{}

func (fakeSessionManager) Create(context.Context, string) (*pkg.Session, error) {
	return &pkg.Session{ID: "s1"}, nil
}

func (fakeSessionManager) Get(context.Context, string) (*pkg.Session, error) {
	return nil, errors.New("not found")
}

func (fakeSessionManager) List(context.Context) ([]*pkg.Session, error) { return nil, nil }

func (fakeSessionManager) AppendMessage(context.Context, string, pkg.Message) error { return nil }

func (fakeSessionManager) SetState(context.Context, string, pkg.LoopState) error { return nil }

func (fakeSessionManager) SetPlan(context.Context, string, pkg.Plan) error { return nil }

func (fakeSessionManager) Delete(context.Context, string) error { return nil }

func (fakeSessionManager) Close(context.Context, string) error { return nil }

// fakeAgent records whether Run was reached so the optional SetAgent wire-up
// can be observed without executing a real engine loop.
type fakeAgent struct{ ran bool }

func (f *fakeAgent) Run(context.Context, string, string) (*pkg.AgentResult, error) {
	f.ran = true
	return &pkg.AgentResult{Summary: "ok"}, nil
}

func (f *fakeAgent) RunPlan(context.Context, string, string) (*pkg.AgentResult, error) {
	return nil, nil
}

func (f *fakeAgent) ApprovePlan(context.Context, string) (*pkg.AgentResult, error) {
	return nil, nil
}

func (f *fakeAgent) GetSession(context.Context, string) (*pkg.Session, error) { return nil, nil }

func (f *fakeAgent) ListTools() []string { return nil }

func (f *fakeAgent) ListHooks() []string { return nil }

func (f *fakeAgent) ExecuteTool(context.Context, string, json.RawMessage) (pkg.ToolResult, error) {
	return pkg.ToolResult{}, nil
}

// fakeSessionsPlugin provides a caller-supplied pkg.SessionManager so the test
// can control what the interaction plugin resolves.
func fakeSessionsPlugin(sm pkg.SessionManager) core.Plugin {
	return core.Plugin{
		Name:     "fake-sessions",
		Provides: []string{pkg.ServiceSessions},
		Apply: func(ctx *core.Context) error {
			core.Provide(ctx, pkg.ServiceSessions, sm)
			return nil
		},
	}
}

// loadTUI loads the fake sessions provider followed by the interaction plugin.
func loadTUI(t *testing.T) (*core.Kernel, *core.Context) {
	t.Helper()
	k := core.New()
	root := k.Root()
	require.NoError(t, k.Load(root, fakeSessionsPlugin(fakeSessionManager{})))
	require.NoError(t, k.Load(root, Plugin(Options{})))
	t.Cleanup(func() { _ = k.Shutdown(context.Background()) })
	return k, root
}

func TestPluginProvidesInteractionService(t *testing.T) {
	k, root := loadTUI(t)

	svc := core.Use[pkg.InteractionService](root, pkg.ServiceInteraction)
	require.NotNil(t, svc)
	_, ok := svc.(*tui.Service)
	assert.True(t, ok, "interaction should be backed by the v1 *tui.Service")

	assert.ElementsMatch(t, []string{"fake-sessions", pluginName}, k.Plugins())
}

func TestPluginMissingSessionsFails(t *testing.T) {
	k := core.New()

	err := k.Load(k.Root(), Plugin(Options{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), pkg.ServiceSessions)
	assert.False(t, k.Root().Has(pkg.ServiceInteraction))
	assert.Empty(t, k.Plugins())
}

func TestPluginNotifyAndHandleCommandWithoutProgram(t *testing.T) {
	_, root := loadTUI(t)
	svc := core.Use[pkg.InteractionService](root, pkg.ServiceInteraction)
	ctx := context.Background()

	require.NotPanics(t, func() {
		svc.Notify(ctx, "s1", pkg.NotifyInfo, "hello")
	})

	res, err := svc.HandleCommand(ctx, "s1", "/help")
	require.NoError(t, err)
	assert.True(t, res.Handled)
	assert.Contains(t, res.Message, "Available commands")

	res, err = svc.HandleCommand(ctx, "s1", "/new")
	require.NoError(t, err)
	assert.True(t, res.Handled)
	assert.True(t, res.NewSession)

	res, err = svc.HandleCommand(ctx, "s1", "plain text")
	require.NoError(t, err)
	assert.False(t, res.Handled)

	res, err = svc.HandleCommand(ctx, "s1", "/bogus")
	require.NoError(t, err)
	assert.False(t, res.Handled)
	assert.Contains(t, res.Message, "Unknown command")
}

// RequestConfirmation and ReadInput route through Bubble Tea channels that only
// a running program (or an explicit SubmitConfirmation/SubmitInput) can
// satisfy. Rather than block on their 60s/5min fallbacks, this test drives the
// documented context-cancellation path: the v1 Service is safe to construct and
// returns ctx.Err() when no program is attached.
func TestPluginConfirmationAndInputNeedProgram(t *testing.T) {
	_, root := loadTUI(t)
	svc := core.Use[pkg.InteractionService](root, pkg.ServiceInteraction)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	approved, err := svc.RequestConfirmation(ctx, "s1", pkg.ConfirmPrompt{Title: "write", Message: "?"})
	require.ErrorIs(t, err, context.Canceled)
	assert.False(t, approved)

	input, err := svc.ReadInput(ctx, "s1")
	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, input)
}

func TestPluginStreamResponseWithoutProgram(t *testing.T) {
	_, root := loadTUI(t)
	svc := core.Use[pkg.InteractionService](root, pkg.ServiceInteraction)

	stream := make(chan pkg.StreamEvent)
	close(stream)

	require.NoError(t, svc.StreamResponse(context.Background(), "s1", stream))
}

func TestPluginWiresOptionalAgent(t *testing.T) {
	k := core.New()
	root := k.Root()
	core.Provide[pkg.SessionManager](root, pkg.ServiceSessions, fakeSessionManager{})

	agent := &fakeAgent{}
	core.Provide[pkg.Agent](root, agentServiceKey, agent)

	require.NoError(t, k.Load(root, Plugin(Options{})))

	svc, ok := core.Use[pkg.InteractionService](root, pkg.ServiceInteraction).(*tui.Service)
	require.True(t, ok)

	cmd := svc.RunQueryCmd(context.Background(), "s1", "hi")
	require.NotNil(t, cmd)
	_ = cmd()
	assert.True(t, agent.ran)
}

func TestManifest(t *testing.T) {
	m := Manifest()
	assert.Equal(t, pluginName, m.Name)
	assert.Equal(t, pluginVersion, m.Version)
	assert.Equal(t, []string{pkg.ServiceSessions}, m.Inject)
	assert.Equal(t, []string{pkg.ServiceInteraction}, m.Provides)
	assert.Equal(t, "builtin:tui", m.Entry)

	p := Plugin(Options{})
	assert.Equal(t, m.Name, p.Name)
	assert.Equal(t, m.Inject, p.Inject)
	assert.Equal(t, m.Provides, p.Provides)
}
