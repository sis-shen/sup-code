package skill

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeExecCommandWriteToLastArg(files map[string]string) func(string, ...string) *exec.Cmd {
	return func(cmd string, args ...string) *exec.Cmd {
		if len(args) == 0 {
			return exec.Command("cmd", "/c", "type", "NUL")
		}
		outputDir := args[len(args)-1]
		for relPath, content := range files {
			fullPath := filepath.Join(outputDir, relPath)
			_ = os.MkdirAll(filepath.Dir(fullPath), 0755)
			_ = os.WriteFile(fullPath, []byte(content), 0644)
		}
		return exec.Command("cmd", "/c", "type", "NUL")
	}
}

func fakeExecCommandFail(cmd string, args ...string) *exec.Cmd {
	return exec.Command("cmd", "/c", "exit", "1")
}

func TestInstallFromGit_Success(t *testing.T) {
	target := t.TempDir()
	inst := NewInstaller(target)
	inst.execCommand = fakeExecCommandWriteToLastArg(map[string]string{
		"skill.json":        `{"name":"git-skill","version":"1.0.0","description":"from git"}`,
		"prompts/system.md": "git prompt content",
	})
	err := inst.InstallFromGit("https://github.com/fake/repo.git")
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(target, "git-skill", "skill.json"))
	assert.NoError(t, err)
}

func TestInstallFromGit_Failure(t *testing.T) {
	inst := NewInstaller(t.TempDir())
	inst.execCommand = fakeExecCommandFail
	err := inst.InstallFromGit("https://github.com/fake/repo.git")
	assert.Error(t, err)
}

func TestInstallFromNPM_Success(t *testing.T) {
	target := t.TempDir()
	inst := NewInstaller(target)
	inst.execCommand = func(cmd string, args ...string) *exec.Cmd {
		outputDir := args[len(args)-1]
		switch cmd {
		case "npm":
			tarFile := filepath.Join(outputDir, "package-1.0.0.tgz")
			_ = os.WriteFile(tarFile, []byte("fake"), 0644)
		case "tar":
			pkgDir := filepath.Join(outputDir, "package")
			_ = os.MkdirAll(filepath.Join(pkgDir, "prompts"), 0755)
			manifest := `{"name":"npm-skill","version":"1.0.0","description":"from npm"}`
			_ = os.WriteFile(filepath.Join(pkgDir, "skill.json"), []byte(manifest), 0644)
			_ = os.WriteFile(filepath.Join(pkgDir, "prompts", "system.md"), []byte("prompt"), 0644)
		}
		return exec.Command("cmd", "/c", "type", "NUL")
	}
	err := inst.InstallFromNPM("fake-npm-package")
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(target, "npm-skill", "skill.json"))
	assert.NoError(t, err)
}

func TestInstallFromNPM_Failure(t *testing.T) {
	inst := NewInstaller(t.TempDir())
	inst.execCommand = fakeExecCommandFail
	err := inst.InstallFromNPM("fake-npm-package")
	assert.Error(t, err)
}
