package permission

import (
	"path/filepath"
	"testing"
)

func setupFirstUseTracker(t *testing.T) *FirstUseTracker {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "first_use_test.db")
	f, err := NewFirstUseTracker(dbPath)
	if err != nil {
		t.Fatalf("NewFirstUseTracker failed: %v", err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestFirstUseInitiallyTrue(t *testing.T) {
	f := setupFirstUseTracker(t)
	if !f.IsFirstUse("git") {
		t.Error("expected IsFirstUse(git) = true for new tracker")
	}
}

func TestFirstUseAfterAuthorizeReturnsFalse(t *testing.T) {
	f := setupFirstUseTracker(t)
	if err := f.Authorize("bash"); err != nil {
		t.Fatalf("Authorize failed: %v", err)
	}
	if f.IsFirstUse("bash") {
		t.Error("expected IsFirstUse(bash) = false after Authorize")
	}
}

func TestFirstUsePersistenceAcrossSessions(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "persist_test.db")

	f1, err := NewFirstUseTracker(dbPath)
	if err != nil {
		t.Fatalf("NewFirstUseTracker failed: %v", err)
	}
	if err := f1.Authorize("persist-tool"); err != nil {
		t.Fatalf("Authorize failed: %v", err)
	}
	f1.Close()

	f2, err := NewFirstUseTracker(dbPath)
	if err != nil {
		t.Fatalf("NewFirstUseTracker failed: %v", err)
	}
	defer f2.Close()

	if f2.IsFirstUse("persist-tool") {
		t.Error("expected IsFirstUse to be false after re-opening the same db")
	}
}

func TestFirstUseAllAuthorized(t *testing.T) {
	f := setupFirstUseTracker(t)
	_ = f.Authorize("a")
	_ = f.Authorize("b")
	_ = f.Authorize("c")

	keys := f.AllAuthorized()
	if len(keys) != 3 {
		t.Errorf("expected 3 authorized tools, got %d: %v", len(keys), keys)
	}
}
