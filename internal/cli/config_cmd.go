package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long: `Manage SupCode configuration settings.

Configuration is stored in ~/.supcode/config.yaml.
You can view, set, and list configuration values.`,
	}

	cmd.AddCommand(newConfigGetCmd(), newConfigSetCmd(), newConfigListCmd())
	return cmd
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if app == nil || app.Config == nil {
				return fmt.Errorf("config not available")
			}
			key := args[0]
			val := app.Config.Get(key)
			if val == nil {
				return fmt.Errorf("config key not found: %s", key)
			}
			fmt.Fprintln(cmd.OutOrStdout(), val)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if app == nil || app.Config == nil {
				return fmt.Errorf("config not available")
			}
			key := args[0]
			value := args[1]

			if err := app.Config.Set(key, value); err != nil {
				return fmt.Errorf("set config: %w", err)
			}
			if err := app.Config.Save(); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Set %s = %s\n", key, value)
			return nil
		},
	}
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configuration values",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if app == nil || app.Config == nil {
				return fmt.Errorf("config not available")
			}
			settings := app.Config.AllSettings()
			keys := make([]string, 0, len(settings))
			for k := range settings {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				fmt.Fprintf(cmd.OutOrStdout(), "%s = %v\n", k, settings[k])
			}
			return nil
		},
	}
}
