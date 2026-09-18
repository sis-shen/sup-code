// Package tools migrates the v1 builtin tool registry (internal/tools) onto
// the Cordis kernel as the plugin-tools leaf plugin. It only assembles the v1
// implementation and bridges Execute onto the tool pipeline events; business
// logic is reused unchanged.
package tools

import (
	"github.com/supcode/supcode/core"
	toolreg "github.com/supcode/supcode/internal/tools"
	"github.com/supcode/supcode/internal/tools/bash"
	"github.com/supcode/supcode/internal/tools/editfile"
	"github.com/supcode/supcode/internal/tools/glob"
	"github.com/supcode/supcode/internal/tools/grep"
	"github.com/supcode/supcode/internal/tools/readfile"
	"github.com/supcode/supcode/internal/tools/writefile"
	"github.com/supcode/supcode/pkg"
)

// pluginName is the stable identifier of the tools plugin.
const pluginName = "plugin-tools"

// Options configures the tools plugin. It is intentionally empty for now; it
// exists so the constructor shape can grow without breaking callers.
type Options struct{}

// Plugin returns the plugin-tools leaf plugin. It injects the permission
// engine, builds the v1 registry with it, registers the six builtin tools and
// provides the resulting decorated registry under pkg.ServiceTools.
func Plugin(_ Options) core.Plugin {
	return core.Plugin{
		Name:     pluginName,
		Inject:   []string{pkg.ServicePermission},
		Provides: []string{pkg.ServiceTools},
		Apply: func(ctx *core.Context) error {
			perm := core.Use[pkg.PermissionEngine](ctx, pkg.ServicePermission)

			inner := toolreg.NewRegistry(perm)
			builtins := []pkg.Tool{
				&bash.Tool{},
				&readfile.Tool{},
				&writefile.Tool{},
				&editfile.Tool{},
				&glob.Tool{},
				&grep.Tool{},
			}
			for _, tool := range builtins {
				if err := inner.Register(tool); err != nil {
					return err
				}
			}

			core.Provide(ctx, pkg.ServiceTools, &eventRegistry{kernel: ctx.Kernel(), inner: inner})
			return nil
		},
	}
}

// Manifest returns the declarative metadata for plugin-tools. It mirrors
// sup.plugin.json.
func Manifest() core.Manifest {
	return core.Manifest{
		Name:     pluginName,
		Provides: []string{pkg.ServiceTools},
		Inject:   []string{pkg.ServicePermission},
		Events: []core.ManifestEvent{
			{Name: pkg.EventToolsPreExecute, Mode: "waterfall"},
			{Name: pkg.EventToolsExecute, Mode: "waterfall"},
			{Name: pkg.EventToolsPostExecute, Mode: "waterfall"},
			{Name: pkg.EventToolsResult, Mode: "emit"},
		},
		Entry: "builtin:tools",
	}
}
