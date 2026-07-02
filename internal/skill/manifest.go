package skill

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var namePattern = regexp.MustCompile(`^[a-z0-9-]+$`)

type SkillManifest struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Author      string   `json:"author,omitempty"`
	Tools       []string `json:"tools,omitempty"`
	Requires    []string `json:"requires,omitempty"`
}

type Skill struct {
	Manifest     SkillManifest
	SystemPrompt string
	SourcePath   string
	SourceLevel  int // 0=builtin, 1=user, 2=project
}

func ParseManifest(data []byte) (*SkillManifest, error) {
	var m SkillManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

func LoadManifestFile(path string) (*SkillManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	return ParseManifest(data)
}

func (m *SkillManifest) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("skill name is required")
	}
	if !namePattern.MatchString(m.Name) {
		return fmt.Errorf("skill name %q must match [a-z0-9-]", m.Name)
	}
	if m.Version == "" {
		return fmt.Errorf("skill version is required")
	}
	if !isValidSemver(m.Version) {
		return fmt.Errorf("skill version %q is not valid semver", m.Version)
	}
	if m.Description == "" {
		return fmt.Errorf("skill description is required")
	}
	return nil
}

func isValidSemver(v string) bool {
	if v == "" {
		return false
	}
	// Accepts: major.minor.patch, with optional prerelease/build metadata
	// Simple check: starts with digit, contains at least one dot
	parts := strings.Split(v, ".")
	if len(parts) < 2 {
		return false
	}
	for i, p := range parts {
		// Check for pre-release/build suffix on last part
		check := p
		if i == len(parts)-1 {
			check = strings.SplitN(p, "-", 2)[0]
			check = strings.SplitN(check, "+", 2)[0]
		}
		if len(check) == 0 {
			return false
		}
		for _, c := range check {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}
