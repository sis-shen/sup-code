package worktree

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	allArgs := append([]string{"-C", dir}, args...)
	cmd := exec.Command("git", allArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %s\n%s", args, err, string(out))
	}
	return strings.TrimSpace(string(out))
}

func setupTestRepo(t *testing.T) (string, string) {
	t.Helper()
	repoDir := t.TempDir()
	baseDir := filepath.Join(t.TempDir(), "wt")

	runGit(t, repoDir, "init")
	runGit(t, repoDir, "config", "user.name", "test")
	runGit(t, repoDir, "config", "user.email", "test@test.com")
	runGit(t, repoDir, "branch", "-m", "main")

	readme := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoDir, "add", "README.md")
	runGit(t, repoDir, "commit", "-m", "initial commit")

	return repoDir, baseDir
}

func setupManager(t *testing.T) (*Manager, string) {
	t.Helper()
	repoDir, baseDir := setupTestRepo(t)
	m, err := NewManagerWithBase(repoDir, baseDir)
	if err != nil {
		t.Fatalf("NewManagerWithBase failed: %v", err)
	}
	return m, repoDir
}

func TestCreateWorktree(t *testing.T) {
	m, _ := setupManager(t)
	ctx := context.Background()

	path, err := m.Create(ctx, "agent-1", "main")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("worktree path %s does not exist", path)
	}

	// In worktrees, .git is a file containing "gitdir: <path>"
	gitPath := filepath.Join(path, ".git")
	if _, err := os.Stat(gitPath); os.IsNotExist(err) {
		t.Errorf("worktree %s is not a valid git worktree (no .git)", path)
	}
}

