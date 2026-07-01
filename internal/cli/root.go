package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/supcode/supcode/internal/config"
	"github.com/supcode/supcode/internal/tui"
)

var (
	cfgFile string
	version = "0.1.0"
	cfg     *config.Manager
)

func RootCmd() *cobra.Command {
	cfg = config.NewManager()

	rootCmd := &cobra.Command{
		Use:     "supcode",
		Version: version,
		Short:   "SupCode - Terminal-native AI coding agent",
		Long: `SupCode is a terminal-native AI coding agent with multi-agent collaboration.
It supports natural language interaction, automatic planning and execution,
and multi-tool integration.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return cfg.Load()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(cmd)
		},
		SilenceUsage: true,
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.supcode/config.yaml)")
	rootCmd.PersistentFlags().String("model", cfg.GetString(config.ConfigKeyLLMModel), "LLM model name")
	rootCmd.PersistentFlags().String("api-key", "", "LLM API key")

	_ = viper.BindPFlag(config.ConfigKeyLLMModel, rootCmd.PersistentFlags().Lookup("model"))
	_ = viper.BindPFlag(config.ConfigKeyLLMAPIKey, rootCmd.PersistentFlags().Lookup("api-key"))
	_ = viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))

	rootCmd.AddCommand(configCmd())

	return rootCmd
}

func runTUI(cmd *cobra.Command) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	dbDir := filepath.Join(homeDir, ".supcode")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	dbPath := filepath.Join(dbDir, "sessions.db")

	sm, err := tui.NewSessionManager(dbPath)
	if err != nil {
		return fmt.Errorf("init session manager: %w", err)
	}
	defer sm.CloseAll()

	ctx := cmd.Context()
	session, err := sm.Create(ctx, "Interactive Session")
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	renderer, err := tui.NewRenderer()
	if err != nil {
		return fmt.Errorf("init renderer: %w", err)
	}
	defer renderer.Close()

	service := tui.NewService(sm)

	return tui.RunTUI(service, renderer, session.ID)
}

func Version() string { return version }
func ConfigManager() *config.Manager { return cfg }
