package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/supcode/supcode/pkg"
)

// MockMemoryStore implements pkg.MemoryStore for testing.
type MockMemoryStore struct {
	mu               sync.Mutex
	entries          []pkg.MemoryEntry
	SaveFunc         func(ctx context.Context, entry pkg.MemoryEntry) error
	SearchFunc       func(ctx context.Context, query string, scope string, limit int) ([]pkg.MemoryEntry, error)
	GetByCategoryFunc func(ctx context.Context, scope string, category string) ([]pkg.MemoryEntry, error)
	DeleteFunc       func(ctx context.Context, id string) error
	CloseFunc        func() error
}

func (m *MockMemoryStore) Save(ctx context.Context, entry pkg.MemoryEntry) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, entry)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	m.entries = append(m.entries, entry)
	return nil
}

func (m *MockMemoryStore) Search(ctx context.Context, query string, scope string, limit int) ([]pkg.MemoryEntry, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, query, scope, limit)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []pkg.MemoryEntry
	for _, e := range m.entries {
		if scope != "" && e.Scope != scope {
			continue
		}
		result = append(result, e)
	}
	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}

func (m *MockMemoryStore) GetByCategory(ctx context.Context, scope string, category string) ([]pkg.MemoryEntry, error) {
	if m.GetByCategoryFunc != nil {
		return m.GetByCategoryFunc(ctx, scope, category)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []pkg.MemoryEntry
	for _, e := range m.entries {
		if e.Scope == scope && e.Category == category {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *MockMemoryStore) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, e := range m.entries {
		if e.ID == id {
			m.entries = append(m.entries[:i], m.entries[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *MockMemoryStore) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// MockEmbeddingProvider implements pkg.EmbeddingProvider for testing.
type MockEmbeddingProvider struct {
	GenerateFunc      func(ctx context.Context, text string) ([]float32, error)
	BatchGenerateFunc func(ctx context.Context, texts []string) ([][]float32, error)
}

func (m *MockEmbeddingProvider) Generate(ctx context.Context, text string) ([]float32, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, text)
	}
	// Return a simple deterministic vector for testing
	return []float32{float32(len(text))}, nil
}

func (m *MockEmbeddingProvider) BatchGenerate(ctx context.Context, texts []string) ([][]float32, error) {
	if m.BatchGenerateFunc != nil {
		return m.BatchGenerateFunc(ctx, texts)
	}
	result := make([][]float32, len(texts))
	for i, t := range texts {
		result[i] = []float32{float32(len(t))}
	}
	return result, nil
}

var _ pkg.MemoryStore = (*MockMemoryStore)(nil)
var _ pkg.EmbeddingProvider = (*MockEmbeddingProvider)(nil)
// UpdateEmbedding is required by pkg.MemoryStore interface.
func (m *MockMemoryStore) UpdateEmbedding(_ context.Context, id string, embedding []float32) error {
	if m.mu.TryLock() {
		defer m.mu.Unlock()
	}
	for i, e := range m.entries {
		if e.ID == id {
			m.entries[i].Embedding = embedding
			return nil
		}
	}
	return nil
}
