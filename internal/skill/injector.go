package skill

import (
	"fmt"
	"sort"
	"strings"
)

const skillSeparator = "\n---\n"

func InjectSkills(basePrompt string, skills []*Skill) string {
	if len(skills) == 0 {
		return basePrompt
	}

	var b strings.Builder
	b.WriteString(basePrompt)
	b.WriteString("\n\n## Active Skills\n\n")

	// Deduplicate by name, keep highest source level
	seen := make(map[string]*Skill)
	for _, s := range skills {
		if existing, ok := seen[s.Manifest.Name]; ok {
			if s.SourceLevel > existing.SourceLevel {
				seen[s.Manifest.Name] = s
			}
		} else {
			seen[s.Manifest.Name] = s
		}
	}

	// Sort by name for deterministic output
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)

	for i, name := range names {
		s := seen[name]
		if i > 0 {
			b.WriteString(skillSeparator)
		}
		b.WriteString(fmt.Sprintf("### %s (%s)\n", s.Manifest.Name, s.Manifest.Version))
		if s.Manifest.Description != "" {
			b.WriteString(fmt.Sprintf("%s\n\n", s.Manifest.Description))
		}
		if s.SystemPrompt != "" {
			b.WriteString(s.SystemPrompt)
		}
	}

	return b.String()
}

func SelectActiveSkills(allSkills []*Skill, names []string) []*Skill {
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}

	var active []*Skill
	for _, s := range allSkills {
		if nameSet[s.Manifest.Name] {
			active = append(active, s)
		}
		if len(active) == len(names) {
			break
		}
	}
	return active
}
