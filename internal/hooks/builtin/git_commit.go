package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/supcode/supcode/pkg"
)

// GitCommitHook is a post-tool-call hook that auto-commits file changes.
type GitCommitHook struct {
	enabled bool
}

// NewGitCommitHook creates a GitCommitHook.
func NewGitCommitHook(enabled bool) *GitCommitHook {
	return &GitCommitHook{enabled: enabled}
}

func (h *GitCommitHook) Name() string    { return "git_auto_commit" }
func (h *GitCommitHook) Enable()         { h.enabled = true }
func (h *GitCommitHook) Disable()        { h.enabled = false }
func (h *GitCommitHook) IsEnabled() bool { return h.enabled }

func (h *GitCommitHook) BeforeTool(ctx context.Context, toolName string, params json.RawMessage) (json.RawMessage, error) {
	return params, nil
}

// AfterTool auto-commits if the tool modified a file in a git repo.
func (h *GitCommitHook) AfterTool(ctx context.Context, toolName string, params json.RawMessage, result pkg.ToolResult) error {
	if !h.enabled || !result.Success {
		return nil
	}
	if toolName != "write_file" && toolName != "edit_file" {
		return nil
	}

	var fp struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(params, &fp); err != nil || fp.Path == "" {
		return nil
	}

	absPath, err := filepath.Abs(fp.Path)
	if err != nil {
		return nil
	}

	repoRoot, err := findGitRoot(filepath.Dir(absPath))
	if err != nil {
		return nil
	}

	relPath, err := filepath.Rel(repoRoot, absPath)
	if err != nil {
		return nil
	}

	addCmd := exec.CommandContext(ctx, "git", "-C", repoRoot, "add", relPath)
	if out, err := addCmd.CombinedOutput(); err != nil {
		log.Printf("[git_commit] git add failed: %v, output: %s", err, string(out))
		return nil
	}

	commitMsg := fmt.Sprintf("SupCode: %s %s", toolName, relPath)
	commitCmd := exec.CommandContext(ctx, "git", "-C", repoRoot, "commit", "-m", commitMsg)
	if out, err := commitCmd.CombinedOutput(); err != nil {
		log.Printf("[git_commit] git commit: %v, output: %s", err, string(out))
		return nil
	}

	return nil
}

// findGitRoot walks up from dir to find the git repository root.
func findGitRoot(dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	current := absDir
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("not a git repository")
		}
		current = parent
	}
}

var _ pkg.ToolHook = (*GitCommitHook)(nil)
