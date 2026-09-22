package context

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/core"
	"github.com/supcode/supcode/pkg"
)

// fakeLLM is a deterministic pkg.LLMClient returning a fixed summary stream.
type fakeLLM struct{}

func (fakeLLM) Chat(_ context.Context, _ string, _ []pkg.Message, _ []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
	ch := make(chan pkg.StreamEvent, 2)
	ch <- pkg.StreamEvent{Type: "text_delta", Delta: "We decided to use Go for the project."}
	ch <- pkg.StreamEvent{Type: "done"}
	close(ch)
	return ch, nil
}

func (fakeLLM) Models(context.Context) ([]pkg.ModelInfo, error) { return nil, nil }

func (fakeLLM) ProviderName() string { return "fake" }

var _ pkg.LLMClient = fakeLLM{}

func loadKernel(t *testing.T, opts Options) *core.Kernel {
	t.Helper()
	k := core.New()
	require.NoError(t, k.LoadPlugins(k.Root(), []core.Plugin{Plugin(opts)}))
	t.Cleanup(func() { _ = k.Shutdown(context.Background()) })
	return k
}

func TestPluginProvidesContextManager(t *testing.T) {
	k := loadKernel(t, Options{})

	mgr := core.Use[pkg.ContextManager](k.Root(), pkg.ServiceContext)
	require.NotNil(t, mgr)
	assert.Equal(t, []string{"plugin-context"}, k.Plugins())
}

func TestAppendMessageAndBuildContext(t *testing.T) {
	k := loadKernel(t, Options{})
	ctx := context.Background()
	mgr := core.Use[pkg.ContextManager](k.Root(), pkg.ServiceContext)
	sessionID := "sess-build"

	initial, err := mgr.TokenCount(ctx, sessionID)
	require.NoError(t, err)
	require.Zero(t, initial)

	require.NoError(t, mgr.AppendMessage(ctx, sessionID, pkg.Message{Role: pkg.RoleSystem, Content: "You are helpful."}))
	afterFirst, err := mgr.TokenCount(ctx, sessionID)
	require.NoError(t, err)
	assert.Positive(t, afterFirst)

	require.NoError(t, mgr.AppendMessage(ctx, sessionID, pkg.Message{Role: pkg.RoleUser, Content: "hello"}))
	afterSecond, err := mgr.TokenCount(ctx, sessionID)
	require.NoError(t, err)
	assert.Greater(t, afterSecond, afterFirst)

	// No memory cards yet, so the v1 manager returns an empty system prompt.
	systemPrompt, messages, err := mgr.BuildContext(ctx, sessionID)
	require.NoError(t, err)
	assert.Empty(t, systemPrompt)
	require.Len(t, messages, 2)
	assert.Equal(t, pkg.RoleSystem, messages[0].Role)
	assert.Equal(t, "You are helpful.", messages[0].Content)
	assert.Equal(t, pkg.RoleUser, messages[1].Role)
	assert.Equal(t, "hello", messages[1].Content)
}

func TestThresholdTriggersCompression(t *testing.T) {
	const threshold = 50
	k := loadKernel(t, Options{Threshold: threshold})
	ctx := context.Background()
	mgr := core.Use[pkg.ContextManager](k.Root(), pkg.ServiceContext)
	sessionID := "sess-threshold"

	should, err := mgr.ShouldCompress(ctx, sessionID)
	require.NoError(t, err)
	assert.False(t, should)

	require.NoError(t, mgr.AppendMessage(ctx, sessionID, pkg.Message{
		Role:    pkg.RoleUser,
		Content: strings.Repeat("context token ", 20),
	}))

	should, err = mgr.ShouldCompress(ctx, sessionID)
	require.NoError(t, err)
	assert.True(t, should)
}

func TestCompressProducesSummary(t *testing.T) {
	k := loadKernel(t, Options{LLM: fakeLLM{}})
	ctx := context.Background()
	mgr := core.Use[pkg.ContextManager](k.Root(), pkg.ServiceContext)
	sessionID := "sess-compress"

	appendMessages(t, mgr, sessionID, 30)
	require.NoError(t, mgr.Compress(ctx, sessionID))

	cards := waitForCards(t, mgr, sessionID)
	require.NotEmpty(t, cards)
	for _, c := range cards {
		assert.NotEmpty(t, c.Content)
	}
	assert.Equal(t, "decision", cards[0].Category)
}

func TestInjectedLLMEnablesCompression(t *testing.T) {
	k := core.New()
	core.Provide[pkg.LLMClient](k.Root(), pkg.ServiceLLM, fakeLLM{})
	require.NoError(t, k.LoadPlugins(k.Root(), []core.Plugin{Plugin(Options{})}))
	t.Cleanup(func() { _ = k.Shutdown(context.Background()) })

	ctx := context.Background()
	mgr := core.Use[pkg.ContextManager](k.Root(), pkg.ServiceContext)
	sessionID := "sess-injected"

	appendMessages(t, mgr, sessionID, 30)
	require.NoError(t, mgr.Compress(ctx, sessionID))

	assert.NotEmpty(t, waitForCards(t, mgr, sessionID))
}

func TestManifestMatchesPlugin(t *testing.T) {
	m := Manifest()
	assert.Equal(t, "plugin-context", m.Name)
	assert.Equal(t, "2.0.0", m.Version)
	assert.Equal(t, []string{"context"}, m.Provides)
	assert.Equal(t, "builtin:context", m.Entry)
}

func appendMessages(t *testing.T, mgr pkg.ContextManager, sessionID string, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		require.NoError(t, mgr.AppendMessage(context.Background(), sessionID, pkg.Message{
			Role:    pkg.RoleUser,
			Content: "message",
		}))
	}
}

// waitForCards polls until the asynchronous compression goroutine publishes its
// summary (observable as extracted memory cards), or fails on timeout.
func waitForCards(t *testing.T, mgr pkg.ContextManager, sessionID string) []pkg.MemoryCard {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		cards, err := mgr.GetMemoryCards(context.Background(), sessionID)
		require.NoError(t, err)
		if len(cards) > 0 {
			return cards
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("compression did not produce memory cards within deadline")
	return nil
}
