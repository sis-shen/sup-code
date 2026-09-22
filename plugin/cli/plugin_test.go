package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/core"
	"github.com/supcode/supcode/internal/contextmgr"
	"github.com/supcode/supcode/internal/llm/openai"
	"github.com/supcode/supcode/internal/tui"
	"github.com/supcode/supcode/pkg"
	"github.com/supcode/supcode/plugin/cli"
)

// fakeSessionManager is an inline pkg.SessionManager used to satisfy plugin-cli
// dependencies without touching the v1 session store.
type fakeSessionManager struct{}

func (fakeSessionManager) Create(context.Context, string) (*pkg.Session, error) {
	return &pkg.Session{ID: "s1"}, nil
}
func (fakeSessionManager) Get(context.Context, string) (*pkg.Session, error) {
	return &pkg.Session{ID: "s1"}, nil
}
func (fakeSessionManager) List(context.Context) ([]*pkg.Session, error) { return nil, nil }
func (fakeSessionManager) AppendMessage(context.Context, string, pkg.Message) error {
	return nil
}
func (fakeSessionManager) SetState(context.Context, string, pkg.LoopState) error { return nil }
func (fakeSessionManager) SetPlan(context.Context, string, pkg.Plan) error       { return nil }
func (fakeSessionManager) Delete(context.Context, string) error                  { return nil }
func (fakeSessionManager) Close(context.Context, string) error                   { return nil }

func fakeSessionsPlugin() core.Plugin {
	return core.Plugin{
		Name:     "fake-sessions",
		Provides: []string{pkg.ServiceSessions},
		Apply: func(ctx *core.Context) error {
			core.Provide[pkg.SessionManager](ctx, pkg.ServiceSessions, fakeSessionManager{})
			return nil
		},
	}
}

// fakeHelpCommand and fakeCommandRegistry keep plugin-cli's tests free of any
// import on plugin/command (arch rule: plugins never import each other).
type fakeHelpCommand struct{}

func (fakeHelpCommand) Name() string        { return "help" }
func (fakeHelpCommand) Description() string { return "show available commands" }
func (fakeHelpCommand) Run(context.Context, string, []string) (pkg.CommandResult, error) {
	return pkg.CommandResult{Handled: true, Message: "commands: /help /version"}, nil
}

type fakeCommandRegistry struct {
	cmds map[string]pkg.SlashCommand
}

func (r *fakeCommandRegistry) Register(cmd pkg.SlashCommand) error {
	r.cmds[cmd.Name()] = cmd
	return nil
}
func (r *fakeCommandRegistry) Unregister(name string) error {
	delete(r.cmds, name)
	return nil
}
func (r *fakeCommandRegistry) Get(name string) (pkg.SlashCommand, bool) {
	cmd, ok := r.cmds[name]
	return cmd, ok
}
func (r *fakeCommandRegistry) List() []pkg.SlashCommand {
	out := make([]pkg.SlashCommand, 0, len(r.cmds))
	for _, cmd := range r.cmds {
		out = append(out, cmd)
	}
	return out
}

func fakeCommandsPlugin() core.Plugin {
	return core.Plugin{
		Name:     "fake-commands",
		Provides: []string{pkg.ServiceCommands},
		Apply: func(ctx *core.Context) error {
			reg := &fakeCommandRegistry{cmds: map[string]pkg.SlashCommand{"help": fakeHelpCommand{}}}
			core.Provide[pkg.CommandRegistry](ctx, pkg.ServiceCommands, reg)
			return nil
		},
	}
}

// loadInteraction wires the fake sessions/commands services and the headless
// CLI plugin, then resolves the provided interaction service.
func loadInteraction(t *testing.T, in io.Reader, out io.Writer, autoConfirm bool) pkg.InteractionService {
	t.Helper()
	k := core.New()
	root := k.Root()
	require.NoError(t, k.Load(root, fakeSessionsPlugin()))
	require.NoError(t, k.Load(root, fakeCommandsPlugin()))
	require.NoError(t, k.Load(root, cli.Plugin(cli.Options{In: in, Out: out, AutoConfirm: autoConfirm})))
	return core.Use[pkg.InteractionService](root, pkg.ServiceInteraction)
}

func TestPluginProvidesInteraction(t *testing.T) {
	var buf bytes.Buffer
	interaction := loadInteraction(t, strings.NewReader(""), &buf, true)

	res, err := interaction.HandleCommand(context.Background(), "s1", "/help")
	require.NoError(t, err)
	assert.True(t, res.Handled)
	assert.Contains(t, res.Message, "/help")
}

