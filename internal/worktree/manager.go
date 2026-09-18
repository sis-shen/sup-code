package worktree

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/supcode/supcode/pkg"
)

type worktreeEntry struct {
	AgentID   string    `json:"agent_id"`
	Path      string    `json:"path"`
	Branch    string    `json:"branch"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}

const (
	worktreesDir  = "supcode-worktrees"
	metadataFile  = ".worktrees.json"
	defaultBranch = "main"
)

type Manager struct {
	mainRepoPath string
	baseDir      string
	mu           sync.Mutex
}

func NewManager(mainRepoPath string) (*Manager, error) {
	absPath, err := filepath.Abs(mainRepoPath)
	if err != nil {
		return nil, fmt.Errorf("resolve repo path: %w", err)
	}
	if err := verifyGitRepo(absPath); err != nil {
		return nil, err
	}
	baseDir := filepath.Join(os.TempDir(), worktreesDir)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create worktree base dir: %w", err)
	}
	return &Manager{mainRepoPath: absPath, baseDir: baseDir}, nil
}

func NewManagerWithBase(mainRepoPath, baseDir string) (*Manager, error) {
	absPath, err := filepath.Abs(mainRepoPath)
	if err != nil {
		return nil, fmt.Errorf("resolve repo path: %w", err)
	}
	if err := verifyGitRepo(absPath); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create worktree base dir: %w", err)
	}
	return &Manager{mainRepoPath: absPath, baseDir: baseDir}, nil
}

func verifyGitRepo(path string) error {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("not a git repository: %s", path)
		}
		return fmt.Errorf("check git repo: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a git repository: %s (.git is not a directory)", path)
	}
	return nil
}

func (m *Manager) Create(ctx context.Context, agentID string, baseBranch string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if agentID == "" {
		return "", fmt.Errorf("agentID must not be empty")
	}
	if baseBranch == "" {
		baseBranch = defaultBranch
	}

	worktreePath := filepath.Join(m.baseDir, agentID)
	branchName := "codex-" + agentID

	if _, err := os.Stat(worktreePath); err == nil {
		return "", fmt.Errorf("worktree already exists for agent %s at %s", agentID, worktreePath)
	}

	cmd := exec.CommandContext(ctx, "git",
		"-C", m.mainRepoPath,
		"worktree", "add", "-b", branchName, worktreePath, baseBranch,
	)
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git worktree add failed: %s: %w", strings.TrimSpace(stderr.String()), err)
	}

	entry := worktreeEntry{
		AgentID:   agentID,
		Path:      worktreePath,
		Branch:    branchName,
		CreatedAt: time.Now().UTC(),
		Status:    "active",
	}
	if err := m.saveEntry(entry); err != nil {
		_ = m.removeWorktree(worktreePath)
		return "", fmt.Errorf("save worktree metadata: %w", err)
	}
	return worktreePath, nil
}

func (m *Manager) Merge(ctx context.Context, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, err := m.loadEntry(agentID)
	if err != nil {
		return fmt.Errorf("load worktree entry: %w", err)
	}
	if entry.Status != "active" {
		return fmt.Errorf("worktree for agent %s is not active (status: %s)", agentID, entry.Status)
	}

	commitMsg := fmt.Sprintf("supcode: agent %s changes", agentID)

	addCmd := exec.CommandContext(ctx, "git", "-C", entry.Path, "add", "-A")
	if out, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %s: %w", strings.TrimSpace(string(out)), err)
	}

	diffCmd := exec.CommandContext(ctx, "git", "-C", entry.Path, "diff", "--cached", "--quiet")
	if diffCmd.Run() != nil {
		commitCmd := exec.CommandContext(ctx, "git", "-C", entry.Path, "commit", "-m", commitMsg)
		if out, err := commitCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git commit failed: %s: %w", strings.TrimSpace(string(out)), err)
		}
	}

	currentBranch, err := m.getCurrentBranch(ctx)
	if err != nil {
		return fmt.Errorf("get current branch: %w", err)
	}

	mergeCmd := exec.CommandContext(ctx, "git", "-C", m.mainRepoPath, "merge", "--no-edit", entry.Branch)
	mergeOut := &bytes.Buffer{}
	mergeCmd.Stdout = mergeOut
	mergeCmd.Stderr = mergeOut
	if err := mergeCmd.Run(); err != nil {
		conflictFiles := extractConflicts(mergeOut.String())
		_ = exec.CommandContext(ctx, "git", "-C", m.mainRepoPath, "merge", "--abort").Run()
		return &MergeConflictError{
			AgentID:      agentID,
			Branch:       entry.Branch,
			TargetBranch: currentBranch,
			Files:        conflictFiles,
			Cause:        err,
		}
	}

	entry.Status = "merged"
	if err := m.saveEntry(entry); err != nil {
		return fmt.Errorf("update worktree metadata: %w", err)
	}
	return nil
}

type MergeConflictError struct {
	AgentID      string
	Branch       string
	TargetBranch string
	Files        []string
	Cause        error
}

func (e *MergeConflictError) Error() string {
	return fmt.Sprintf("merge conflict: worktree %s (%s) into %s, files: %v", e.AgentID, e.Branch, e.TargetBranch, e.Files)
}

func (e *MergeConflictError) Unwrap() error {
	return e.Cause
}

func extractConflicts(output string) []string {
	var files []string
	for _, line := range strings.Split(output, "\n") {
		if !strings.Contains(line, "CONFLICT") {
			continue
		}
		// Format: CONFLICT (content): Merge conflict in <file>
		idx := strings.Index(line, "Merge conflict in ")
		if idx >= 0 {
			f := strings.TrimSpace(line[idx+len("Merge conflict in "):])
			if f != "" {
				files = append(files, f)
			}
			continue
		}
		// Also try: CONFLICT in <file> (older format)
		idx = strings.Index(line, "CONFLICT in ")
		if idx >= 0 {
			f := strings.TrimSpace(line[idx+len("CONFLICT in "):])
			if f != "" {
				files = append(files, f)
			}
		}
	}
	seen := make(map[string]bool)
	var unique []string
	for _, f := range files {
		if !seen[f] {
			seen[f] = true
			unique = append(unique, f)
		}
	}
	return unique
}

func (m *Manager) Abandon(ctx context.Context, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, err := m.loadEntry(agentID)
	if err != nil {
		return fmt.Errorf("load worktree entry: %w", err)
	}
	if err := m.removeWorktree(entry.Path); err != nil {
		return err
	}
	entry.Status = "abandoned"
	if err := m.saveEntry(entry); err != nil {
		return fmt.Errorf("update worktree metadata: %w", err)
	}
	return nil
}

func (m *Manager) removeWorktree(path string) error {
	cmd := exec.Command("git", "worktree", "remove", "--force", path)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	if removeErr := os.RemoveAll(path); removeErr != nil {
		return fmt.Errorf("worktree remove failed (git: %s, os: %v)", strings.TrimSpace(string(out)), removeErr)
	}
	return nil
}

func (m *Manager) ListActive(ctx context.Context) ([]pkg.WorktreeInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entries, err := m.loadAllEntries()
	if err != nil {
		return nil, err
	}

	var result []pkg.WorktreeInfo
	for _, e := range entries {
		if e.Status == "active" {
			result = append(result, pkg.WorktreeInfo{
				AgentID:   e.AgentID,
				Path:      e.Path,
				Branch:    e.Branch,
				CreatedAt: e.CreatedAt,
				Status:    "active",
			})
		}
	}
	if result == nil {
		result = []pkg.WorktreeInfo{}
	}
	return result, nil
}

func (m *Manager) Cleanup(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entries, err := m.loadAllEntries()
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Status == "abandoned" {
			_ = m.removeWorktree(e.Path)
		}
	}
	return nil
}

func (m *Manager) metadataPath() string {
	return filepath.Join(m.baseDir, metadataFile)
}

func (m *Manager) loadAllEntries() ([]worktreeEntry, error) {
	data, err := os.ReadFile(m.metadataPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read worktree metadata: %w", err)
	}
	var entries []worktreeEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse worktree metadata: %w", err)
	}
	return entries, nil
}

func (m *Manager) loadEntry(agentID string) (worktreeEntry, error) {
	entries, err := m.loadAllEntries()
	if err != nil {
		return worktreeEntry{}, err
	}
	for _, e := range entries {
		if e.AgentID == agentID {
			return e, nil
		}
	}
	return worktreeEntry{}, fmt.Errorf("worktree entry not found for agent %s", agentID)
}

func (m *Manager) saveEntry(entry worktreeEntry) error {
	entries, err := m.loadAllEntries()
	if err != nil {
		entries = nil
	}
	found := false
	for i, e := range entries {
		if e.AgentID == entry.AgentID {
			entries[i] = entry
			found = true
			break
		}
	}
	if !found {
		entries = append(entries, entry)
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal worktree metadata: %w", err)
	}
	return os.WriteFile(m.metadataPath(), data, 0644)
}

func (m *Manager) getCurrentBranch(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", m.mainRepoPath, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("get current branch: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

var _ pkg.WorktreeManager = (*Manager)(nil)
