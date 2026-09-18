package permission

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/supcode/supcode/pkg"
)

// AuditEntry represents a single audit log record.
type AuditEntry struct {
	ID            int64     `json:"id"`
	SessionID     string    `json:"session_id"`
	ToolName      string    `json:"tool_name"`
	Params        string    `json:"params"`
	ResultSummary string    `json:"result_summary"`
	Decision      string    `json:"decision"`
	Timestamp     time.Time `json:"timestamp"`
}

const (
	auditTableDDL = `CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL DEFAULT "",
		tool_name TEXT NOT NULL DEFAULT "",
		params TEXT NOT NULL DEFAULT "",
		result_summary TEXT NOT NULL DEFAULT "",
		decision TEXT NOT NULL DEFAULT "",
		timestamp TEXT NOT NULL
	);`
	maxParamsLen  = 1024
	maxSummaryLen = 500
)

type auditRecord struct {
	sessionID string
	action    pkg.Action
	decision  pkg.Decision
	result    string
}

type AuditLogger struct {
	db       *sql.DB
	insertCh chan auditRecord
	wg       sync.WaitGroup
	closeMu  sync.Mutex
	closed   bool

	pendingWg sync.WaitGroup
}

func NewAuditLogger(dbPath string) (*AuditLogger, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open audit db: %w", err)
	}
	if _, err := db.Exec(auditTableDDL); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("[AUDIT] close db after table creation failure: %v", closeErr)
		}
		return nil, fmt.Errorf("create audit table: %w", err)
	}
	a := &AuditLogger{db: db, insertCh: make(chan auditRecord, 64)}
	a.wg.Add(1)
	go a.batchWriter()
	return a, nil
}

func (a *AuditLogger) batchWriter() {
	defer a.wg.Done()
	stmt, err := a.db.Prepare(`INSERT INTO audit_logs
		(session_id, tool_name, params, result_summary, decision, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		log.Printf("[AUDIT] prepare statement failed: %v", err)
		return
	}
	defer func() { _ = stmt.Close() }()

	for rec := range a.insertCh {
		params := sanitizeParams(truncate(rec.action.Params, maxParamsLen))
		result := truncate(rec.result, maxSummaryLen)
		toolName := rec.action.ToolName
		if toolName == "" && rec.action.Type != "" {
			toolName = rec.action.Type
		}
		_, err := stmt.Exec(rec.sessionID, toolName, params, result, string(rec.decision), time.Now().UTC().Format(time.RFC3339))
		if err != nil {
			log.Printf("[AUDIT] insert failed: %v", err)
		}
		a.pendingWg.Done()
	}
}

func (a *AuditLogger) LogAction(ctx context.Context, sessionID string, action pkg.Action, decision pkg.Decision, result string) error {
	a.pendingWg.Add(1)
	select {
	case a.insertCh <- auditRecord{sessionID: sessionID, action: action, decision: decision, result: result}:
	default:
		a.pendingWg.Done()
		log.Printf("[AUDIT] channel full, dropping audit record for %s/%s", action.Type, action.Target)
	}
	return nil
}

// Sync waits for all pending writes to complete.
func (a *AuditLogger) Sync() {
	a.pendingWg.Wait()
}

func (a *AuditLogger) Query(sessionID string, limit int) ([]AuditEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := a.db.Query(
		`SELECT id, session_id, tool_name, params, result_summary, decision, timestamp
		 FROM audit_logs WHERE session_id = ?
		 ORDER BY timestamp DESC LIMIT ?`, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("query audit: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanEntries(rows)
}

func (a *AuditLogger) QueryByTool(toolName string, limit int) ([]AuditEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := a.db.Query(
		`SELECT id, session_id, tool_name, params, result_summary, decision, timestamp
		 FROM audit_logs WHERE tool_name = ?
		 ORDER BY timestamp DESC LIMIT ?`, toolName, limit)
	if err != nil {
		return nil, fmt.Errorf("query audit by tool: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanEntries(rows)
}

func scanEntries(rows *sql.Rows) ([]AuditEntry, error) {
	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var ts string
		if err := rows.Scan(&e.ID, &e.SessionID, &e.ToolName, &e.Params, &e.ResultSummary, &e.Decision, &ts); err != nil {
			return nil, fmt.Errorf("scan audit entry: %w", err)
		}
		t, err := time.Parse(time.RFC3339, ts)
		if err == nil {
			e.Timestamp = t
		}
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []AuditEntry{}
	}
	return entries, rows.Err()
}

func (a *AuditLogger) Close() error {
	a.closeMu.Lock()
	if a.closed {
		a.closeMu.Unlock()
		return nil
	}
	a.closed = true
	a.closeMu.Unlock()
	close(a.insertCh)
	a.wg.Wait()
	return a.db.Close()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func sanitizeParams(s string) string {
	if strings.Contains(s, "sk-") {
		return "[REDACTED: potential API key]"
	}
	return s
}
