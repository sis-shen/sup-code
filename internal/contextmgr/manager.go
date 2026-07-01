package contextmgr

import (
	"context"
	"sync"

	"github.com/supcode/supcode/pkg"
)

// sessionContext holds per-session in-memory state.
type sessionContext struct {
	mu         sync.RWMutex
	messages   []pkg.Message
	tokenCount int
	threshold  int
}

// Manager implements pkg.ContextManager for short-term context management (MVP).
type Manager struct {
	mu        sync.RWMutex
	sessions  map[string]*sessionContext
	counter   *TokenCounter
	threshold int
}

// NewManager creates a new Manager with default 100000 token threshold.
func NewManager() *Manager {
	return &Manager{
		sessions:  make(map[string]*sessionContext),
		counter:   NewTokenCounter(),
		threshold: 100000,
	}
}

// NewManagerWithThreshold creates a new Manager with a custom threshold.
func NewManagerWithThreshold(threshold int) *Manager {
	return &Manager{
		sessions:  make(map[string]*sessionContext),
		counter:   NewTokenCounter(),
		threshold: threshold,
	}
}

func (m *Manager) getOrCreateSession(sessionID string) *sessionContext {
	m.mu.RLock()
	sc, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if ok {
		return sc
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if sc, ok := m.sessions[sessionID]; ok {
		return sc
	}

	sc = &sessionContext{
		messages:  make([]pkg.Message, 0),
		threshold: m.threshold,
	}
	m.sessions[sessionID] = sc
	return sc
}

// BuildContext returns the system prompt and session messages for a given session.
func (m *Manager) BuildContext(_ context.Context, sessionID string) (string, []pkg.Message, error) {
	sc := m.getOrCreateSession(sessionID)
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	messages := make([]pkg.Message, len(sc.messages))
	copy(messages, sc.messages)

	return "", messages, nil
}

// AppendMessage adds a message to the session and updates the token count.
func (m *Manager) AppendMessage(_ context.Context, sessionID string, msg pkg.Message) error {
	sc := m.getOrCreateSession(sessionID)

	count, err := m.counter.CountMessages([]pkg.Message{msg})
	if err != nil {
		return err
	}

	sc.mu.Lock()
	sc.messages = append(sc.messages, msg)
	sc.tokenCount += count
	sc.mu.Unlock()

	return nil
}

// TokenCount returns the current estimated token count for the session.
func (m *Manager) TokenCount(_ context.Context, sessionID string) (int, error) {
	sc := m.getOrCreateSession(sessionID)
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.tokenCount, nil
}

// ShouldCompress returns true if the session's token count has reached the threshold.
func (m *Manager) ShouldCompress(_ context.Context, sessionID string) (bool, error) {
	sc := m.getOrCreateSession(sessionID)
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.tokenCount >= sc.threshold, nil
}

// Compress is a stub — compression is not implemented in MVP.
func (m *Manager) Compress(_ context.Context, _ string) error {
	return nil
}

// GetMemoryCards is a stub — long-term memory is not implemented in MVP.
func (m *Manager) GetMemoryCards(_ context.Context, _ string) ([]pkg.MemoryCard, error) {
	return nil, nil
}

// Clear removes all messages and resets the token count for a session.
func (m *Manager) Clear(_ context.Context, sessionID string) error {
	sc := m.getOrCreateSession(sessionID)
	sc.mu.Lock()
	sc.messages = make([]pkg.Message, 0)
	sc.tokenCount = 0
	sc.mu.Unlock()
	return nil
}
