package skill

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestSkill(t *testing.T, dir, name string) {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Join(skillDir, "prompts"), 0755))
	manifest := `{"name":"` + name + `","version":"1.0.0","description":"test skill for ` + name + `"}`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "skill.json"), []byte(manifest), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "prompts", "system.md"), []byte("You are a `+name+` expert."), 0644))
}

func TestDiscoverSingle(t *testing.T) {
	dir := t.TempDir()
	setupTestSkill(t, dir, "my-skill")

	loader := NewSkillLoader(dir, "", "")
	infos, err := loader.Discover()
	require.NoError(t, err)
	assert.Len(t, infos, 1)
	assert.Equal(t, "my-skill", infos[0].Name)
}

func TestDiscoverEmpty(t *testing.T) {
	dir := t.TempDir()
	loader := NewSkillLoader(dir, "", "")
	infos, err := loader.Discover()
	require.NoError(t, err)
	assert.Empty(t, infos)
}

func TestDiscoverMultiple(t *testing.T) {
	dir := t.TempDir()
	setupTestSkill(t, dir, "skill-a")
	setupTestSkill(t, dir, "skill-b")

	loader := NewSkillLoader(dir, "", "")
	infos, err := loader.Discover()
	require.NoError(t, err)
	assert.Len(t, infos, 2)
}

func TestLoadSkill(t *testing.T) {
	dir := t.TempDir()
	setupTestSkill(t, dir, "my-skill")

	loader := NewSkillLoader(dir, "", "")
	skill, err := loader.Load("my-skill")
	require.NoError(t, err)
	require.NotNil(t, skill)
	assert.Equal(t, "my-skill", skill.Manifest.Name)
	assert.NotEmpty(t, skill.SystemPrompt)
}

func TestLoadNotFound(t *testing.T) {
	dir := t.TempDir()
	loader := NewSkillLoader(dir, "", "")
	_, err := loader.Load("nonexistent")
	assert.Error(t, err)
}

func TestLoadInvalidManifest(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "broken")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "skill.json"), []byte(`{invalid`), 0644))

	loader := NewSkillLoader(dir, "", "")
	_, err := loader.Load("broken")
	assert.Error(t, err)
}

func TestPriorityProjectOverridesBuiltin(t *testing.T) {
	builtin := t.TempDir()
	project := t.TempDir()

	setupTestSkill(t, builtin, "common")
	// Project overrides with different description
	projDir := filepath.Join(project, "common")
	require.NoError(t, os.MkdirAll(projDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projDir, "skill.json"), []byte(`{"name":"common","version":"2.0.0","description":"project version"}`), 0644))

	loader := NewSkillLoader(builtin, "", project)
	infos, err := loader.Discover()
	require.NoError(t, err)
	assert.Len(t, infos, 1)
	assert.Equal(t, "project version", infos[0].Description)
	assert.Equal(t, 2, infos[0].SourceLevel)
}

func TestLoadWithCache(t *testing.T) {
	dir := t.TempDir()
	setupTestSkill(t, dir, "cached-skill")

	loader := NewSkillLoader(dir, "", "")
	skill1, err := loader.Load("cached-skill")
	require.NoError(t, err)

	skill2, err := loader.Load("cached-skill")
	require.NoError(t, err)
	assert.Equal(t, skill1, skill2)
}

func TestInvalidateCache(t *testing.T) {
	dir := t.TempDir()
	setupTestSkill(t, dir, "skill")

	loader := NewSkillLoader(dir, "", "")
	_, err := loader.Load("skill")
	require.NoError(t, err)

	loader.InvalidateCache("skill")
	_, err = loader.Load("skill")
	require.NoError(t, err)
}

func TestInvalidateAll(t *testing.T) {
	dir := t.TempDir()
	setupTestSkill(t, dir, "s1")
	setupTestSkill(t, dir, "s2")

	loader := NewSkillLoader(dir, "", "")
	_, err := loader.Load("s1")
	require.NoError(t, err)
	_, err = loader.Load("s2")
	require.NoError(t, err)
	loader.InvalidateAll()

	infos, err := loader.Discover()
	require.NoError(t, err)
	assert.Len(t, infos, 2)
}
