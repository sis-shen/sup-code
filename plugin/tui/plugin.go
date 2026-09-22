// Package tui exposes the v1 Bubble Tea interaction layer (internal/tui) as the
// plugin-tui leaf plugin. It provides the replaceable interaction seam
// (pkg.ServiceInteraction) so the host can swap in a headless plugin-cli
// implementation for tests or single-shot queries.
//
// The plugin never starts a Bubble Tea program at Apply time. Program startup
// belongs to the host (cmd/sup, Phase 5); building Service here only wires the
// engine-to-interaction channels so later phases can connect directly.
package tui

import (
	"github.com/supcode/supcode/core"
	tui "github.com/supcode/supcode/internal/tui"
	"github.com/supcode/supcode/pkg"
)

const (
	// pluginName is the stable identifier of the interaction plugin.
	pluginName = "plugin-tui"
	// pluginVersion is the plugin manifest version.
	pluginVersion = "2.0.0"

	// agentServiceKey is the Phase 4 agent seam. It is read opportunistically
	// via MaybeUse and deliberately omitted from Inject so plugin-tui stays
	// loadable before plugin-agent exists.
	agentServiceKey = "agent"
)

// Options configures the TUI plugin. It is intentionally empty for now; it
// exists so the constructor shape can grow without breaking callers.
type Options struct{}

// Plugin returns the loadable TUI plugin. It injects the session manager,
// builds the v1 interaction service and provides it under
// pkg.ServiceInteraction. The Bubble Tea program is not started here.
func Plugin(_ Options) core.Plugin {
	return core.Plugin{
		Name:     pluginName,
		Inject:   []string{pkg.ServiceSessions},
		Provides: []string{pkg.ServiceInteraction},
		Apply: func(ctx *core.Context) error {
			sm := core.Use[pkg.SessionManager](ctx, pkg.ServiceSessions)

			svc := tui.NewService(concreteSessionManager(sm))

			if a, ok := core.MaybeUse[pkg.Agent](ctx, agentServiceKey); ok && a != nil {
				svc.SetAgent(a)
			}

			core.Provide(ctx, pkg.ServiceInteraction, svc)
			return nil
		},
	}
}

// concreteSessionManager adapts the injected pkg.SessionManager to the concrete
// *tui.SessionManager that v1's NewService requires. The sessions seam is typed
// as the pkg.SessionManager interface, but the v1 constructor takes its own
// concrete type, so only the v1 implementation can be recovered. Any other
// provider yields nil: none of the pkg.InteractionService methods dereference
// the session manager, so the interaction service remains fully functional.
func concreteSessionManager(sm pkg.SessionManager) *tui.SessionManager {
	c, _ := sm.(*tui.SessionManager)
	return c
}

// Manifest returns the declarative metadata for plugin-tui. It mirrors
// sup.plugin.json.
func Manifest() core.Manifest {
	return core.Manifest{
		Name:        pluginName,
		Version:     pluginVersion,
		Description: "Bubble Tea interaction layer exposed as the replaceable interaction service.",
		Inject:      []string{pkg.ServiceSessions},
		Provides:    []string{pkg.ServiceInteraction},
		Entry:       "builtin:tui",
	}
}
