// Package permission migrates the v1 permission engine into a Sup Harness 2.0
// leaf plugin. It provides pkg.ServicePermission and listens on the tool
// pre-execute waterfall to short-circuit denied invocations.
package permission

import (
	"context"
	"fmt"

	"github.com/supcode/supcode/core"
	perm "github.com/supcode/supcode/internal/permission"
	"github.com/supcode/supcode/pkg"
)

// Options configures the permission plugin. Engine takes precedence; when nil,
// v1's engine is built with Rules (or its built-in defaults when Rules is empty).
type Options struct {
	Engine pkg.PermissionEngine
	Rules  []pkg.PermissionRule
}

// Plugin returns the loadable permission plugin.
func Plugin(opts Options) core.Plugin {
	return core.Plugin{
		Name:     "plugin-permission",
		Provides: []string{pkg.ServicePermission},
		Apply: func(ctx *core.Context) error {
			eng := opts.Engine
			if eng == nil {
				if len(opts.Rules) > 0 {
					eng = perm.NewWithRules(opts.Rules)
				} else {
					eng = perm.New()
				}
			}
			core.Provide(ctx, pkg.ServicePermission, eng)

			ctx.On(pkg.EventToolsPreExecute, func(c context.Context, payload any, _ core.Next) (any, error) {
				inv, ok := payload.(*pkg.ToolInvocation)
				if !ok {
					return nil, fmt.Errorf("plugin-permission: expected *pkg.ToolInvocation, got %T", payload)
				}

				action := pkg.Action{
					Type:     "tool",
					Target:   inv.Name,
					ToolName: inv.Name,
					Params:   string(inv.Params),
				}

				d, err := eng.Check(c, action)
				if err != nil {
					return nil, err
				}
				inv.Decision = d

				if d == pkg.DecisionDeny {
					_ = eng.LogAction(c, action, d, "denied by permission plugin")
					return nil, &pkg.ErrPermissionDenied{Action: action, Rule: "deny"}
				}

				return inv, nil
			})

			return nil
		},
	}
}

// Manifest returns the plugin metadata, mirroring sup.plugin.json.
func Manifest() core.Manifest {
	return core.Manifest{
		Name:        "plugin-permission",
		Version:     "2.0.0",
		Description: "Tool pre-execute permission enforcement backed by the v1 permission engine.",
		Provides:    []string{"permission"},
		Events:      []core.ManifestEvent{{Name: pkg.EventToolsPreExecute, Mode: "waterfall"}},
		Entry:       "builtin:permission",
	}
}
