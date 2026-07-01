package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/supcode/supcode/pkg"
)

var _ pkg.SessionManager = (*SessionManager)(nil)

type cacheEntry struct {
	session *pkg.Session
	dirty   bool
}

type SessionManager struct {
	mu     sync.RWMutex
	cache  map[string]*cacheEntry
	maxHot int
	dbPath string
}

type sessionData struct {
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	Messages     []pkg.Message   `json:"messages"`
	Plan         json.RawMessage `json:"plan,omitempty"`
	State        string          `json:"state"`
	TokenCount   int             `json:"token_count"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type storeFile struct {
	Sessions map[string]*sessionData `json:"sessions"`
}

func NewSessionManager(dbPath string) (*SessionManager, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	sm := &SessionManager{
		cache:  make(map[string]*cacheEntry),
		maxHot: 10,
		dbPath: dbPath,
	}

	if err := sm.loadFromDisk(); err != nil {
		slog.Warn("failed to load persisted sessions", "error", err)
	}

	return sm, nil
}

func (sm *SessionManager) loadFromDisk() error {
	data, err := os.ReadFile(sm.dbPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var store storeFile
	if err := json.Unmarshal(data, &store); err != nil {
		return err
	}

	for _, sd := range store.Sessions {
		sm.cache[sd.ID] = &cacheEntry{
			session: &pkg.Session{
				ID:         sd.ID,
				Title:      sd.Title,
				Messages:   sd.Messages,
				State:      pkg.LoopState(sd.State),
				TokenCount: sd.TokenCount,
				CreatedAt:  sd.CreatedAt,
				UpdatedAt:  sd.UpdatedAt,
			},
			dirty: false,
		}
		if sd.Plan != nil {
			var plan pkg.Plan
			if err := json.Unmarshal(sd.Plan, &plan); err == nil {
				sm.cache[sd.ID].session.Plan = &plan
			}
		}
	}

	return nil
}

func (sm *SessionManager) saveToDisk() error {
	sm.mu.RLock()
	sessions := make(map[string]*sessionData, len(sm.cache))
	for id, entry := range sm.cache {
		sd := &sessionData{
			ID:         entry.session.ID,
			Title:      entry.session.Title,
			Messages:   entry.session.Messages,
			State:      string(entry.session.State),
			TokenCount: entry.session.TokenCount,
			CreatedAt:  entry.session.CreatedAt,
			UpdatedAt:  entry.session.UpdatedAt,
		}
		if entry.session.Plan != nil {
			data, err := json.Marshal(entry.session.Plan)
			if err == nil {
				sd.Plan = data
			}
		}
		sessions[id] = sd
	}
	sm.mu.RUnlock()

	store := storeFile{Sessions: sessions}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sessions: %w", err)
	}

	dir := filepath.Dir(sm.dbPath)
	tmpFile := filepath.Join(dir, ".sessions_tmp.json")
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := os.Rename(tmpFile, sm.dbPath); err != nil {
		return fmt.Errorf("rename: %w", err)
	}

	return nil
}

func (sm *SessionManager) Create(ctx context.Context, title string) (*pkg.Session, error) {
	now := time.Now().UTC()
	session := &pkg.Session{
		ID:        uuid.New().String(),
		Title:     title,
		Messages:  []pkg.Message{},
		State:     pkg.StateIdle,
		CreatedAt: now,
		UpdatedAt: now,
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.evictIfNeeded()
	sm.cache[session.ID] = &cacheEntry{session: session, dirty: true}

	return session, nil
}

func (sm *SessionManager) Get(ctx context.Context, sessionID string) (*pkg.Session, error) {
	sm.mu.RLock()
	entry, ok := sm.cache[sessionID]
	sm.mu.RUnlock()

	if ok {
		return entry.session, nil
	}

	if err := sm.loadFromDisk(); err == nil {
		sm.mu.RLock()
		entry, ok = sm.cache[sessionID]
		sm.mu.RUnlock()
		if ok {
			return entry.session, nil
		}
	}

	return nil, fmt.Errorf("session not found: %s", sessionID)
}

func (sm *SessionManager) List(ctx context.Context) ([]*pkg.Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*pkg.Session, 0, len(sm.cache))
	for _, entry := range sm.cache {
		sessions = append(sessions, entry.session)
	}

	for i := 0; i < len(sessions); i++ {
		for j := i + 1; j < len(sessions); j++ {
			if sessions[j].UpdatedAt.After(sessions[i].UpdatedAt) {
				sessions[i], sessions[j] = sessions[j], sessions[i]
			}
		}
	}

	return sessions, nil
}

func (sm *SessionManager) AppendMessage(ctx context.Context, sessionID string, msg pkg.Message) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	entry, ok := sm.cache[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	entry.session.Messages = append(entry.session.Messages, msg)
	entry.session.UpdatedAt = time.Now().UTC()
	entry.dirty = true

	return nil
}

func (sm *SessionManager) SetState(ctx context.Context, sessionID string, state pkg.LoopState) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	entry, ok := sm.cache[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	entry.session.State = state
	entry.session.UpdatedAt = time.Now().UTC()
	entry.dirty = true

	return nil
}

func (sm *SessionManager) SetPlan(ctx context.Context, sessionID string, plan pkg.Plan) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	entry, ok := sm.cache[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	entry.session.Plan = &plan
	entry.session.UpdatedAt = time.Now().UTC()
	entry.dirty = true

	return nil
}

func (sm *SessionManager) Delete(ctx context.Context, sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.cache, sessionID)

	return nil
}

func (sm *SessionManager) Close(ctx context.Context, sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	entry, ok := sm.cache[sessionID]
	if !ok {
		return fmt.Errorf("session not found in cache: %s", sessionID)
	}

	entry.dirty = true

	saveErr := sm.saveToDiskNow()
	delete(sm.cache, sessionID)

	return saveErr
}

func (sm *SessionManager) CloseAll() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if err := sm.saveToDiskNow(); err != nil {
		slog.Error("failed to save sessions", "error", err)
	}

	sm.cache = make(map[string]*cacheEntry)
	return nil
}

func (sm *SessionManager) saveToDiskNow() error {
	sessions := make(map[string]*sessionData, len(sm.cache))
	for id, entry := range sm.cache {
		sd := &sessionData{
			ID:         entry.session.ID,
			Title:      entry.session.Title,
			Messages:   entry.session.Messages,
			State:      string(entry.session.State),
			TokenCount: entry.session.TokenCount,
			CreatedAt:  entry.session.CreatedAt,
			UpdatedAt:  entry.session.UpdatedAt,
		}
		if entry.session.Plan != nil {
			data, err := json.Marshal(entry.session.Plan)
			if err == nil {
				sd.Plan = data
			}
		}
		sessions[id] = sd
	}

	store := storeFile{Sessions: sessions}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	dir := filepath.Dir(sm.dbPath)
	tmpFile := filepath.Join(dir, ".sessions_tmp.json")
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("write temp: %w", err)
	}

	return os.Rename(tmpFile, sm.dbPath)
}

func (sm *SessionManager) evictIfNeeded() {
	if len(sm.cache) < sm.maxHot {
		return
	}

	var oldestID string
	var oldestTime time.Time
	first := true
	for id, entry := range sm.cache {
		if first || entry.session.UpdatedAt.Before(oldestTime) {
			oldestID = id
			oldestTime = entry.session.UpdatedAt
			first = false
		}
	}

	if oldestID != "" {
		delete(sm.cache, oldestID)
	}
}

func (sm *SessionManager) DBPath() string { return sm.dbPath }
func (sm *SessionManager) CacheSize() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.cache)
}
