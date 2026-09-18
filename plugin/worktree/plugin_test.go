package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/core"
	"github.com/supcode/supcode/pkg"
)

// runGit runs a git command in dir, failing the test on error.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	allArgs := append([]string{"-C", dir}, args...)
	cmd := exec.Command("git", allArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %s\n%s", args, err, string(out))
	}
}

// setupRepo creates a git repository with a single commit, mirroring
// internal/worktree's test setup so NewManager succeeds.
func setupRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.name", "test")
	runGit(t, dir, "config", "user.email", "test@test.com")

	readme := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readme, []byte("# test\n"), 0644))
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "initial commit")

	return dir
}

func TestPluginProvidesWorktreeManager(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	repo := setupRepo(t)

	t.Run("default base dir", func(t *testing.T) {
		k := core.New()
		require.NoError(t, k.Load(k.Root(), Plugin(Options{RepoPath: repo})))

		mgr := core.Use[pkg.WorktreeManager](k.Root(), pkg.ServiceWorktree)
		require.NotNil(t, mgr)
		assert.Equal(t, []string{pluginName}, k.Plugins())
	})

	t.Run("custom base dir", func(t *testing.T) {
		base := filepath.Join(t.TempDir(), "wt")
		k := core.New()
		require.NoError(t, k.Load(k.Root(), Plugin(Options{RepoPath: repo, BaseDir: base})))

		mgr := core.Use[pkg.WorktreeManager](k.Root(), pkg.ServiceWorktree)
		require.NotNil(t, mgr)
	})
}

func TestPluginEmptyRepoPath(t *testing.T) {
	k := core.New()
	err := k.Load(k.Root(), Plugin(Options{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RepoPath")
}

func TestPluginNonGitRepo(t *testing.T) {
	k := core.New()
	err := k.Load(k.Root(), Plugin(Options{RepoPath: t.TempDir()}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")
}

func TestManifestMatchesPlugin(t *testing.T) {
	m := Manifest()
	assert.Equal(t, pluginName, m.Name)
	assert.Equal(t, []string{"worktree"}, m.Provides)
	assert.Equal(t, "builtin:worktree", m.Entry)

	p := Plugin(Options{})
	assert.Equal(t, m.Name, p.Name)
	assert.Equal(t, m.Provides, p.Provides)
	assert.Empty(t, p.Inject)
}
