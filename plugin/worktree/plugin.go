// Package worktree migrates the v1 Git worktree manager (internal/worktree)
// into a Sup Harness 2.0 leaf plugin. It only assembles the v1 implementation
// and exposes it under pkg.ServiceWorktree; business logic is reused unchanged.
package worktree

import (
	"fmt"

	"github.com/supcode/supcode/core"
	worktree "github.com/supcode/supcode/internal/worktree"
	"github.com/supcode/supcode/pkg"
)

// pluginName is the stable identifier of the worktree plugin.
const pluginName = "plugin-worktree"

// Options configures the worktree plugin.
type Options struct {
	// RepoPath is the git repository the manager operates on. It is required.
	RepoPath string
	// BaseDir is where worktrees are created. When empty, the v1 default under
	// the system temp directory is used.
	BaseDir string
}

// Plugin returns the plugin-worktree leaf plugin. It constructs the v1 worktree
// manager for RepoPath and provides it under pkg.ServiceWorktree.
func Plugin(opts Options) core.Plugin {
	return core.Plugin{
		Name:     pluginName,
		Provides: []string{pkg.ServiceWorktree},
		Apply: func(ctx *core.Context) error {
			if opts.RepoPath == "" {
				return fmt.Errorf("%s: RepoPath must not be empty", pluginName)
			}

			var (
				mgr pkg.WorktreeManager
				err error
			)
			if opts.BaseDir != "" {
				mgr, err = worktree.NewManagerWithBase(opts.RepoPath, opts.BaseDir)
			} else {
				mgr, err = worktree.NewManager(opts.RepoPath)
			}
			if err != nil {
				return err
			}

			core.Provide(ctx, pkg.ServiceWorktree, mgr)
			return nil
		},
	}
}

// Manifest returns the declarative metadata for plugin-worktree. It mirrors
// sup.plugin.json.
func Manifest() core.Manifest {
	return core.Manifest{
		Name:     pluginName,
		Provides: []string{pkg.ServiceWorktree},
		Entry:    "builtin:worktree",
	}
}
