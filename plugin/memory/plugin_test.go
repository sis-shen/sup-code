package memory

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/core"
	"github.com/supcode/supcode/pkg"
)

// fakeEmbedder is a deterministic EmbeddingProvider for exercising the
// semantic-search branch of the store.
type fakeEmbedder struct {
	calls int
}

func (f *fakeEmbedder) Generate(_ context.Context, text string) ([]float32, error) {
	f.calls++
	return []float32{float32(len(text)), 1, 0}, nil
}

func (f *fakeEmbedder) BatchGenerate(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, t := range texts {
		v, err := f.Generate(ctx, t)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

func dbPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "mem.db")
}

func loadKernel(t *testing.T, opts Options) *core.Kernel {
	t.Helper()
	k := core.New()
	require.NoError(t, k.LoadPlugins(k.Root(), []core.Plugin{Plugin(opts)}))
	t.Cleanup(func() { _ = k.Shutdown(context.Background()) })
	return k
}

func TestPluginProvidesMemoryStore(t *testing.T) {
	k := loadKernel(t, Options{DBPath: dbPath(t)})

	store := core.Use[pkg.MemoryStore](k.Root(), pkg.ServiceMemory)
	require.NotNil(t, store)
	assert.Equal(t, []string{"plugin-memory"}, k.Plugins())
}

func TestPluginSaveSearchRoundTrip(t *testing.T) {
	k := loadKernel(t, Options{DBPath: dbPath(t)})
	ctx := context.Background()
	store := core.Use[pkg.MemoryStore](k.Root(), pkg.ServiceMemory)

	require.NoError(t, store.Save(ctx, pkg.MemoryEntry{
		ID:       "m1",
		Scope:    "project",
		Category: "structure",
		Content:  "alpha beta gamma",
	}))

	found, err := store.Search(ctx, "beta", "project", 10)
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, "m1", found[0].ID)
	assert.Equal(t, "alpha beta gamma", found[0].Content)

	byCat, err := store.GetByCategory(ctx, "project", "structure")
	require.NoError(t, err)
	require.Len(t, byCat, 1)
	assert.Equal(t, "m1", byCat[0].ID)
}

func TestPluginWithEmbedder(t *testing.T) {
	emb := &fakeEmbedder{}
	k := loadKernel(t, Options{DBPath: dbPath(t), Embedder: emb})
	ctx := context.Background()
	store := core.Use[pkg.MemoryStore](k.Root(), pkg.ServiceMemory)

	require.NoError(t, store.Save(ctx, pkg.MemoryEntry{
		ID:        "m1",
		Scope:     "project",
		Category:  "note",
		Content:   "semantic memory",
		Embedding: []float32{1, 0, 0},
	}))

	found, err := store.Search(ctx, "semantic", "project", 10)
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, "m1", found[0].ID)
	assert.Positive(t, emb.calls)
}

func TestPluginUpdateEmbedding(t *testing.T) {
	k := loadKernel(t, Options{DBPath: dbPath(t)})
	ctx := context.Background()
	store := core.Use[pkg.MemoryStore](k.Root(), pkg.ServiceMemory)

	require.NoError(t, store.Save(ctx, pkg.MemoryEntry{
		ID:       "m1",
		Scope:    "project",
		Category: "note",
		Content:  "backfill me",
	}))

	vec := []float32{0.25, 0.5, 0.75}
	require.NoError(t, store.UpdateEmbedding(ctx, "m1", vec))

	found, err := store.Search(ctx, "backfill", "project", 10)
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, vec, found[0].Embedding)
}

func TestPluginEmptyDBPathErrors(t *testing.T) {
	err := Plugin(Options{}).Apply(core.New().Root())
	require.Error(t, err)
}

func TestPluginLoadEmptyDBPathErrors(t *testing.T) {
	k := core.New()
	err := k.LoadPlugins(k.Root(), []core.Plugin{Plugin(Options{})})
	require.Error(t, err)
}

func TestUnloadClosesStore(t *testing.T) {
	k := loadKernel(t, Options{DBPath: dbPath(t)})
	ctx := context.Background()
	store := core.Use[pkg.MemoryStore](k.Root(), pkg.ServiceMemory)

	require.NoError(t, k.Unload("plugin-memory"))
	assert.Error(t, store.Save(ctx, pkg.MemoryEntry{ID: "m2", Content: "after close"}))
	require.NotPanics(t, func() { _ = store.Close() })
}

func TestManifestMatchesPlugin(t *testing.T) {
	m := Manifest()
	assert.Equal(t, "plugin-memory", m.Name)
	assert.Equal(t, "2.0.0", m.Version)
	assert.Equal(t, []string{"memory"}, m.Provides)
	assert.Equal(t, "builtin:memory", m.Entry)
}
