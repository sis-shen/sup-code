package tui

import (
	"context"
	"database/sql"
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

type sessionData struct {
	ID         string          `json:"id"`
	Title      string          `json:"title"`
	Messages   []pkg.Message   `json:"messages"`
	Plan       json.RawMessage `json:"plan,omitempty"`
	State      string          `json:"state"`
	TokenCount int             `json:"token_count"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type storeFile struct {
	Sessions map[string]*sessionData `json:"sessions"`
}

type SessionManager struct {
	mu       sync.RWMutex
	cache    map[string]*cacheEntry
	maxHot   int
	dbPath   string
	sqliteDB *sql.DB
}

func NewSessionManager(dbPath string) (*SessionManager, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	sm := &SessionManager{cache: make(map[string]*cacheEntry), maxHot: 10, dbPath: dbPath}
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
		s := &pkg.Session{
			ID: sd.ID, Title: sd.Title, Messages: sd.Messages,
			State: pkg.LoopState(sd.State), TokenCount: sd.TokenCount,
			CreatedAt: sd.CreatedAt, UpdatedAt: sd.UpdatedAt,
		}
		if sd.Plan != nil {
			var plan pkg.Plan
			if json.Unmarshal(sd.Plan, &plan) == nil {
				s.Plan = &plan
			}
		}
		sm.cache[sd.ID] = &cacheEntry{session: s, dirty: false}
	}
	return nil
}

func (sm *SessionManager) saveToDisk() error {
	sm.mu.RLock()
	sessions := sm.buildSessionData()
	sm.mu.RUnlock()
	return sm.writeStore(storeFile{Sessions: sessions})
}

func (sm *SessionManager) saveToDiskLocked() error {
	sessions := sm.buildSessionData()
	return sm.writeStore(storeFile{Sessions: sessions})
}

func (sm *SessionManager) buildSessionData() map[string]*sessionData {
	sessions := make(map[string]*sessionData, len(sm.cache))
	for id, e := range sm.cache {
		sd := &sessionData{
			ID: e.session.ID, Title: e.session.Title, Messages: e.session.Messages,
			State: string(e.session.State), TokenCount: e.session.TokenCount,
			CreatedAt: e.session.CreatedAt, UpdatedAt: e.session.UpdatedAt,
		}
		if e.session.Plan != nil {
			if data, err := json.Marshal(e.session.Plan); err == nil {
				sd.Plan = data
			}
		}
		sessions[id] = sd
	}
	return sessions
}

func (sm *SessionManager) writeStore(store storeFile) error {
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

func (sm *SessionManager) Create(ctx context.Context, title string) (*pkg.Session, error) {
	now := time.Now().UTC()
	s := &pkg.Session{
		ID: uuid.New().String(), Title: title, Messages: []pkg.Message{},
		State: pkg.StateIdle, CreatedAt: now, UpdatedAt: now,
	}
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.evictIfNeeded()
	sm.cache[s.ID] = &cacheEntry{session: s, dirty: true}
	return s, nil
}

func (sm *SessionManager) Get(ctx context.Context, sid string) (*pkg.Session, error) {
	sm.mu.RLock()
	e, ok := sm.cache[sid]
	sm.mu.RUnlock()
	if ok {
		return e.session, nil
	}
	if err := sm.loadFromDisk(); err == nil {
		sm.mu.RLock()
		e, ok = sm.cache[sid]
		sm.mu.RUnlock()
		if ok {
			return e.session, nil
		}
	}
	return nil, fmt.Errorf("session not found: %s", sid)
}

func (sm *SessionManager) List(ctx context.Context) ([]*pkg.Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	var ss []*pkg.Session
	for _, e := range sm.cache {
		ss = append(ss, e.session)
	}
	for i := 0; i < len(ss); i++ {
		for j := i + 1; j < len(ss); j++ {
			if ss[j].UpdatedAt.After(ss[i].UpdatedAt) {
				ss[i], ss[j] = ss[j], ss[i]
			}
		}
	}
	if ss == nil {
		ss = []*pkg.Session{}
	}
	return ss, nil
}

func (sm *SessionManager) AppendMessage(ctx context.Context, sid string, msg pkg.Message) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	e, ok := sm.cache[sid]
	if !ok {
		return fmt.Errorf("session not found: %s", sid)
	}
	e.session.Messages = append(e.session.Messages, msg)
	e.session.UpdatedAt = time.Now().UTC()
	e.dirty = true
	return nil
}

func (sm *SessionManager) SetState(ctx context.Context, sid string, state pkg.LoopState) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	e, ok := sm.cache[sid]
	if !ok {
		return fmt.Errorf("session not found: %s", sid)
	}
	e.session.State = state
	e.session.UpdatedAt = time.Now().UTC()
	e.dirty = true
	return nil
}

func (sm *SessionManager) SetPlan(ctx context.Context, sid string, plan pkg.Plan) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	e, ok := sm.cache[sid]
	if !ok {
		return fmt.Errorf("session not found: %s", sid)
	}
	e.session.Plan = &plan
	e.session.UpdatedAt = time.Now().UTC()
	e.dirty = true
	return nil
}

func (sm *SessionManager) Delete(ctx context.Context, sid string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.cache, sid)
	return nil
}

func (sm *SessionManager) Close(ctx context.Context, sid string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if _, ok := sm.cache[sid]; !ok {
		return fmt.Errorf("session not found in cache: %s", sid)
	}
	// Persist dirty data before closing
	if e, ok := sm.cache[sid]; ok && e.dirty {
		sm.saveToDiskLocked()
	}
	delete(sm.cache, sid)
	return nil
}

func (sm *SessionManager) CloseAll() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if err := sm.saveToDiskLocked(); err != nil {
		slog.Error("failed to save sessions", "error", err)
	}
	sm.cache = make(map[string]*cacheEntry)
	return nil
}

func (sm *SessionManager) evictIfNeeded() {
	if len(sm.cache) < sm.maxHot {
		return
	}
	var oldestID string
	var oldestTime time.Time
	first := true
	for id, e := range sm.cache {
		if first || e.session.UpdatedAt.Before(oldestTime) {
			oldestID = id
			oldestTime = e.session.UpdatedAt
			first = false
		}
	}
	if oldestID != "" {
		if _, ok := sm.cache[oldestID]; ok {
			sm.saveToDiskLocked()
			delete(sm.cache, oldestID)
		}
	}
}

func (sm *SessionManager) DBPath() string { return sm.dbPath }
func (sm *SessionManager) CacheSize() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.cache)
}