func TestCreateAndMerge(t *testing.T) {
	m, repoDir := setupManager(t)
	ctx := context.Background()

	path, err := m.Create(ctx, "agent-merge", "main")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	newFile := filepath.Join(path, "new_file.txt")
	if err := os.WriteFile(newFile, []byte("agent work\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := m.Merge(ctx, "agent-merge"); err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	mergedFile := filepath.Join(repoDir, "new_file.txt")
	if _, err := os.Stat(mergedFile); os.IsNotExist(err) {
		t.Errorf("merged file not found in main repo: %s", mergedFile)
	}
}

func TestMergeConflict(t *testing.T) {
	m, repoDir := setupManager(t)
	ctx := context.Background()

	// Create two worktrees from the same base BEFORE either merges
	path1, err := m.Create(ctx, "agent-conflict-a", "main")
	if err != nil {
		t.Fatalf("Create for agent A failed: %v", err)
	}

	path2, err := m.Create(ctx, "agent-conflict-b", "main")
	if err != nil {
		t.Fatalf("Create for agent B failed: %v", err)
	}

	// Both write to the same file with different content
	fileA := filepath.Join(path1, "conflict_file.txt")
	if err := os.WriteFile(fileA, []byte("AAA content\n"), 0644); err != nil {
		t.Fatal(err)
	}

	fileB := filepath.Join(path2, "conflict_file.txt")
	if err := os.WriteFile(fileB, []byte("BBB content\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// First merge succeeds
	if err := m.Merge(ctx, "agent-conflict-a"); err != nil {
		t.Fatalf("First merge (A) should succeed: %v", err)
	}

	// Second merge should conflict
	err = m.Merge(ctx, "agent-conflict-b")
	if err == nil {
		t.Fatal("expected merge conflict error, got nil")
	}

	var conflictErr *MergeConflictError
	if !asConflictError(err, &conflictErr) {
		t.Fatalf("expected *MergeConflictError, got %T: %v", err, err)
	}
	if len(conflictErr.Files) == 0 {
		t.Errorf("expected conflict files list to be non-empty, got %v", conflictErr.Files)
	}
	t.Logf("conflict error: %v (files: %v)", err, conflictErr.Files)

	_ = exec.CommandContext(ctx, "git", "-C", repoDir, "merge", "--abort").Run()
}

func asConflictError(err error, target **MergeConflictError) bool {
	*target = nil
	for e := err; e != nil; {
		if mce, ok := e.(*MergeConflictError); ok {
			*target = mce
			return true
		}
		if unwrapper, ok := e.(interface{ Unwrap() error }); ok {
			e = unwrapper.Unwrap()
		} else {
			return false
		}
	}
	return false
}

func TestAbandon(t *testing.T) {
	m, _ := setupManager(t)
	ctx := context.Background()

	path, err := m.Create(ctx, "agent-abandon", "main")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := m.Abandon(ctx, "agent-abandon"); err != nil {
		t.Fatalf("Abandon failed: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("worktree path %s should not exist after abandon", path)
	}
}

func TestListActive(t *testing.T) {
	m, _ := setupManager(t)
	ctx := context.Background()

	_, err := m.Create(ctx, "agent-list-1", "main")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = m.Create(ctx, "agent-list-2", "main")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := m.Abandon(ctx, "agent-list-1"); err != nil {
		t.Fatalf("Abandon failed: %v", err)
	}

	active, err := m.ListActive(ctx)
	if err != nil {
		t.Fatalf("ListActive failed: %v", err)
	}
	if len(active) != 1 {
		t.Errorf("expected 1 active worktree, got %d: %+v", len(active), active)
	}
	if len(active) > 0 && active[0].AgentID != "agent-list-2" {
		t.Errorf("expected active agent 'agent-list-2', got %q", active[0].AgentID)
	}
}

func TestCleanup(t *testing.T) {
	m, _ := setupManager(t)
	ctx := context.Background()

	_, err := m.Create(ctx, "agent-cleanup-1", "main")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = m.Create(ctx, "agent-cleanup-2", "main")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	_ = m.Abandon(ctx, "agent-cleanup-1")
	_ = m.Abandon(ctx, "agent-cleanup-2")

	if err := m.Cleanup(ctx); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	active, err := m.ListActive(ctx)
	if err != nil {
		t.Fatalf("ListActive failed: %v", err)
	}
	if len(active) != 0 {
		t.Errorf("expected 0 active worktrees after cleanup, got %d", len(active))
	}
}

func TestNonGitRepo(t *testing.T) {
	nonGitDir := t.TempDir()
	_, err := NewManagerWithBase(nonGitDir, t.TempDir())
	if err == nil {
		t.Fatal("expected error for non-git repo, got nil")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("expected 'not a git repository' error, got: %v", err)
	}
}

func TestConcurrentCreate(t *testing.T) {
	m, _ := setupManager(t)
	ctx := context.Background()
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			agentID := "agent-concurrent-" + strings.TrimRight(string(rune('0'+id)), "\x00")
			_, err := m.Create(ctx, agentID, "main")
			if err != nil && !strings.Contains(err.Error(), "already exists") {
				t.Errorf("concurrent Create failed: %v", err)
			}
		}(i)
	}
	wg.Wait()
}

func TestCreateDuplicate(t *testing.T) {
	m, _ := setupManager(t)
	ctx := context.Background()

	_, err := m.Create(ctx, "agent-duplicate", "main")
	if err != nil {
		t.Fatalf("First Create failed: %v", err)
	}

	_, err = m.Create(ctx, "agent-duplicate", "main")
	if err == nil {
		t.Fatal("expected error for duplicate create, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestMergeWithNoChanges(t *testing.T) {
	m, _ := setupManager(t)
	ctx := context.Background()

	_, err := m.Create(ctx, "agent-nochange", "main")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := m.Merge(ctx, "agent-nochange"); err != nil {
		t.Fatalf("Merge with no changes should succeed: %v", err)
	}
}

func TestAbandonNonexistent(t *testing.T) {
	m, _ := setupManager(t)
	ctx := context.Background()

	err := m.Abandon(ctx, "nonexistent-agent")
	if err == nil {
		t.Fatal("expected error for abandoning nonexistent agent, got nil")
	}
}

func TestNewManagerInvalidPath(t *testing.T) {
	_, err := NewManager("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
}

func TestWorktreeInfoHasTimestamp(t *testing.T) {
	m, _ := setupManager(t)
	ctx := context.Background()

	before := time.Now().UTC()
	_, err := m.Create(ctx, "agent-ts", "main")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	after := time.Now().UTC()

	active, err := m.ListActive(ctx)
	if err != nil {
		t.Fatalf("ListActive failed: %v", err)
	}

	var found bool
	for _, info := range active {
		if info.AgentID == "agent-ts" {
			found = true
			if info.CreatedAt.Before(before) || info.CreatedAt.After(after) {
				t.Errorf("CreatedAt %v outside expected range [%v, %v]", info.CreatedAt, before, after)
			}
			if info.Status != "active" {
				t.Errorf("Status = %q, want 'active'", info.Status)
			}
			if info.Path == "" {
				t.Error("Path should not be empty")
			}
			if info.Branch != "codex-agent-ts" {
				t.Errorf("Branch = %q, want 'codex-agent-ts'", info.Branch)
			}
		}
	}
	if !found {
		t.Error("agent-ts not found in ListActive")
	}
}

func TestWorktreeBranchNaming(t *testing.T) {
	m, repoDir := setupManager(t)
	ctx := context.Background()

	_, err := m.Create(ctx, "special-agent", "main")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	out := runGit(t, repoDir, "branch", "--list", "codex-special-agent")
	if !strings.Contains(out, "codex-special-agent") {
		t.Errorf("branch codex-special-agent not found in repo: %s", out)
	}
}
func TestNewManager(t *testing.T) {
	repoDir, _ := setupTestRepo(t)
	m, err := NewManager(repoDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.mainRepoPath == "" {
		t.Error("mainRepoPath should not be empty")
	}
}
