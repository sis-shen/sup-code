package memory

import (
	"context"
	"math"
	"os"
	"testing"

	"github.com/supcode/supcode/pkg"
)

func tempDB(t *testing.T) *Store {
	t.Helper()
	f, err := os.CreateTemp("", "memory-test-*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	f.Close()
	s, err := NewStore(f.Name())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() {
		s.Close()
		os.Remove(f.Name())
	})
	return s
}

func TestStore_SaveAndSearch(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	entry := pkg.MemoryEntry{
		Scope:    "project",
		Category: "structure",
		Content:  "The project uses a layered architecture",
	}
	if err := s.Save(ctx, entry); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Default search (LIKE fallback)
	results, err := s.Search(ctx, "layered", "", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Search returned %d results, want 1", len(results))
	}
	if len(results) > 0 && results[0].Content != entry.Content {
		t.Errorf("Search content = %q, want %q", results[0].Content, entry.Content)
	}
}

func TestStore_SaveAndSearch_Scoped(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	s.Save(ctx, pkg.MemoryEntry{Scope: "project", Category: "task", Content: "task one"})
	s.Save(ctx, pkg.MemoryEntry{Scope: "user", Category: "preference", Content: "prefer dark mode"})

	// Search with scope
	results, err := s.Search(ctx, "task", "project", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("scoped Search returned %d results, want 1", len(results))
	}

	// Search without scope (should find both matching "prefer")
	results, err = s.Search(ctx, "prefer", "", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Search 'prefer' = %d results, want 1", len(results))
	}
}

func TestStore_SearchLimit(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	for i := range 5 {
		s.Save(ctx, pkg.MemoryEntry{
			Scope: "project", Category: "note",
			Content: "entry number " + string(rune('0'+i)),
		})
	}

	results, _ := s.Search(ctx, "entry", "", 3)
	if len(results) > 3 {
		t.Errorf("Search limit = %d results, want <= 3", len(results))
	}
}

