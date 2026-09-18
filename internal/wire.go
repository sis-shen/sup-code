package internal

import (
	"context"
	"os"
	"path/filepath"

	"github.com/supcode/supcode/internal/tui"
	"github.com/supcode/supcode/pkg"
)

// InitTUI 初始化 TUI 并注入 Agent。
// agent 可为 nil（TUI 可启动，输入回车会显示"no agent configured"而非 panic）。
func InitTUI(ctx context.Context, agent pkg.Agent) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	dbDir := filepath.Join(homeDir, ".supcode")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return err
	}
	dbPath := filepath.Join(dbDir, "sessions.db")

	sm, err := tui.NewSessionManager(dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = sm.CloseAll() }()

	session, err := sm.Create(ctx, "Interactive Session")
	if err != nil {
		return err
	}

	renderer, err := tui.NewRenderer()
	if err != nil {
		return err
	}
	defer func() { _ = renderer.Close() }()

	service := tui.NewService(sm)
	if agent != nil {
		service.SetAgent(agent)
	}

	return tui.RunTUI(service, renderer, session.ID)
}
