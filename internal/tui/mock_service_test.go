package tui

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/supcode/supcode/pkg"
)

// MockInteractionService 是 pkg.InteractionService 的 testify mock 实现
// 供 engine 等其它 Agent 在测试中使用
type MockInteractionService struct {
	mock.Mock
}

func (m *MockInteractionService) StreamResponse(ctx context.Context, sessionID string, stream <-chan pkg.StreamEvent) error {
	args := m.Called(ctx, sessionID, stream)
	return args.Error(0)
}

func (m *MockInteractionService) RequestConfirmation(ctx context.Context, sessionID string, prompt pkg.ConfirmPrompt) (bool, error) {
	args := m.Called(ctx, sessionID, prompt)
	return args.Bool(0), args.Error(1)
}

func (m *MockInteractionService) ReadInput(ctx context.Context, sessionID string) (string, error) {
	args := m.Called(ctx, sessionID)
	return args.String(0), args.Error(1)
}

func (m *MockInteractionService) Notify(ctx context.Context, sessionID string, level pkg.NotifyLevel, message string) {
	m.Called(ctx, sessionID, level, message)
}

func (m *MockInteractionService) HandleCommand(ctx context.Context, sessionID string, input string) (pkg.CommandResult, error) {
	args := m.Called(ctx, sessionID, input)
	return args.Get(0).(pkg.CommandResult), args.Error(1)
}

// ─── compile-time interface check ──────────────────────────────
var _ pkg.InteractionService = (*MockInteractionService)(nil)