func TestStore_EmptySearch(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	results, err := s.Search(ctx, "nothing", "", 10)
	if err != nil {
		t.Fatalf("Search on empty store: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("empty store Search = %d results, want 0", len(results))
	}
}

func TestStore_GetByCategory(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	s.Save(ctx, pkg.MemoryEntry{Scope: "project", Category: "structure", Content: "layered arch"})
	s.Save(ctx, pkg.MemoryEntry{Scope: "project", Category: "convention", Content: "use tabs"})
	s.Save(ctx, pkg.MemoryEntry{Scope: "user", Category: "preference", Content: "vim mode"})

	results, err := s.GetByCategory(ctx, "project", "structure")
	if err != nil {
		t.Fatalf("GetByCategory: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("GetByCategory = %d results, want 1", len(results))
	}
	if len(results) > 0 && results[0].Content != "layered arch" {
		t.Errorf("content = %q, want %q", results[0].Content, "layered arch")
	}

	// Empty category
	results, _ = s.GetByCategory(ctx, "project", "nonexistent")
	if len(results) != 0 {
		t.Errorf("nonexistent category = %d results, want 0", len(results))
	}
}

func TestStore_Delete(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	entry := pkg.MemoryEntry{Scope: "project", Category: "task", Content: "to delete"}
	if err := s.Save(ctx, entry); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Get all saved entries to retrieve the ID
	results, _ := s.Search(ctx, "to delete", "", 10)
	if len(results) == 0 {
		t.Fatal("no results to delete")
	}
	id := results[0].ID

	if err := s.Delete(ctx, id); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	results, _ = s.Search(ctx, "to delete", "", 10)
	if len(results) != 0 {
		t.Errorf("Search after Delete = %d results, want 0", len(results))
	}
}

func TestStore_Close(t *testing.T) {
	s := tempDB(t)
	if err := s.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestStore_SemanticSearch(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	mockEmb := &MockEmbeddingProvider{
		GenerateFunc: func(_ context.Context, text string) ([]float32, error) {
			// Simple deterministic: length-based vector for testing
			return []float32{float32(len(text))}, nil
		},
	}
	s.embedder = mockEmb

	entry1 := pkg.MemoryEntry{
		Scope:     "project",
		Category:  "structure",
		Content:   "longer architecture description",
		Embedding: []float32{28.0},
	}
	entry2 := pkg.MemoryEntry{
		Scope:     "user",
		Category:  "preference",
		Content:   "hi",
		Embedding: []float32{2.0},
	}
	s.Save(ctx, entry1)
	s.Save(ctx, entry2)

	results, err := s.Search(ctx, "architecture pattern", "", 10)
	if err != nil {
		t.Fatalf("SemanticSearch: %v", err)
	}
	// With our mock, "architecture pattern" (20 chars) has embedding [20],
	// so it should have similarity closer to entry1 ([28]) than entry2 ([2]).
	if len(results) > 0 && results[0].Content != entry1.Content {
		t.Errorf("top result should be the architecture entry, got %q", results[0].Content)
	}
}

func TestStore_SemanticSearch_Fallback(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	// No embedder set -> falls back to LIKE search
	s.Save(ctx, pkg.MemoryEntry{Scope: "project", Category: "note", Content: "important note"})

	results, err := s.Search(ctx, "important", "", 10)
	if err != nil {
		t.Fatalf("Search fallback: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("fallback Search = %d results, want 1", len(results))
	}
}

func TestStore_ConcurrentSave(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	var wg testWaitGroup
	wg.Add(20)
	for i := range 20 {
		go func(i int) {
			defer wg.Done()
			err := s.Save(ctx, pkg.MemoryEntry{
				Scope:    "project",
				Category: "note",
				Content:  "concurrent entry",
			})
			if err != nil {
				t.Errorf("concurrent Save: %v", err)
			}
		}(i)
	}
	wg.Wait()

	results, _ := s.Search(ctx, "concurrent", "", 100)
	if len(results) != 20 {
		t.Errorf("concurrent save got %d entries, want 20", len(results))
	}
}

// Helper to avoid data races in test itself
type testWaitGroup struct {
	ch chan struct{}
}

func (w *testWaitGroup) Add(n int) {
	w.ch = make(chan struct{}, n)
}

func (w *testWaitGroup) Done() {
	if w.ch != nil {
		w.ch <- struct{}{}
	}
}

func (w *testWaitGroup) Wait() {
	if w.ch != nil {
		for i := cap(w.ch); i > 0; i-- {
			<-w.ch
		}
	}
}

func TestFloatsBlobRoundtrip(t *testing.T) {
	original := []float32{1.0, 2.5, -3.14, 0.0, math.MaxFloat32}
	blob := floatsToBlob(original)
	decoded := blobToFloats(blob)

	if len(decoded) != len(original) {
		t.Fatalf("len mismatch: %d vs %d", len(decoded), len(original))
	}
	for i := range original {
		if original[i] != decoded[i] {
			t.Errorf("index %d: %f vs %f", i, original[i], decoded[i])
		}
	}
}

func TestCosineSimilarity(t *testing.T) {
	a := []float32{1, 0}
	b := []float32{0, 1}
	if sim := cosineSimilarity(a, b); sim != 0 {
		t.Errorf("orthogonal vectors similarity = %f, want 0", sim)
	}

	c := []float32{1, 2}
	d := []float32{2, 4}
	sim := cosineSimilarity(c, d)
	if math.Abs(sim-1.0) > 0.0001 {
		t.Errorf("parallel vectors similarity = %f, want 1.0", sim)
	}
}

func TestBlobToFloats_Nil(t *testing.T) {
	if v := blobToFloats(nil); v != nil {
		t.Errorf("nil blob should return nil")
	}
	if v := blobToFloats([]byte{1, 2, 3}); v != nil {
		t.Errorf("non-4-byte-aligned blob should return nil")
	}
}

func TestCosineSimilarity_Empty(t *testing.T) {
	if sim := cosineSimilarity(nil, []float32{1}); sim != 0 {
		t.Errorf("mismatched lengths should return 0")
	}
	if sim := cosineSimilarity([]float32{}, []float32{}); sim != 0 {
		t.Errorf("empty vectors should return 0")
	}
}

func TestStore_SaveWithEmbedding(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	entry := pkg.MemoryEntry{
		Scope:     "project",
		Category:  "structure",
		Content:   "system architecture",
		Embedding: []float32{0.1, 0.2, 0.3},
	}
	if err := s.Save(ctx, entry); err != nil {
		t.Fatalf("Save with embedding: %v", err)
	}

	results, _ := s.Search(ctx, "architecture", "", 10)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Embedding == nil {
		t.Errorf("embedding should not be nil after round-trip")
	}
}
