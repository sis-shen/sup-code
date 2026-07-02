package worktree

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/supcode/supcode/pkg"
)

// MockWorktreeManager is a testify mock for pkg.WorktreeManager.
type MockWorktreeManager struct {
	mock.Mock
}

func (m *MockWorktreeManager) Create(ctx context.Context, agentID string, baseBranch string) (string, error) {
	args := m.Called(ctx, agentID, baseBranch)
	return args.String(0), args.Error(1)
}

func (m *MockWorktreeManager) Merge(ctx context.Context, agentID string) error {
	args := m.Called(ctx, agentID)
	return args.Error(0)
}

func (m *MockWorktreeManager) Abandon(ctx context.Context, agentID string) error {
	args := m.Called(ctx, agentID)
	return args.Error(0)
}

func (m *MockWorktreeManager) ListActive(ctx context.Context) ([]pkg.WorktreeInfo, error) {
	args := m.Called(ctx)
	return args.Get(0).([]pkg.WorktreeInfo), args.Error(1)
}

func (m *MockWorktreeManager) Cleanup(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

var _ pkg.WorktreeManager = (*MockWorktreeManager)(nil)