package permission

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/supcode/supcode/pkg"
)

const firstUseTableDDL = `CREATE TABLE IF NOT EXISTS first_use (
	tool_name TEXT PRIMARY KEY,
	authorized_at TEXT NOT NULL
);`

// FirstUseTracker tracks whether tools have been used before.
type FirstUseTracker struct {
	mu         sync.RWMutex
	db         *sql.DB
	authorized map[string]bool
}

func NewFirstUseTracker(dbPath string) (*FirstUseTracker, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open first_use db: %w", err)
	}
	if _, err := db.Exec(firstUseTableDDL); err != nil {
		db.Close()
		return nil, fmt.Errorf("create first_use table: %w", err)
	}
	t := &FirstUseTracker{db: db, authorized: make(map[string]bool)}
	if err := t.load(); err != nil {
		log.Printf("[FIRST_USE] load error: %v", err)
	}
	return t, nil
}

func (t *FirstUseTracker) load() error {
	rows, err := t.db.Query(`SELECT tool_name FROM first_use`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		t.authorized[name] = true
	}
	return rows.Err()
}

func (t *FirstUseTracker) IsFirstUse(toolName string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return !t.authorized[toolName]
}

func (t *FirstUseTracker) Authorize(toolName string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.authorized[toolName] = true
	_, err := t.db.Exec(`INSERT OR REPLACE INTO first_use (tool_name, authorized_at) VALUES (?, ?)`,
		toolName, time.Now().UTC().Format(time.RFC3339))
	return err
}

func (t *FirstUseTracker) CheckTool(ctx context.Context, action pkg.Action) (pkg.Decision, error) {
	if action.ToolName == "" {
		return pkg.DecisionAllow, nil
	}
	if t.IsFirstUse(action.ToolName) {
		return pkg.DecisionAsk, nil
	}
	return pkg.DecisionAllow, nil
}

func (t *FirstUseTracker) Close() error {
	return t.db.Close()
}

func (t *FirstUseTracker) AllAuthorized() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	keys := make([]string, 0, len(t.authorized))
	for k := range t.authorized {
		keys = append(keys, k)
	}
	return keys
}
