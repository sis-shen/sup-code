// Package config migrates the v1 configuration manager (internal/config) onto
// the Cordis kernel as the plugin-config leaf plugin. It only assembles the v1
// Viper-backed manager: the manager is loaded from the resolved config sources
// and provided under pkg.ServiceConfig. API key validation stays deferred to
// the agent layer, so configuration loads offline without credentials.
package config

import (
	"fmt"

	"github.com/supcode/supcode/core"
	config "github.com/supcode/supcode/internal/config"
	"github.com/supcode/supcode/pkg"
)

// pluginName is the stable identifier of the config plugin.
const pluginName = "plugin-config"

// Options configures the config plugin. Path, when set, names an explicit
// config file that overrides discovery of ~/.supcode/config.yaml and the
// project-level .supcode/config.yaml.
type Options struct {
	Path string
}

// Plugin returns the loadable plugin-config leaf plugin. It constructs the v1
// Viper manager, optionally points it at an explicit file, loads the merged
// configuration, and provides it under pkg.ServiceConfig.
func Plugin(opts Options) core.Plugin {
	return core.Plugin{
		Name:     pluginName,
		Provides: []string{pkg.ServiceConfig},
		Apply: func(ctx *core.Context) error {
			cm := config.NewManager()
			if err := cm.Load(); err != nil {
				return err
			}

			// v1 Load calls viper.SetConfigName, which clears any previously
			// set config file, so the explicit path is applied afterwards.
			// ReadInConfig then replaces the discovered file, matching
			// SetConfigFile semantics.
			if opts.Path != "" {
				cm.Viper().SetConfigFile(opts.Path)
				if err := cm.Viper().ReadInConfig(); err != nil {
					return fmt.Errorf("%s: read %q: %w", pluginName, opts.Path, err)
				}
			}

			core.Provide(ctx, pkg.ServiceConfig, cm)
			return nil
		},
	}
}

// Manifest returns the declarative metadata for plugin-config. It mirrors
// sup.plugin.json.
func Manifest() core.Manifest {
	return core.Manifest{
		Name:     pluginName,
		Provides: []string{pkg.ServiceConfig},
		Entry:    "builtin:config",
	}
}
