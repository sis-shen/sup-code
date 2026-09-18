package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
)

func GetSkillDirs() (builtin, user, project string) {
	builtin = "skills"
	if home, err := os.UserHomeDir(); err == nil {
		user = filepath.Join(home, ".supcode", "skills")
	}
	if cwd, err := os.Getwd(); err == nil {
		for i := 0; i < 10; i++ {
			candidate := filepath.Join(cwd, ".supcode", "skills")
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				project = candidate
				break
			}
			parent := filepath.Dir(cwd)
			if parent == cwd {
				break
			}
			cwd = parent
		}
	}
	return
}

func ListAllSkills() error {
	builtin, user, project := GetSkillDirs()
	loader := NewSkillLoader(builtin, user, project)

	infos, err := loader.Discover()
	if err != nil {
		return fmt.Errorf("discover skills: %w", err)
	}

	if len(infos) == 0 {
		fmt.Println("No skills installed.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if _, err := fmt.Fprintln(w, "Name\tVersion\tDescription\tSource"); err != nil {
		return err
	}
	for _, info := range infos {
		source := "builtin"
		switch info.SourceLevel {
		case 1:
			source = "user"
		case 2:
			source = "project"
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", info.Name, info.Version, info.Description, source); err != nil {
			return err
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return nil
}

func InstallSkill(source string) error {
	_, _, user := GetSkillDirs()
	if user == "" {
		return fmt.Errorf("cannot determine user skill directory")
	}

	if err := os.MkdirAll(user, 0755); err != nil {
		return fmt.Errorf("create skill dir: %w", err)
	}

	inst := NewInstaller(user)

	// Determine install method based on source path
	if _, err := os.Stat(source); err == nil {
		return inst.InstallFromLocal(source)
	}
	return fmt.Errorf("unknown source: %s", source)
}
