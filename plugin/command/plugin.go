// Package command migrates the v1 slash-command registry (internal/tui
// completions) onto the Cordis kernel as the plugin-command leaf plugin. It
// provides pkg.ServiceCommands and pre-registers the deterministic builtin
// commands (help, version). Commands are dispatched locally by the interaction
// layer and never call the model.
package command

import (
	"github.com/supcode/supcode/core"
	"github.com/supcode/supcode/pkg"
)

const (
	pluginName    = "plugin-command"
	pluginVersion = "2.0.0"
)

// Options configures the command plugin.
type Options struct {
	// Version is returned by the builtin /version command. When empty it
	// defaults to the plugin version.
	Version string
}

// Plugin returns the command registry plugin. It provides pkg.ServiceCommands
// and registers the builtin help and version commands.
func Plugin(opts Options) core.Plugin {
	return core.Plugin{
		Name:     pluginName,
		Provides: []string{pkg.ServiceCommands},
		Apply: func(ctx *core.Context) error {
			reg := newRegistry()

			version := opts.Version
			if version == "" {
				version = pluginVersion
			}

			for _, cmd := range []pkg.SlashCommand{
				&helpCommand{registry: reg},
				&versionCommand{version: version},
			} {
				if err := reg.Register(cmd); err != nil {
					return err
				}
			}

			core.Provide[pkg.CommandRegistry](ctx, pkg.ServiceCommands, reg)
			return nil
		},
	}
}

// Manifest describes plugin-command and mirrors sup.plugin.json.
func Manifest() core.Manifest {
	return core.Manifest{
		Name:        pluginName,
		Version:     pluginVersion,
		Description: "Slash command registry with builtin help and version commands.",
		Provides:    []string{pkg.ServiceCommands},
		Entry:       "plugin/command",
	}
}
