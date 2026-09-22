package session

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/core"
	"github.com/supcode/supcode/pkg"
)

// dbPath returns a fresh session-store path inside the test's temp dir.
func dbPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "sessions.db")
}

// loadKernel loads the session plugin onto a fresh kernel and shuts it down on
// test cleanup.
func loadKernel(t *testing.T, opts Options) *core.Kernel {
	t.Helper()
	k := core.New()
	require.NoError(t, k.Load(k.Root(), Plugin(opts)))
	t.Cleanup(func() { _ = k.Shutdown(context.Background()) })
	return k
}

func TestPluginProvidesSessionManager(t *testing.T) {
	k := loadKernel(t, Options{DBPath: dbPath(t)})

	sm := core.Use[pkg.SessionManager](k.Root(), pkg.ServiceSessions)
	require.NotNil(t, sm)
	assert.Equal(t, []string{pluginName}, k.Plugins())
}

func TestPluginPersistenceRoundTrip(t *testing.T) {
	ctx := context.Background()
	path := dbPath(t)

	k1 := core.New()
	require.NoError(t, k1.Load(k1.Root(), Plugin(Options{DBPath: path})))
	sm1 := core.Use[pkg.SessionManager](k1.Root(), pkg.ServiceSessions)

	s, err := sm1.Create(ctx, "persisted")
	require.NoError(t, err)
	require.NoError(t, sm1.AppendMessage(ctx, s.ID, pkg.Message{
		Role:    pkg.RoleUser,
		Content: "hello",
	}))

	require.NoError(t, k1.Shutdown(ctx))

	k2 := core.New()
	require.NoError(t, k2.Load(k2.Root(), Plugin(Options{DBPath: path})))
	t.Cleanup(func() { _ = k2.Shutdown(context.Background()) })
	sm2 := core.Use[pkg.SessionManager](k2.Root(), pkg.ServiceSessions)

	restored, err := sm2.Get(ctx, s.ID)
	require.NoError(t, err)
	require.NotNil(t, restored)
	assert.Equal(t, "persisted", restored.Title)
	require.Len(t, restored.Messages, 1)
	assert.Equal(t, "hello", restored.Messages[0].Content)
}

func TestPluginListAndDelete(t *testing.T) {
	ctx := context.Background()
	k := loadKernel(t, Options{DBPath: dbPath(t)})
	sm := core.Use[pkg.SessionManager](k.Root(), pkg.ServiceSessions)

	s, err := sm.Create(ctx, "listed")
	require.NoError(t, err)

	list, err := sm.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, s.ID, list[0].ID)

	require.NoError(t, sm.Delete(ctx, s.ID))

	list, err = sm.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, list)

	_, err = sm.Get(ctx, s.ID)
	require.Error(t, err)
}

func TestPluginEmptyDBPathErrors(t *testing.T) {
	err := Plugin(Options{}).Apply(core.New().Root())
	require.Error(t, err)
}

func TestPluginLoadEmptyDBPathErrors(t *testing.T) {
	k := core.New()
	err := k.Load(k.Root(), Plugin(Options{}))
	require.Error(t, err)
}

func TestManifestMatchesPlugin(t *testing.T) {
	m := Manifest()
	assert.Equal(t, pluginName, m.Name)
	assert.Equal(t, pluginVersion, m.Version)
	assert.Equal(t, []string{"sessions"}, m.Provides)
	assert.Equal(t, "builtin:session", m.Entry)

	p := Plugin(Options{})
	assert.Equal(t, m.Name, p.Name)
	assert.Equal(t, m.Provides, p.Provides)
	assert.Empty(t, p.Inject)
}
