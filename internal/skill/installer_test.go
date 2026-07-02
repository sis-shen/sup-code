package skill

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSourceSkill(t *testing.T, dir, name string) {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	os.MkdirAll(filepath.Join(skillDir, "prompts"), 0755)
	manifest := `{"name":"` + name + `","version":"1.0.0","description":"test"}`
	os.WriteFile(filepath.Join(skillDir, "skill.json"), []byte(manifest), 0644)
	os.WriteFile(filepath.Join(skillDir, "prompts", "system.md"), []byte("prompt content"), 0644)
}

func TestInstallFromLocal(t *testing.T) {
	srcDir := t.TempDir()
	targetDir := t.TempDir()

	setupSourceSkill(t, srcDir, "my-skill")

	inst := NewInstaller(targetDir)
	err := inst.InstallFromLocal(filepath.Join(srcDir, "my-skill"))
	require.NoError(t, err)

	installedPath := filepath.Join(targetDir, "my-skill", "skill.json")
	_, err = os.Stat(installedPath)
	assert.NoError(t, err, "skill should be installed")

	promptPath := filepath.Join(targetDir, "my-skill", "prompts", "system.md")
	_, err = os.Stat(promptPath)
	assert.NoError(t, err, "prompts should be copied")
}

func TestUninstall(t *testing.T) {
	srcDir := t.TempDir()
	targetDir := t.TempDir()

	setupSourceSkill(t, srcDir, "to-remove")
	inst := NewInstaller(targetDir)
	err := inst.InstallFromLocal(filepath.Join(srcDir, "to-remove"))
	require.NoError(t, err)

	err = inst.Uninstall("to-remove")
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(targetDir, "to-remove"))
	assert.True(t, os.IsNotExist(err), "skill dir should be deleted")
}

func TestUninstallNonexistent(t *testing.T) {
	inst := NewInstaller(t.TempDir())
	err := inst.Uninstall("nonexistent")
	assert.Error(t, err)
}

func TestInstallFromLocalInvalid(t *testing.T) {
	inst := NewInstaller(t.TempDir())
	err := inst.InstallFromLocal("/nonexistent/path")
	assert.Error(t, err)
}

func TestListInstalled(t *testing.T) {
	src := t.TempDir()
	target := t.TempDir()
	setupSourceSkill(t, src, "s1")
	setupSourceSkill(t, src, "s2")

	inst := NewInstaller(target)
	inst.InstallFromLocal(filepath.Join(src, "s1"))
	inst.InstallFromLocal(filepath.Join(src, "s2"))

	infos, err := inst.ListInstalled()
	require.NoError(t, err)
	assert.Len(t, infos, 2)
}

func TestCopyDir(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	os.WriteFile(filepath.Join(src, "a.txt"), []byte("content"), 0644)
	os.MkdirAll(filepath.Join(src, "sub"), 0755)
	os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("nested"), 0644)

	err := copyDir(src, filepath.Join(dst, "copied"))
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dst, "copied", "a.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content", string(data))

	data, err = os.ReadFile(filepath.Join(dst, "copied", "sub", "b.txt"))
	require.NoError(t, err)
	assert.Equal(t, "nested", string(data))
}
