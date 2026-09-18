package permission

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/pkg"
)

func setupAuditLogger(t *testing.T) *AuditLogger {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "audit_test.db")
	a, err := NewAuditLogger(dbPath)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	t.Cleanup(func() { require.NoError(t, a.Close()) })
	return a
}

func TestAuditLogActionAndQuery(t *testing.T) {
	a := setupAuditLogger(t)
	ctx := context.Background()

	err := a.LogAction(ctx, "sess-1", pkg.Action{Type: "command", Target: "ls -la", ToolName: "bash"}, pkg.DecisionAllow, "executed successfully")
	if err != nil {
		t.Fatalf("LogAction failed: %v", err)
	}
	a.Sync()

	entries, err := a.Query("sess-1", 10)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].SessionID != "sess-1" {
		t.Errorf("session_id = %q, want %q", entries[0].SessionID, "sess-1")
	}
	if entries[0].ToolName != "bash" {
		t.Errorf("tool_name = %q, want %q", entries[0].ToolName, "bash")
	}
	if entries[0].Decision != "allow" {
		t.Errorf("decision = %q, want %q", entries[0].Decision, "allow")
	}
}

func TestAuditParamsTruncation(t *testing.T) {
	a := setupAuditLogger(t)
	longParams := strings.Repeat("A", 2000)

	err := a.LogAction(context.Background(), "sess-2", pkg.Action{Type: "tool", ToolName: "test", Params: longParams}, pkg.DecisionAsk, "ok")
	if err != nil {
		t.Fatalf("LogAction failed: %v", err)
	}
	a.Sync()

	entries, err := a.Query("sess-2", 10)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if len(entries[0].Params) > maxParamsLen {
		t.Errorf("params length = %d, want <= %d", len(entries[0].Params), maxParamsLen)
	}
}

func TestAuditResultSummaryTruncation(t *testing.T) {
	a := setupAuditLogger(t)
	longResult := strings.Repeat("B", 1000)

	err := a.LogAction(context.Background(), "sess-3", pkg.Action{Type: "tool", ToolName: "test"}, pkg.DecisionDeny, longResult)
	if err != nil {
		t.Fatalf("LogAction failed: %v", err)
	}
	a.Sync()

	entries, err := a.Query("sess-3", 10)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if len(entries[0].ResultSummary) > maxSummaryLen {
		t.Errorf("result_summary length = %d, want <= %d", len(entries[0].ResultSummary), maxSummaryLen)
	}
}

func TestAuditNoAPIKey(t *testing.T) {
	a := setupAuditLogger(t)

	err := a.LogAction(context.Background(), "sess-4", pkg.Action{Type: "tool", ToolName: "openai", Params: `{"api_key":"sk-abc123"}`}, pkg.DecisionDeny, "blocked")
	if err != nil {
		t.Fatalf("LogAction failed: %v", err)
	}
	a.Sync()

	entries, err := a.Query("sess-4", 10)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Params == `{"api_key":"sk-abc123"}` {
		t.Errorf("params should not contain raw API key, got %q", entries[0].Params)
	}
	if !strings.Contains(entries[0].Params, "REDACTED") {
		t.Errorf("params should be redacted, got %q", entries[0].Params)
	}
}

func TestAuditConcurrentWrites(t *testing.T) {
	a := setupAuditLogger(t)
	ctx := context.Background()
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = a.LogAction(ctx, "sess-concurrent", pkg.Action{Type: "tool", ToolName: "test", Params: "data"}, pkg.DecisionAllow, "ok")
		}()
	}
	wg.Wait()
	a.Sync()

	entries, err := a.Query("sess-concurrent", 100)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(entries) != 20 {
		t.Errorf("expected 20 entries, got %d", len(entries))
	}
}

func TestAuditQueryByTool(t *testing.T) {
	a := setupAuditLogger(t)
	ctx := context.Background()

	_ = a.LogAction(ctx, "s1", pkg.Action{Type: "tool", ToolName: "bash"}, pkg.DecisionAllow, "ok")
	_ = a.LogAction(ctx, "s2", pkg.Action{Type: "tool", ToolName: "git"}, pkg.DecisionAllow, "ok")
	_ = a.LogAction(ctx, "s3", pkg.Action{Type: "tool", ToolName: "bash"}, pkg.DecisionAllow, "ok")
	a.Sync()

	entries, err := a.QueryByTool("bash", 10)
	if err != nil {
		t.Fatalf("QueryByTool failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 bash entries, got %d", len(entries))
	}

	entries, err = a.QueryByTool("git", 10)
	if err != nil {
		t.Fatalf("QueryByTool failed: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 git entry, got %d", len(entries))
	}
}

func TestAuditSanitizeParams(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"safe data", "safe data"},
		{"sk-abc123", "[REDACTED: potential API key]"},
		{`{"key":"sk-xyz"}`, "[REDACTED: potential API key]"},
	}
	for _, tt := range tests {
		got := sanitizeParams(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeParams(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAuditEmptyQuery(t *testing.T) {
	a := setupAuditLogger(t)
	entries, err := a.Query("nonexistent", 10)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestAuditTruncate(t *testing.T) {
	tests := []struct {
		input string
		n     int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello", 3, "hel"},
		{"", 10, ""},
	}
	for _, tt := range tests {
		got := truncate(tt.input, tt.n)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.n, got, tt.want)
		}
	}
}
