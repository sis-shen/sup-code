package command

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/core"
	"github.com/supcode/supcode/pkg"
)

// fakeCommand is a minimal pkg.SlashCommand used to exercise the registry.
type fakeCommand struct {
	name string
}

func (f *fakeCommand) Name() string        { return f.name }
func (f *fakeCommand) Description() string { return "fake " + f.name }
func (f *fakeCommand) Run(context.Context, string, []string) (pkg.CommandResult, error) {
	return pkg.CommandResult{Handled: true, Message: f.name}, nil
}

func TestRegistryRegisterGetList(t *testing.T) {
	reg := newRegistry()
	require.NoError(t, reg.Register(&fakeCommand{name: "b"}))
	require.NoError(t, reg.Register(&fakeCommand{name: "a"}))

	got, ok := reg.Get("a")
	require.True(t, ok)
	assert.Equal(t, "a", got.Name())

	list := reg.List()
	require.Len(t, list, 2)
	assert.Equal(t, "a", list[0].Name())
	assert.Equal(t, "b", list[1].Name())
}

func TestRegistryRegisterReplacesDuplicate(t *testing.T) {
	reg := newRegistry()
	require.NoError(t, reg.Register(&fakeCommand{name: "x"}))

	replacement := &fakeCommand{name: "x"}
	require.NoError(t, reg.Register(replacement))

	got, ok := reg.Get("x")
	require.True(t, ok)
	assert.Same(t, replacement, got)
	assert.Len(t, reg.List(), 1)
}

func TestRegistryUnregister(t *testing.T) {
	reg := newRegistry()
	require.NoError(t, reg.Register(&fakeCommand{name: "x"}))

	require.NoError(t, reg.Unregister("x"))
	_, ok := reg.Get("x")
	assert.False(t, ok)
	assert.Empty(t, reg.List())

	// Removing an unknown command is a no-op.
	require.NoError(t, reg.Unregister("missing"))
}

func TestRegistryRejectsNilAndEmpty(t *testing.T) {
	reg := newRegistry()
	assert.Error(t, reg.Register(nil))
	assert.Error(t, reg.Register(&fakeCommand{name: ""}))
}

func TestPluginRegistersBuiltins(t *testing.T) {
	k := core.New()
	root := k.Root()
	require.NoError(t, k.Load(root, Plugin(Options{})))

	reg := core.Use[pkg.CommandRegistry](root, pkg.ServiceCommands)
	names := make([]string, 0, 2)
	for _, cmd := range reg.List() {
		names = append(names, cmd.Name())
	}
	assert.Equal(t, []string{"help", "version"}, names)
}

func TestHelpCommandReturnsHandled(t *testing.T) {
	k := core.New()
	root := k.Root()
	require.NoError(t, k.Load(root, Plugin(Options{})))
	reg := core.Use[pkg.CommandRegistry](root, pkg.ServiceCommands)

	help, ok := reg.Get("help")
	require.True(t, ok)

	res, err := help.Run(context.Background(), "s1", nil)
	require.NoError(t, err)
	assert.True(t, res.Handled)
	assert.Contains(t, res.Message, "/help")
	assert.Contains(t, res.Message, "/version")
}

func TestVersionCommandUsesOption(t *testing.T) {
	k := core.New()
	root := k.Root()
	require.NoError(t, k.Load(root, Plugin(Options{Version: "9.9.9"})))
	reg := core.Use[pkg.CommandRegistry](root, pkg.ServiceCommands)

	version, ok := reg.Get("version")
	require.True(t, ok)

	res, err := version.Run(context.Background(), "s1", nil)
	require.NoError(t, err)
	assert.True(t, res.Handled)
	assert.Equal(t, "9.9.9", res.Message)
}

func TestManifest(t *testing.T) {
	m := Manifest()
	assert.Equal(t, pluginName, m.Name)
	assert.Equal(t, pluginVersion, m.Version)
	assert.Equal(t, []string{pkg.ServiceCommands}, m.Provides)
	assert.Empty(t, m.Inject)
	assert.Equal(t, "plugin/command", m.Entry)
}
