package memory

import (
	"context"
	"database/sql"
	"encoding/binary"
	"math"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/google/uuid"
	"github.com/supcode/supcode/pkg"
)

// Store implements pkg.MemoryStore backed by SQLite.
type Store struct {
	mu       sync.RWMutex
	db       *sql.DB
	embedder pkg.EmbeddingProvider // optional, for semantic search
}

// NewStore opens or creates a SQLite database at the given path.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		return nil, err
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS memories (
			id         TEXT PRIMARY KEY,
			scope      TEXT NOT NULL,
			category   TEXT NOT NULL,
			content    TEXT NOT NULL,
			embedding  BLOB,
			updated_at TEXT NOT NULL
		)
	`); err != nil {
		return nil, err
	}
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_memories_scope_cat
		ON memories(scope, category)
	`); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// NewStoreWithEmbedder creates a Store with an EmbeddingProvider for semantic search.
func NewStoreWithEmbedder(dbPath string, embedder pkg.EmbeddingProvider) (*Store, error) {
	s, err := NewStore(dbPath)
	if err != nil {
		return nil, err
	}
	s.embedder = embedder
	return s, nil
}

// Save insert or updates a memory entry (UPSERT by ID).
func (s *Store) Save(_ context.Context, entry pkg.MemoryEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	var embBlob []byte
	if entry.Embedding != nil {
		embBlob = floatsToBlob(entry.Embedding)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(
		`INSERT INTO memories (id, scope, category, content, embedding, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   scope      = excluded.scope,
		   category   = excluded.category,
		   content    = excluded.content,
		   embedding  = excluded.embedding,
		   updated_at = excluded.updated_at`,
		entry.ID, entry.Scope, entry.Category, entry.Content, embBlob, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

// Search performs semantic search if an EmbeddingProvider is configured,
// otherwise falls back to LIKE-based keyword search.
func (s *Store) Search(ctx context.Context, query string, scope string, limit int) ([]pkg.MemoryEntry, error) {
	if s.embedder != nil {
		entries, err := s.semanticSearch(ctx, query, scope, limit)
		if err == nil {
			return entries, nil
		}
		// Fall through to keyword search on error
	}
	return s.keywordSearch(ctx, query, scope, limit)
}

// keywordSearch uses SQL LIKE for fallback search.
func (s *Store) keywordSearch(_ context.Context, query string, scope string, limit int) ([]pkg.MemoryEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pattern := "%" + strings.ReplaceAll(query, "%", "\\%") + "%"
	var rows *sql.Rows
	var err error

	if scope != "" {
		rows, err = s.db.Query(
			`SELECT id, scope, category, content, embedding, updated_at
			 FROM memories WHERE scope = ? AND content LIKE ? ESCAPE '\'
			 ORDER BY updated_at DESC LIMIT ?`,
			scope, pattern, limit,
		)
	} else {
		rows, err = s.db.Query(
			`SELECT id, scope, category, content, embedding, updated_at
			 FROM memories WHERE content LIKE ? ESCAPE '\'
			 ORDER BY updated_at DESC LIMIT ?`,
			pattern, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEntries(rows)
}

// semanticSearch generates an embedding for the query and performs cosine similarity ranking.
func (s *Store) semanticSearch(ctx context.Context, query string, scope string, limit int) ([]pkg.MemoryEntry, error) {
	if s.embedder == nil {
		return s.keywordSearch(ctx, query, scope, limit)
	}

	queryEmb, err := s.embedder.Generate(ctx, query)
	if err != nil {
		return nil, err
	}

	all, err := s.getAllWithEmbeddings(ctx, scope)
	if err != nil {
		return nil, err
	}

	type scored struct {
		entry pkg.MemoryEntry
		score float64
	}
	var scoredEntries []scored

	for _, e := range all {
		sim := cosineSimilarity(queryEmb, e.Embedding)
		scoredEntries = append(scoredEntries, scored{entry: e, score: sim})
	}

	// Sort by score descending
	for i := 0; i < len(scoredEntries); i++ {
		for j := i + 1; j < len(scoredEntries); j++ {
			if scoredEntries[j].score > scoredEntries[i].score {
				scoredEntries[i], scoredEntries[j] = scoredEntries[j], scoredEntries[i]
			}
		}
	}

	if limit > 0 && limit < len(scoredEntries) {
		scoredEntries = scoredEntries[:limit]
	}

	result := make([]pkg.MemoryEntry, len(scoredEntries))
	for i, se := range scoredEntries {
		result[i] = se.entry
	}
	return result, nil
}

// getAllWithEmbeddings returns all entries (with embeddings) for a given scope.
func (s *Store) getAllWithEmbeddings(_ context.Context, scope string) ([]pkg.MemoryEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var rows *sql.Rows
	var err error

	if scope != "" {
		rows, err = s.db.Query(
			`SELECT id, scope, category, content, embedding, updated_at
			 FROM memories WHERE scope = ? AND embedding IS NOT NULL`,
			scope,
		)
	} else {
		rows, err = s.db.Query(
			`SELECT id, scope, category, content, embedding, updated_at
			 FROM memories WHERE embedding IS NOT NULL`,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEntries(rows)
}

// GetByCategory returns entries matching scope and category.
func (s *Store) GetByCategory(_ context.Context, scope string, category string) ([]pkg.MemoryEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(
		`SELECT id, scope, category, content, embedding, updated_at
		 FROM memories WHERE scope = ? AND category = ?
		 ORDER BY updated_at DESC`,
		scope, category,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEntries(rows)
}

// Delete removes a memory entry by ID.
func (s *Store) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM memories WHERE id = ?`, id)
	return err
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// --- helpers ---

func scanEntries(rows *sql.Rows) ([]pkg.MemoryEntry, error) {
	var entries []pkg.MemoryEntry
	for rows.Next() {
		var e pkg.MemoryEntry
		var embBlob []byte
		var updatedAtStr string
		if err := rows.Scan(&e.ID, &e.Scope, &e.Category, &e.Content, &embBlob, &updatedAtStr); err != nil {
			return nil, err
		}
		if embBlob != nil {
			e.Embedding = blobToFloats(embBlob)
		}
		parsed, _ := time.Parse(time.RFC3339, updatedAtStr)
		e.UpdatedAt = parsed
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// floatsToBlob encodes a []float32 as a big-endian byte blob.
func floatsToBlob(v []float32) []byte {
	buf := make([]byte, len(v)*4)
	for i, f := range v {
		binary.BigEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

// blobToFloats decodes a byte blob back to []float32.
func blobToFloats(data []byte) []float32 {
	if len(data) == 0 || len(data)%4 != 0 {
		return nil
	}
	v := make([]float32, len(data)/4)
	for i := range v {
		v[i] = math.Float32frombits(binary.BigEndian.Uint32(data[i*4:]))
	}
	return v
}

// cosineSimilarity computes cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
