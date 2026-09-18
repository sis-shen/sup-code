package skill

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAllSkillsEmpty(t *testing.T) {
	err := ListAllSkills()
	require.NoError(t, err)
}

func TestInstallSkillViaInstaller(t *testing.T) {
	src := t.TempDir()
	target := t.TempDir()

	skillDir := filepath.Join(src, "myskill")
	require.NoError(t, os.MkdirAll(filepath.Join(skillDir, "prompts"), 0755))
	manifest := `{"name":"myskill","version":"1.0.0","description":"a skill"}`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "skill.json"), []byte(manifest), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "prompts", "system.md"), []byte("prompt"), 0644))

	inst := NewInstaller(target)
	err := inst.InstallFromLocal(skillDir)
	require.NoError(t, err)

	installed := filepath.Join(target, "myskill", "skill.json")
	_, err = os.Stat(installed)
	assert.NoError(t, err, "skill should be installed")
}

func TestInstallSkillInvalidSource(t *testing.T) {
	inst := NewInstaller(t.TempDir())
	err := inst.InstallFromLocal("/nonexistent/path")
	assert.Error(t, err)
}

func TestGetSkillDirsBasic(t *testing.T) {
	builtin, user, project := GetSkillDirs()
	assert.NotEmpty(t, builtin)
	assert.NotEmpty(t, user)
	assert.Empty(t, project)
}
