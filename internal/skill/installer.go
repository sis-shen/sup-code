package skill

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Installer struct {
	TargetDir   string
	execCommand func(command string, args ...string) *exec.Cmd
}

func NewInstaller(targetDir string) *Installer {
	return &Installer{TargetDir: targetDir, execCommand: exec.Command}
}

func (inst *Installer) InstallFromLocal(srcPath string) error {
	manifestPath := filepath.Join(srcPath, "skill.json")
	m, err := LoadManifestFile(manifestPath)
	if err != nil {
		return fmt.Errorf("load source manifest: %w", err)
	}
	target := filepath.Join(inst.TargetDir, m.Name)
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("clean target: %w", err)
	}
	if err := copyDir(srcPath, target); err != nil {
		return fmt.Errorf("copy skill: %w", err)
	}
	return nil
}

func (inst *Installer) InstallFromGit(repoURL string) error {
	tmpDir, err := os.MkdirTemp("", "skill-git-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cmd := inst.execCommand("git", "clone", repoURL, tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %w\nOutput: %s", err, string(output))
	}
	return inst.InstallFromLocal(tmpDir)
}

func (inst *Installer) InstallFromNPM(packageName string) error {
	tmpDir, err := os.MkdirTemp("", "skill-npm-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cmd := inst.execCommand("npm", "pack", packageName, "--pack-destination", tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("npm pack failed: %w\nOutput: %s", err, string(output))
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return fmt.Errorf("read temp dir: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("no package file found from npm pack")
	}

	tarPath := filepath.Join(tmpDir, entries[0].Name())
	extractDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("create extract dir: %w", err)
	}

	cmd = inst.execCommand("tar", "-xzf", tarPath, "-C", extractDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("extract failed: %w\nOutput: %s", err, string(output))
	}

	pkgDir := filepath.Join(extractDir, "package")
	if _, err := os.Stat(pkgDir); err == nil {
		return inst.InstallFromLocal(pkgDir)
	}
	return inst.InstallFromLocal(extractDir)
}

func (inst *Installer) Uninstall(name string) error {
	target := filepath.Join(inst.TargetDir, name)
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return fmt.Errorf("skill not installed: %s", name)
	}
	return os.RemoveAll(target)
}

func (inst *Installer) ListInstalled() ([]SkillInfo, error) {
	loader := NewSkillLoader(inst.TargetDir, "", "")
	return loader.Discover()
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}
