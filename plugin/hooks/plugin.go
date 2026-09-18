// Package hooks migrates the v1 builtin post-tool hooks into Cordis leaf
// plugins. AuditPlugin bridges builtin.AuditHook onto the tools/result event;
// GitPlugin bridges builtin.GitCommitHook onto the tools/post-execute event.
// The v1 hook logic is reused verbatim; this package only assembles it.
package hooks

import (
	"context"

	"github.com/supcode/supcode/core"
	"github.com/supcode/supcode/internal/hooks/builtin"
	"github.com/supcode/supcode/pkg"
)

// AuditOptions configures the audit plugin. The v1 audit hook has no tunable
// parameters, so the struct is intentionally empty and reserved for future
// configuration (Phase 3 config seam).
type AuditOptions struct{}

// AuditPlugin returns the audit hook plugin. It injects the permission service
// and listens for tools/result (emit), delegating to builtin.AuditHook.
func AuditPlugin(_ AuditOptions) core.Plugin {
	return core.Plugin{
		Name:   "plugin-hooks-audit",
		Inject: []string{pkg.ServicePermission},
		Apply: func(ctx *core.Context) error {
			eng := core.Use[pkg.PermissionEngine](ctx, pkg.ServicePermission)
			h := builtin.NewAuditHook(eng)

			ctx.On(pkg.EventToolsResult, func(c context.Context, payload any, _ core.Next) (any, error) {
				inv, ok := payload.(*pkg.ToolInvocation)
				if !ok {
					return nil, nil
				}
				if err := h.AfterTool(c, inv.Name, inv.Params, inv.Result); err != nil {
					return nil, err
				}
				return nil, nil
			})

			return nil
		},
	}
}

// GitPlugin returns the git auto-commit hook plugin. It has no dependencies and
// listens for tools/post-execute (waterfall), delegating to
// builtin.GitCommitHook. A disabled hook is a no-op.
func GitPlugin(enabled bool) core.Plugin {
	return core.Plugin{
		Name: "plugin-hooks-git",
		Apply: func(ctx *core.Context) error {
			h := builtin.NewGitCommitHook(enabled)

			ctx.On(pkg.EventToolsPostExecute, func(c context.Context, payload any, _ core.Next) (any, error) {
				inv, ok := payload.(*pkg.ToolInvocation)
				if !ok {
					return payload, nil
				}
				if err := h.AfterTool(c, inv.Name, inv.Params, inv.Result); err != nil {
					return nil, err
				}
				return inv, nil
			})

			return nil
		},
	}
}

// Manifest returns the audit plugin's metadata. The git plugin is described by
// GitManifest. Events are not representable in core.Manifest and live in the
// sup.plugin.json companion files.
func Manifest() core.Manifest {
	return core.Manifest{
		Name:        "plugin-hooks-audit",
		Version:     "2.0.0",
		Description: "Audit hook plugin: records tool invocations through the permission engine.",
		Inject:      []string{pkg.ServicePermission},
		Events:      []core.ManifestEvent{{Name: pkg.EventToolsResult, Mode: "emit"}},
		Entry:       "plugin/hooks",
	}
}

// GitManifest returns the git hook plugin's metadata.
func GitManifest() core.Manifest {
	return core.Manifest{
		Name:        "plugin-hooks-git",
		Version:     "2.0.0",
		Description: "Git auto-commit hook plugin: commits file changes after write/edit tools.",
		Events:      []core.ManifestEvent{{Name: pkg.EventToolsPostExecute, Mode: "waterfall"}},
		Entry:       "plugin/hooks",
	}
}
