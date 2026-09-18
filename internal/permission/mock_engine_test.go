package permission

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/supcode/supcode/pkg"
)

// MockPermissionEngine is a testify mock for pkg.PermissionEngine.
// Used by tools Agent for testing.
type MockPermissionEngine struct {
	mock.Mock
}

func (m *MockPermissionEngine) Check(ctx context.Context, action pkg.Action) (pkg.Decision, error) {
	args := m.Called(ctx, action)
	return args.Get(0).(pkg.Decision), args.Error(1)
}

func (m *MockPermissionEngine) AddRule(rule pkg.PermissionRule) error {
	args := m.Called(rule)
	return args.Error(0)
}

func (m *MockPermissionEngine) RemoveRule(ruleID string) error {
	args := m.Called(ruleID)
	return args.Error(0)
}

func (m *MockPermissionEngine) ListRules() []pkg.PermissionRule {
	args := m.Called()
	return args.Get(0).([]pkg.PermissionRule)
}

func (m *MockPermissionEngine) LogAction(ctx context.Context, action pkg.Action, decision pkg.Decision, result string) error {
	args := m.Called(ctx, action, decision, result)
	return args.Error(0)
}

// Compile-time interface check
var _ pkg.PermissionEngine = (*MockPermissionEngine)(nil)
