// Package context migrates the v1 short-term context manager
// (internal/contextmgr) onto the Cordis kernel as the plugin-context leaf
// plugin. It only assembles the v1 implementation and exposes it as
// pkg.ServiceContext; business logic is reused unchanged.
package context

import (
	"github.com/supcode/supcode/core"
	"github.com/supcode/supcode/internal/contextmgr"
	"github.com/supcode/supcode/pkg"
)

const (
	pluginName    = "plugin-context"
	pluginVersion = "2.0.0"
)

// Options configures the context plugin. LLM, when non-nil, enables LLM-based
// context compression; otherwise the plugin opportunistically resolves
// pkg.ServiceLLM from the scope. Threshold sets the compression trigger for
// the non-LLM path and is ignored when an LLM is available (the compression
// manager owns its own threshold).
type Options struct {
	Threshold int
	LLM       pkg.LLMClient
}

// Plugin returns the plugin-context leaf plugin. It builds the v1 contextmgr
// manager, preferring LLM compression when a client is supplied (options win,
// otherwise pkg.ServiceLLM is used), and provides it under pkg.ServiceContext.
func Plugin(opts Options) core.Plugin {
	return core.Plugin{
		Name:     pluginName,
		Inject:   []string{},
		Provides: []string{pkg.ServiceContext},
		Apply: func(ctx *core.Context) error {
			llm := opts.LLM
			if llm == nil {
				if v, ok := core.MaybeUse[pkg.LLMClient](ctx, pkg.ServiceLLM); ok {
					llm = v
				}
			}

			var mgr *contextmgr.Manager
			switch {
			case llm != nil:
				mgr = contextmgr.NewManagerWithCompression(llm)
			case opts.Threshold > 0:
				mgr = contextmgr.NewManagerWithThreshold(opts.Threshold)
			default:
				mgr = contextmgr.NewManager()
			}

			core.Provide[pkg.ContextManager](ctx, pkg.ServiceContext, mgr)
			return nil
		},
	}
}

// Manifest returns the declarative metadata for plugin-context. It mirrors
// sup.plugin.json.
func Manifest() core.Manifest {
	return core.Manifest{
		Name:        pluginName,
		Version:     pluginVersion,
		Description: "Short-term context manager exposing pkg.ContextManager, reusing the v1 implementation.",
		Inject:      []string{},
		Provides:    []string{pkg.ServiceContext},
		Entry:       "builtin:context",
	}
}
