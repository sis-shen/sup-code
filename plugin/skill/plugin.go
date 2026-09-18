// Package skill migrates the v1 skill loader (internal/skill) onto the Cordis
// kernel as the plugin-skill leaf plugin. It only assembles the v1
// implementation: the loader is provided under pkg.ServiceSkills and the v1
// skill tool is registered on the injected tool registry so that skill
// discovery and loading flow through the standard tools event pipeline.
package skill

import (
	"github.com/supcode/supcode/core"
	skills "github.com/supcode/supcode/internal/skill"
	skilltool "github.com/supcode/supcode/internal/tools/skilltool"
	"github.com/supcode/supcode/pkg"
)

// pluginName is the stable identifier of the skill plugin.
const pluginName = "plugin-skill"

// Options configures the skill plugin's discovery directories. They mirror the
// v1 loader's three priority levels: ProjectDir overrides UserDir, which
// overrides BuiltinDir.
type Options struct {
	BuiltinDir string
	UserDir    string
	ProjectDir string
}

// Plugin returns the loadable skill plugin. It injects the tool registry,
// provides the v1 skill loader under pkg.ServiceSkills, and registers the v1
// skill tool onto the registry so it participates in the tools events.
func Plugin(opts Options) core.Plugin {
	return core.Plugin{
		Name:     pluginName,
		Inject:   []string{pkg.ServiceTools},
		Provides: []string{pkg.ServiceSkills},
		Apply: func(ctx *core.Context) error {
			loader := skills.NewSkillLoader(opts.BuiltinDir, opts.UserDir, opts.ProjectDir)
			core.Provide(ctx, pkg.ServiceSkills, loader)

			reg := core.Use[pkg.ToolRegistry](ctx, pkg.ServiceTools)
			if err := reg.Register(skilltool.New(loader)); err != nil {
				return err
			}
			return nil
		},
	}
}

// Manifest returns the declarative metadata for plugin-skill. It mirrors
// sup.plugin.json.
func Manifest() core.Manifest {
	return core.Manifest{
		Name:        pluginName,
		Version:     "2.0.0",
		Description: "Skill discovery and loading exposed as the skills service and the skill tool.",
		Provides:    []string{pkg.ServiceSkills},
		Inject:      []string{pkg.ServiceTools},
		Entry:       "builtin:skill",
	}
}
