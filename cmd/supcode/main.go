package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/supcode/supcode/internal"
	"github.com/supcode/supcode/internal/cli"
	"github.com/supcode/supcode/internal/config"
	"github.com/supcode/supcode/pkg"
)

func main() {
	// ── Parse --debug and --config before cobra ──────────
	var debug bool
	var configPath string
	cleaned := make([]string, 0, len(os.Args))
	cleaned = append(cleaned, os.Args[0])
	skipNext := false
	for i, arg := range os.Args[1:] {
		if skipNext {
			skipNext = false
			continue
		}
		if arg == "--debug" {
			debug = true
			continue
		}
		if arg == "--config" && i+1 < len(os.Args)-1 {
			configPath = os.Args[i+2]
			skipNext = true
			continue
		}
		if strings.HasPrefix(arg, "--config=") {
			configPath = strings.TrimPrefix(arg, "--config=")
			continue
		}
		cleaned = append(cleaned, arg)
	}
	os.Args = cleaned

	// ── Check for --help/-h/--version before requiring config ──
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" || arg == "--version" || arg == "-v" {
			rootCmd := cli.RootCmd()
			rootCmd.SetArgs(os.Args[1:])
			if err := rootCmd.Execute(); err != nil {
				os.Exit(1)
			}
			return
		}
	}

	// ── Configure logging ────────────────────────────────
	logLevel := slog.LevelInfo
	if debug {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})))

	// ── Signal handling for graceful shutdown ────────────
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// ── Determine run mode BEFORE wiring ─────────────────
	isSubcommand := false
	subcommands := map[string]bool{"config": true, "skill": true, "completion": true, "help": true}
	var queryParts []string
	for _, a := range os.Args[1:] {
		if !strings.HasPrefix(a, "-") {
			if subcommands[a] {
				isSubcommand = true
			} else {
				queryParts = append(queryParts, a)
			}
			break
		}
	}

	// ── Mode A: Subcommand (lightweight, no agent needed) ──
	if isSubcommand {
		supcode, err := internal.NewSupCode(configPath)
		if err != nil {
			slog.Warn("partial init for subcommand", "error", err)
		}
		if supcode != nil {
			defer supcode.Close()
			cli.SetApp(supcode)
		}
		rootCmd := cli.RootCmd()
		rootCmd.SetArgs(os.Args[1:])
		if err := rootCmd.ExecuteContext(ctx); err != nil {
			slog.Error("command failed", "error", err)
			os.Exit(1)
		}
		return
	}

	// ── Load config (shared by Mode B and C) ─────────────
	cm := config.NewManager()
	if configPath != "" {
		cm.Viper().SetConfigFile(configPath)
	}
	if err := cm.Load(); err != nil {
		fmt.Fprintln(os.Stderr, "Error: LLM API key not configured.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  Set the SUPCODE_LLM_API_KEY environment variable or")
		fmt.Fprintln(os.Stderr, "  create ~/.supcode/config.yaml with:")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "    llm:")
		fmt.Fprintln(os.Stderr, "      api_key: \"your-api-key-here\"")
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}

	// ── Mode B: Single-shot query ────────────────────────
	if len(queryParts) > 0 {
		runSingleShot(ctx, cm, strings.Join(queryParts, " "))
		return
	}

	// ── Mode C: TUI interactive session ──────────────────
	agent, err := internal.BuildAgent(cm)
	if err != nil {
		slog.Error("failed to build agent for TUI", "error", err)
		os.Exit(1)
	}
	if closer, ok := agent.(io.Closer); ok {
		defer func() { _ = closer.Close() }()
	}
	if err := internal.InitTUI(ctx, agent); err != nil {
		slog.Error("TUI exited with error", "error", err)
		os.Exit(1)
	}
}

func runSingleShot(ctx context.Context, cfg pkg.Config, query string) {
	logger := slog.With("mode", "single_shot")

	agent, err := internal.BuildAgent(cfg)
	if err != nil {
		slog.Error("build agent for single shot", "error", err)
		os.Exit(1)
	}
	if closer, ok := agent.(io.Closer); ok {
		defer func() { _ = closer.Close() }()
	}

	sessionID := "single-shot-" + strings.ReplaceAll(query[0:min(len(query), 20)], " ", "_")

	logger.Info("executing query", "session_id", sessionID, "query", query)

	result, err := agent.Run(ctx, sessionID, query)
	if err != nil {
		if result != nil && result.Summary != "" {
			fmt.Println(result.Summary)
		}
		if result != nil && result.Error != "" {
			fmt.Fprintln(os.Stderr, result.Error)
		}
		slog.Error("execution failed", "error", err)
		os.Exit(1)
	}

	if result.Summary != "" {
		fmt.Println(result.Summary)
	}
	if result.Error != "" {
		fmt.Fprintln(os.Stderr, result.Error)
		os.Exit(1)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
