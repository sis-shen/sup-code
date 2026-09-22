// Package session migrates the v1 session manager (internal/tui/session.go)
// onto the Cordis kernel as the plugin-session leaf plugin. It only assembles
// the v1 implementation and exposes it as pkg.ServiceSessions; session
// persistence logic is reused unchanged.
package session

import (
	"errors"

	"github.com/supcode/supcode/core"
	tui "github.com/supcode/supcode/internal/tui"
	"github.com/supcode/supcode/pkg"
)

// pluginName is the stable identifier of the session plugin.
const pluginName = "plugin-session"

// pluginVersion is the plugin manifest version line.
const pluginVersion = "2.0.0"

// Options configures the session plugin. DBPath is the JSON session-store
// location and is required.
type Options struct {
	DBPath string
}

// Plugin returns the plugin-session leaf plugin. It opens the v1 session
// manager over DBPath, provides it under pkg.ServiceSessions as a
// pkg.SessionManager and registers CloseAll as a scope effect so unloading the
// plugin flushes persisted sessions and releases the in-memory cache.
func Plugin(opts Options) core.Plugin {
	return core.Plugin{
		Name:     pluginName,
		Inject:   []string{},
		Provides: []string{pkg.ServiceSessions},
		Apply: func(ctx *core.Context) error {
			if opts.DBPath == "" {
				return errors.New("plugin-session: DBPath must not be empty")
			}

			sm, err := tui.NewSessionManager(opts.DBPath)
			if err != nil {
				return err
			}

			core.Provide[pkg.SessionManager](ctx, pkg.ServiceSessions, sm)

			return ctx.Effect(func() (core.Disposer, error) {
				return sm.CloseAll, nil
			})
		},
	}
}

// Manifest returns the declarative metadata for plugin-session. It mirrors
// sup.plugin.json.
func Manifest() core.Manifest {
	return core.Manifest{
		Name:        pluginName,
		Version:     pluginVersion,
		Description: "Multi-turn session manager reusing the v1 tui session store.",
		Inject:      []string{},
		Provides:    []string{"sessions"},
		Entry:       "builtin:session",
	}
}
