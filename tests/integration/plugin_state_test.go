package integration

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/core"
	"github.com/supcode/supcode/pkg"
	cliplugin "github.com/supcode/supcode/plugin/cli"
	commandplugin "github.com/supcode/supcode/plugin/command"
	contextplugin "github.com/supcode/supcode/plugin/context"
	sessionplugin "github.com/supcode/supcode/plugin/session"
)

type fakeLLM struct{ reply string }

func (f *fakeLLM) Chat(_ context.Context, _ string, _ []pkg.Message, _ []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
	ch := make(chan pkg.StreamEvent, 2)
	ch <- pkg.StreamEvent{Type: "text_delta", Delta: f.reply}
	ch <- pkg.StreamEvent{Type: "done"}
	close(ch)
	return ch, nil
}

func (f *fakeLLM) Models(context.Context) ([]pkg.ModelInfo, error) { return nil, nil }
func (f *fakeLLM) ProviderName() string                            { return "fake" }

// TestPluginState_HeadlessSingleShot wires the Phase 3 state/interaction
// plugins (session, context, command, headless CLI) into one kernel and
// exercises a locally-handled slash command plus a mock-LLM single-shot query.
func TestPluginState_HeadlessSingleShot(t *testing.T) {
	k := core.New()
	root := k.Root()
	dbPath := filepath.Join(t.TempDir(), "sessions.db")
	var out bytes.Buffer

	require.NoError(t, k.LoadPlugins(root, []core.Plugin{
		sessionplugin.Plugin(sessionplugin.Options{DBPath: dbPath}),
		contextplugin.Plugin(contextplugin.Options{LLM: &fakeLLM{reply: "unused"}}),
		commandplugin.Plugin(commandplugin.Options{Version: "test"}),
		cliplugin.Plugin(cliplugin.Options{Out: &out, AutoConfirm: true}),
	}))

	sessions := core.Use[pkg.SessionManager](root, pkg.ServiceSessions)
	ctxMgr := core.Use[pkg.ContextManager](root, pkg.ServiceContext)
	interaction := core.Use[pkg.InteractionService](root, pkg.ServiceInteraction)

	sess, err := sessions.Create(context.Background(), "test")
	require.NoError(t, err)

	// A slash command is handled locally, without touching the model.
	res, err := interaction.HandleCommand(context.Background(), sess.ID, "/help")
	require.NoError(t, err)
	assert.True(t, res.Handled)

	// Single-shot query against a mock LLM.
	text, err := cliplugin.SingleShot(context.Background(), &fakeLLM{reply: "hello from llm"}, ctxMgr, sessions, sess.ID, "hello")
	require.NoError(t, err)
	assert.Equal(t, "hello from llm", text)

	require.NoError(t, k.Shutdown(context.Background()))
	assert.Empty(t, k.Plugins())
}
