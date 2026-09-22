package cli_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/core"
	"github.com/supcode/supcode/internal/contextmgr"
	"github.com/supcode/supcode/pkg"
	"github.com/supcode/supcode/plugin/cli"
)

func TestStreamResponseErrorEvent(t *testing.T) {
	var buf bytes.Buffer
	interaction := loadInteraction(t, strings.NewReader(""), &buf, true)

	ch := make(chan pkg.StreamEvent, 1)
	ch <- pkg.StreamEvent{Type: "error", Error: "boom"}
	close(ch)

	require.NoError(t, interaction.StreamResponse(context.Background(), "s1", ch))
	assert.Contains(t, buf.String(), "error: boom")
}

func TestStreamResponseCanceled(t *testing.T) {
	interaction := loadInteraction(t, strings.NewReader(""), &bytes.Buffer{}, true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := interaction.StreamResponse(ctx, "s1", make(chan pkg.StreamEvent))
	assert.ErrorIs(t, err, context.Canceled)
}

func TestReadInputWithoutTrailingNewline(t *testing.T) {
	interaction := loadInteraction(t, strings.NewReader("last"), &bytes.Buffer{}, true)
	line, err := interaction.ReadInput(context.Background(), "s1")
	require.NoError(t, err)
	assert.Equal(t, "last", line)
}

func TestHandleCommandEdgeCases(t *testing.T) {
	interaction := loadInteraction(t, strings.NewReader(""), &bytes.Buffer{}, true)

	for _, input := range []string{"/", "  ", "/only-with-no-name"} {
		res, err := interaction.HandleCommand(context.Background(), "s1", input)
		require.NoError(t, err)
		assert.False(t, res.Handled, "input %q should not be handled", input)
	}
}

func TestPluginDefaultInOut(t *testing.T) {
	k := core.New()
	root := k.Root()
	require.NoError(t, k.Load(root, fakeSessionsPlugin()))
	require.NoError(t, k.Load(root, fakeCommandsPlugin()))
	require.NoError(t, k.Load(root, cli.Plugin(cli.Options{}))) // nil In/Out -> os defaults
	require.NotNil(t, core.Use[pkg.InteractionService](root, pkg.ServiceInteraction))
}

type chatErrLLM struct{}

func (chatErrLLM) Chat(context.Context, string, []pkg.Message, []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
	return nil, errors.New("chat failed")
}
func (chatErrLLM) Models(context.Context) ([]pkg.ModelInfo, error) { return nil, nil }
func (chatErrLLM) ProviderName() string                            { return "chat-err" }

func TestSingleShotNilAndChatError(t *testing.T) {
	_, err := cli.SingleShot(context.Background(), nil, nil, nil, "s1", "x")
	assert.Error(t, err)

	_, err = cli.SingleShot(context.Background(), chatErrLLM{}, contextmgr.NewManager(), nil, "s1", "x")
	assert.Error(t, err)
}
