package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/supcode/supcode/internal/skill"
)

func skillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage skills",
		Long:  "Install, list, and manage SupCode skill packages.",
	}

	cmd.AddCommand(skillInstallCmd())
	cmd.AddCommand(skillListCmd())

	return cmd
}

func skillInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install <path>",
		Short: "Install a skill from local path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			source := args[0]
			if err := skill.InstallSkill(source); err != nil {
				return fmt.Errorf("install skill: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Skill installed from: %s\n", source)
			return nil
		},
	}
}

func skillListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return skill.ListAllSkills()
		},
	}
}
