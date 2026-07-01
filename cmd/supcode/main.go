package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/supcode/supcode/internal"
	"github.com/supcode/supcode/internal/cli"
)

func main() {
	// ── Parse --debug and --config before cobra ───────────────
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

	// ── Configure logging ─────────────────────────────────────
	logLevel := slog.LevelInfo
	if debug {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})))

	// ── Signal handling for graceful shutdown ─────────────────
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// ── Wire all layers ───────────────────────────────────────
	supcode, err := internal.NewSupCode(configPath)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "missing required config") || strings.Contains(errMsg, "llm.api_key") {
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
		slog.Error("failed to initialize", "error", err)
		os.Exit(1)
	}
	defer supcode.Close()

	// ── Wire CLI with application components ──────────────────
	cli.SetApp(supcode)

	// ── Determine run mode ────────────────────────────────────
	hasPositionalArgs := false
	for _, a := range os.Args[1:] {
		if !strings.HasPrefix(a, "-") {
			hasPositionalArgs = true
			break
		}
	}

	if hasPositionalArgs {
		var queryParts []string
		for _, a := range os.Args[1:] {
			if !strings.HasPrefix(a, "-") {
				queryParts = append(queryParts, a)
			}
		}
		query := strings.Join(queryParts, " ")
		runSingleShot(ctx, supcode, query)
	} else {
		rootCmd := cli.RootCmd()
		if err := rootCmd.ExecuteContext(ctx); err != nil {
			slog.Error("command failed", "error", err)
			os.Exit(1)
		}
	}
}

func runSingleShot(ctx context.Context, supcode *internal.SupCode, query string) {
	logger := slog.With("mode", "single_shot")

	session, err := supcode.SessionMgr.Create(ctx, "Single Shot")
	if err != nil {
		slog.Error("create session", "error", err)
		os.Exit(1)
	}

	logger.Info("executing query", "session_id", session.ID, "query", query)

	result, err := supcode.Agent.Run(ctx, session.ID, query)
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

	if err := supcode.SessionMgr.Close(ctx, session.ID); err != nil {
		logger.Warn("failed to close session", "error", err)
	}
}
