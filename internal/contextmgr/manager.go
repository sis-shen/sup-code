package contextmgr

import (
	"context"
	"sync"

	"github.com/supcode/supcode/pkg"
)

// sessionContext holds per-session in-memory state.
type sessionContext struct {
	mu                sync.RWMutex
	messages          []pkg.Message
	tokenCount        int
	threshold         int
	compressedSummary string
	memoryCards       []pkg.MemoryCard
}

// Manager implements pkg.ContextManager for short-term context management.
type Manager struct {
	mu            sync.RWMutex
	sessions      map[string]*sessionContext
	counter       *TokenCounter
	threshold     int
	compressor    *Compressor
	cardExtractor *CardExtractor
}

// NewManager creates a new Manager with default 80000 token threshold.
func NewManager() *Manager {
	return &Manager{
		sessions:      make(map[string]*sessionContext),
		counter:       NewTokenCounter(),
		threshold:     80000,
		cardExtractor: NewCardExtractor(),
	}
}

// NewManagerWithThreshold creates a new Manager with a custom threshold.
func NewManagerWithThreshold(threshold int) *Manager {
	return &Manager{
		sessions:      make(map[string]*sessionContext),
		counter:       NewTokenCounter(),
		threshold:     threshold,
		cardExtractor: NewCardExtractor(),
	}
}

// NewManagerWithCompression creates a fully configured Manager with LLM-based compression.
func NewManagerWithCompression(llm pkg.LLMClient) *Manager {
	return &Manager{
		sessions:      make(map[string]*sessionContext),
		counter:       NewTokenCounter(),
		threshold:     80000,
		compressor:    NewCompressor(llm),
		cardExtractor: NewCardExtractor(),
	}
}

// SetCompressor sets the compressor for LLM-based context compression.
func (m *Manager) SetCompressor(compressor *Compressor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.compressor = compressor
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
// Order: system prompt -> memory cards (if any) -> recent messages
func (m *Manager) BuildContext(_ context.Context, sessionID string) (string, []pkg.Message, error) {
	sc := m.getOrCreateSession(sessionID)
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	messages := make([]pkg.Message, len(sc.messages))
	copy(messages, sc.messages)

	// Build system prompt with memory cards
	systemPrompt := ""
	if cards := sc.memoryCards; len(cards) > 0 {
		systemPrompt = InjectCards(cards)
	}

	return systemPrompt, messages, nil
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

	// Don't recompress if already compressed
	if sc.compressedSummary != "" {
		return false, nil
	}

	return sc.tokenCount >= sc.threshold, nil
}

// Compress triggers context compression using Map-Reduce.
// If a compressor is configured, it runs compression asynchronously.
// Otherwise, it's a no-op (Phase 1 compatibility).
func (m *Manager) Compress(ctx context.Context, sessionID string) error {
	sc := m.getOrCreateSession(sessionID)

	// Check if compressor is available
	m.mu.RLock()
	comp := m.compressor
	m.mu.RUnlock()

	if comp == nil {
		return nil
	}

	// Run compression asynchronously
	go func() {
		sc.mu.RLock()
		msgs := make([]pkg.Message, len(sc.messages))
		copy(msgs, sc.messages)
		sc.mu.RUnlock()

		summary, err := comp.CompressOldMessages(ctx, msgs, defaultKeepRecent)
		if err != nil {
			return
		}

		if summary == "" {
			return
		}

		// Extract memory cards from the summary
		cards := m.cardExtractor.ExtractCards(summary)

		// Store results on the session
		setCompressAsyncResult(sc, summary, cards)
	}()

	return nil
}

// GetMemoryCards returns the current session's memory cards.
func (m *Manager) GetMemoryCards(_ context.Context, sessionID string) ([]pkg.MemoryCard, error) {
	sc := m.getOrCreateSession(sessionID)
	cards := getCards(sc)
	return cards, nil
}

// Clear removes all messages and resets the token count for a session.
func (m *Manager) Clear(_ context.Context, sessionID string) error {
	sc := m.getOrCreateSession(sessionID)
	sc.mu.Lock()
	sc.messages = make([]pkg.Message, 0)
	sc.tokenCount = 0
	sc.compressedSummary = ""
	sc.memoryCards = nil
	sc.mu.Unlock()
	return nil
}
