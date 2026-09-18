// Package memory migrates the v1 long-term memory store (internal/memory) onto
// the Cordis kernel as the plugin-memory leaf plugin. It only assembles the v1
// implementation and exposes it as pkg.ServiceMemory; business logic is reused
// unchanged.
package memory

import (
	"errors"

	"github.com/supcode/supcode/core"
	memory "github.com/supcode/supcode/internal/memory"
	"github.com/supcode/supcode/pkg"
)

// pluginName is the stable identifier of the memory plugin.
const pluginName = "plugin-memory"

// Options configures the memory plugin. DBPath is the SQLite database location
// and is required; Embedder is optional and enables semantic search when set.
type Options struct {
	DBPath   string
	Embedder pkg.EmbeddingProvider
}

// Plugin returns the plugin-memory leaf plugin. It opens the v1 SQLite store,
// provides it under pkg.ServiceMemory and registers its Close as a scope
// effect so unloading the plugin releases the database.
func Plugin(opts Options) core.Plugin {
	return core.Plugin{
		Name:     pluginName,
		Inject:   []string{},
		Provides: []string{pkg.ServiceMemory},
		Apply: func(ctx *core.Context) error {
			if opts.DBPath == "" {
				return errors.New("plugin-memory: DBPath must not be empty")
			}

			var (
				store *memory.Store
				err   error
			)
			if opts.Embedder != nil {
				store, err = memory.NewStoreWithEmbedder(opts.DBPath, opts.Embedder)
			} else {
				store, err = memory.NewStore(opts.DBPath)
			}
			if err != nil {
				return err
			}

			core.Provide(ctx, pkg.ServiceMemory, store)

			return ctx.Effect(func() (core.Disposer, error) {
				return store.Close, nil
			})
		},
	}
}

// Manifest returns the declarative metadata for plugin-memory. It mirrors
// sup.plugin.json.
func Manifest() core.Manifest {
	return core.Manifest{
		Name:        pluginName,
		Version:     "2.0.0",
		Description: "Long-term memory backed by the v1 SQLite store.",
		Inject:      []string{},
		Provides:    []string{"memory"},
		Entry:       "builtin:memory",
	}
}
