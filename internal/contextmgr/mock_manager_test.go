package contextmgr

import (
	"context"

	"github.com/supcode/supcode/pkg"
)

// MockContextManager implements pkg.ContextManager for testing.
type MockContextManager struct {
	BuildContextFunc   func(ctx context.Context, sessionID string) (string, []pkg.Message, error)
	AppendMessageFunc  func(ctx context.Context, sessionID string, msg pkg.Message) error
	TokenCountFunc     func(ctx context.Context, sessionID string) (int, error)
	ShouldCompressFunc func(ctx context.Context, sessionID string) (bool, error)
	CompressFunc       func(ctx context.Context, sessionID string) error
	GetMemoryCardsFunc func(ctx context.Context, sessionID string) ([]pkg.MemoryCard, error)
	ClearFunc          func(ctx context.Context, sessionID string) error
}

func (m *MockContextManager) BuildContext(ctx context.Context, sessionID string) (string, []pkg.Message, error) {
	if m.BuildContextFunc != nil {
		return m.BuildContextFunc(ctx, sessionID)
	}
	return "", nil, nil
}

func (m *MockContextManager) AppendMessage(ctx context.Context, sessionID string, msg pkg.Message) error {
	if m.AppendMessageFunc != nil {
		return m.AppendMessageFunc(ctx, sessionID, msg)
	}
	return nil
}

func (m *MockContextManager) TokenCount(ctx context.Context, sessionID string) (int, error) {
	if m.TokenCountFunc != nil {
		return m.TokenCountFunc(ctx, sessionID)
	}
	return 0, nil
}

func (m *MockContextManager) ShouldCompress(ctx context.Context, sessionID string) (bool, error) {
	if m.ShouldCompressFunc != nil {
		return m.ShouldCompressFunc(ctx, sessionID)
	}
	return false, nil
}

func (m *MockContextManager) Compress(ctx context.Context, sessionID string) error {
	if m.CompressFunc != nil {
		return m.CompressFunc(ctx, sessionID)
	}
	return nil
}

func (m *MockContextManager) GetMemoryCards(ctx context.Context, sessionID string) ([]pkg.MemoryCard, error) {
	if m.GetMemoryCardsFunc != nil {
		return m.GetMemoryCardsFunc(ctx, sessionID)
	}
	return nil, nil
}

func (m *MockContextManager) Clear(ctx context.Context, sessionID string) error {
	if m.ClearFunc != nil {
		return m.ClearFunc(ctx, sessionID)
	}
	return nil
}
