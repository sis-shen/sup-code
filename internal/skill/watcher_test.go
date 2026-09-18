package skill

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWatcher(t *testing.T) {
	dir := t.TempDir()
	loader := NewSkillLoader(dir, "", "")

	w, err := NewWatcher(loader, []string{dir})
	require.NoError(t, err)
	require.NotNil(t, w)
	defer func() { _ = w.Close() }()
}

func TestDetectSkillName(t *testing.T) {
	dir := t.TempDir()
	loader := NewSkillLoader(dir, "", "")

	w, err := NewWatcher(loader, []string{dir})
	require.NoError(t, err)
	defer func() { _ = w.Close() }()

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "my-skill", "prompts"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "my-skill", "skill.json"), []byte("{}"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "my-skill", "prompts", "system.md"), []byte("content"), 0644))

	name := w.detectSkillName(filepath.Join(dir, "my-skill", "skill.json"))
	assert.Equal(t, "my-skill", name)

	name = w.detectSkillName(filepath.Join(dir, "my-skill", "prompts", "system.md"))
	assert.Equal(t, "my-skill", name)

	name = w.detectSkillName(filepath.Join(dir, "other", "file.txt"))
	assert.Equal(t, "other", name)
}

func TestDetectSkillNameOutsideWatch(t *testing.T) {
	dir := t.TempDir()
	loader := NewSkillLoader(dir, "", "")

	w, err := NewWatcher(loader, []string{dir})
	require.NoError(t, err)
	defer func() { _ = w.Close() }()

	name := w.detectSkillName("/some/other/path/file.txt")
	assert.Empty(t, name)
}

func TestWatcherClose(t *testing.T) {
	dir := t.TempDir()
	loader := NewSkillLoader(dir, "", "")

	w, err := NewWatcher(loader, []string{dir})
	require.NoError(t, err)

	err = w.Close()
	assert.NoError(t, err)
}

func TestWatcherStartDoesNotPanic(t *testing.T) {
	dir := t.TempDir()
	loader := NewSkillLoader(dir, "", "")

	w, err := NewWatcher(loader, []string{dir})
	require.NoError(t, err)

	w.Start()
	_ = w.Close()
}

func TestWatcherHandleEvent(t *testing.T) {
	dir := t.TempDir()
	loader := NewSkillLoader(dir, "", "")

	w, err := NewWatcher(loader, []string{dir})
	require.NoError(t, err)
	defer func() { _ = w.Close() }()

	w.Start()

	// Create a skill directory
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "test-skill", "prompts"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test-skill", "skill.json"), []byte(`{"name":"test-skill","version":"1.0.0","description":"test"}`), 0644))

	// Load it once to populate cache
	_, err = loader.Load("test-skill")
	require.NoError(t, err)

	// Modify the skill file - this should trigger cache invalidation via watcher
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test-skill", "prompts", "system.md"), []byte("updated content"), 0644))
}
