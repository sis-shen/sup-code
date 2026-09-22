package cli

import (
	"io"
	"os"

	"github.com/supcode/supcode/core"
	"github.com/supcode/supcode/pkg"
)

const (
	pluginName    = "plugin-cli"
	pluginVersion = "2.0.0"
)

// Options configures the headless CLI interaction plugin. In and Out default to
// os.Stdin and os.Stdout when nil. AutoConfirm is returned by
// RequestConfirmation; set it to true for fully non-interactive runs. Only one
// interaction provider (plugin-cli or plugin-tui) may be loaded at a time.
type Options struct {
	In          io.Reader
	Out         io.Writer
	AutoConfirm bool
}

// Plugin returns the headless interaction plugin. It injects the session and
// command services and provides pkg.ServiceInteraction.
func Plugin(opts Options) core.Plugin {
	return core.Plugin{
		Name:     pluginName,
		Inject:   []string{pkg.ServiceSessions, pkg.ServiceCommands},
		Provides: []string{pkg.ServiceInteraction},
		Apply: func(ctx *core.Context) error {
			in := opts.In
			if in == nil {
				in = os.Stdin
			}
			out := opts.Out
			if out == nil {
				out = os.Stdout
			}

			commands := core.Use[pkg.CommandRegistry](ctx, pkg.ServiceCommands)
			core.Provide[pkg.InteractionService](ctx, pkg.ServiceInteraction,
				newHeadlessInteraction(in, out, opts.AutoConfirm, commands))
			return nil
		},
	}
}

// Manifest describes plugin-cli and mirrors sup.plugin.json.
func Manifest() core.Manifest {
	return core.Manifest{
		Name:        pluginName,
		Version:     pluginVersion,
		Description: "Headless interaction service reading stdin and writing stdout.",
		Inject:      []string{pkg.ServiceSessions, pkg.ServiceCommands},
		Provides:    []string{pkg.ServiceInteraction},
		Entry:       "plugin/cli",
	}
}