func TestPluginRequiresDependencies(t *testing.T) {
	k := core.New()
	err := k.Load(k.Root(), cli.Plugin(cli.Options{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing dependencies")
}

func TestManifest(t *testing.T) {
	m := cli.Manifest()
	assert.Equal(t, "plugin-cli", m.Name)
	assert.Equal(t, "2.0.0", m.Version)
	assert.ElementsMatch(t, []string{pkg.ServiceSessions, pkg.ServiceCommands}, m.Inject)
	assert.Equal(t, []string{pkg.ServiceInteraction}, m.Provides)
	assert.Equal(t, "plugin/cli", m.Entry)
}

func TestStreamResponseWritesText(t *testing.T) {
	var buf bytes.Buffer
	interaction := loadInteraction(t, strings.NewReader(""), &buf, true)

	stream := make(chan pkg.StreamEvent, 3)
	stream <- pkg.StreamEvent{Type: "text_delta", Delta: "hello "}
	stream <- pkg.StreamEvent{Type: "text_delta", Delta: "world"}
	stream <- pkg.StreamEvent{Type: "done"}
	close(stream)

	require.NoError(t, interaction.StreamResponse(context.Background(), "s1", stream))
	assert.Equal(t, "hello world", buf.String())
}

func TestReadInputAndEOF(t *testing.T) {
	var buf bytes.Buffer
	interaction := loadInteraction(t, strings.NewReader("first\nsecond\n"), &buf, true)

	line, err := interaction.ReadInput(context.Background(), "s1")
	require.NoError(t, err)
	assert.Equal(t, "first", line)

	line, err = interaction.ReadInput(context.Background(), "s1")
	require.NoError(t, err)
	assert.Equal(t, "second", line)

	_, err = interaction.ReadInput(context.Background(), "s1")
	assert.ErrorIs(t, err, io.EOF)
}

func TestRequestConfirmationReturnsAutoConfirm(t *testing.T) {
	var buf bytes.Buffer
	confirmed := loadInteraction(t, strings.NewReader(""), &buf, true)
	got, err := confirmed.RequestConfirmation(context.Background(), "s1", pkg.ConfirmPrompt{Title: "run"})
	require.NoError(t, err)
	assert.True(t, got)

	rejected := loadInteraction(t, strings.NewReader(""), &buf, false)
	got, err = rejected.RequestConfirmation(context.Background(), "s1", pkg.ConfirmPrompt{Title: "run"})
	require.NoError(t, err)
	assert.False(t, got)
}

func TestNotifyWritesLine(t *testing.T) {
	var buf bytes.Buffer
	interaction := loadInteraction(t, strings.NewReader(""), &buf, true)

	interaction.Notify(context.Background(), "s1", pkg.NotifyWarn, "careful")
	assert.Equal(t, "[warn] careful\n", buf.String())
}

func TestHandleCommandUnhandledInputs(t *testing.T) {
	var buf bytes.Buffer
	interaction := loadInteraction(t, strings.NewReader(""), &buf, true)

	for _, input := range []string{"plain text", "/unknown", "   "} {
		res, err := interaction.HandleCommand(context.Background(), "s1", input)
		require.NoError(t, err)
		assert.False(t, res.Handled, "input %q should not be handled", input)
	}
}

func TestSingleShotStreamsResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, sseBody("Hello world"))
	}))
	defer server.Close()

	client := openai.NewLLMClient(openai.Config{
		APIKey:    "sk-test",
		BaseURL:   server.URL,
		Model:     "gpt-4o-test",
		MaxTokens: 100,
	})

	sessionMgr, err := tui.NewSessionManager(filepath.Join(t.TempDir(), "sessions.json"))
	require.NoError(t, err)
	session, err := sessionMgr.Create(context.Background(), "Single Shot")
	require.NoError(t, err)

	ctxMgr := contextmgr.NewManager()
	got, err := cli.SingleShot(context.Background(), client, ctxMgr, sessionMgr, session.ID, "hello")
	require.NoError(t, err)
	assert.Equal(t, "Hello world", got)

	_, messages, err := ctxMgr.BuildContext(context.Background(), session.ID)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	assert.Equal(t, pkg.RoleUser, messages[0].Role)
	assert.Equal(t, "hello", messages[0].Content)
	assert.Equal(t, pkg.RoleAssistant, messages[1].Role)
	assert.Equal(t, "Hello world", messages[1].Content)
}

// fakeLLM emits a fixed event sequence, used to exercise the SingleShot error
// path without an HTTP server.
type fakeLLM struct {
	events []pkg.StreamEvent
}

func (f *fakeLLM) Chat(context.Context, string, []pkg.Message, []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
	ch := make(chan pkg.StreamEvent, len(f.events))
	for _, ev := range f.events {
		ch <- ev
	}
	close(ch)
	return ch, nil
}

func (f *fakeLLM) Models(context.Context) ([]pkg.ModelInfo, error) { return nil, nil }
func (f *fakeLLM) ProviderName() string                            { return "fake" }

func TestSingleShotStreamError(t *testing.T) {
	ctxMgr := contextmgr.NewManager()
	llm := &fakeLLM{events: []pkg.StreamEvent{
		{Type: "text_delta", Delta: "partial"},
		{Type: "error", Error: "boom"},
	}}

	got, err := cli.SingleShot(context.Background(), llm, ctxMgr, nil, "s1", "hi")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
	assert.Equal(t, "partial", got)
}

func sseBody(content string) string {
	escaped, _ := json.Marshal(content)
	body := fmt.Sprintf(`data: {"id":"test-id","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":%s},"finish_reason":null}]}`+"\n\n", string(escaped))
	body += `data: {"id":"test-id","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}` + "\n\n"
	body += "data: [DONE]\n\n"
	return body
}
