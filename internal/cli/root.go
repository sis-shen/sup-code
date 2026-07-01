package cli

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/supcode/supcode/internal"
	"github.com/supcode/supcode/internal/tui"
)

var (
	version = "0.1.0"
)

func RootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "supcode [query...]",
		Version:       version,
		Short:         "SupCode - Terminal-native AI coding agent",
		Long:          "SupCode is a terminal-native AI coding agent with multi-agent collaboration.",
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if app == nil {
				return fmt.Errorf("application not initialized — use 'supcode' from the main binary")
			}
			if len(args) > 0 {
				return runSingleShot(cmd, strings.Join(args, " "))
			}
			return runTUI(cmd)
		},
	}

	rootCmd.AddCommand(newConfigCmd())
	return rootCmd
}

func runTUI(cmd *cobra.Command) error {
	ctx := cmd.Context()
	logger := slog.With("mode", "tui")

	renderer, err := tui.NewRenderer()
	if err != nil {
		return fmt.Errorf("init renderer: %w", err)
	}
	defer renderer.Close()

	modelName := app.LLMClient.ProviderName()
	cwd, _ := os.Getwd()
	logger.Info("starting supcode",
		"version", version,
		"model", modelName,
		"cwd", cwd,
	)

	session, err := app.SessionMgr.Create(ctx, "Interactive Session")
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return tui.RunTUI(app.Service, renderer, session.ID)
}

func runSingleShot(cmd *cobra.Command, input string) error {
	ctx := cmd.Context()
	logger := slog.With("mode", "single_shot")

	logger.Info("executing single-shot mode", "input", input)

	session, err := app.SessionMgr.Create(ctx, "Single Shot")
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	result, err := app.Agent.Run(ctx, session.ID, input)
	if err != nil {
		return fmt.Errorf("execute: %w", err)
	}

	if result.Summary != "" {
		fmt.Fprintln(cmd.OutOrStdout(), result.Summary)
	}
	if result.Error != "" {
		return fmt.Errorf("%s", result.Error)
	}

	if err := app.SessionMgr.Close(ctx, session.ID); err != nil {
		logger.Warn("failed to close session", "error", err)
	}

	return nil
}

var app *internal.SupCode

func SetApp(s *internal.SupCode) {
	app = s
}

func App() *internal.SupCode {
	return app
}

func Version() string { return version }
