package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type SkillLoader struct {
	BuiltinDir string
	UserDir    string
	ProjectDir string
	mu         sync.RWMutex
	cache      map[string]*Skill
}

func NewSkillLoader(builtinDir, userDir, projectDir string) *SkillLoader {
	return &SkillLoader{
		BuiltinDir: builtinDir,
		UserDir:    userDir,
		ProjectDir: projectDir,
		cache:      make(map[string]*Skill),
	}
}

func (l *SkillLoader) Discover() ([]SkillInfo, error) {
	var infos []SkillInfo
	paths := []struct {
		dir   string
		level int
	}{
		{l.BuiltinDir, 0},
		{l.UserDir, 1},
		{l.ProjectDir, 2},
	}

	for _, p := range paths {
		if p.dir == "" {
			continue
		}
		s, err := l.discoverDir(p.dir, p.level)
		if err != nil {
			continue
		}
		infos = append(infos, s...)
	}

	// Deduplicate by name, keep highest level
	seen := make(map[string]SkillInfo)
	for _, info := range infos {
		if existing, ok := seen[info.Name]; ok {
			if info.SourceLevel > existing.SourceLevel {
				seen[info.Name] = info
			}
		} else {
			seen[info.Name] = info
		}
	}

	result := make([]SkillInfo, 0, len(seen))
	for _, info := range seen {
		result = append(result, info)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

type SkillInfo struct {
	Name        string
	Version     string
	Description string
	Author      string
	SourceLevel int
	Path        string
}

func (l *SkillLoader) discoverDir(dir string, level int) ([]SkillInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var infos []SkillInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		manifestPath := filepath.Join(dir, e.Name(), "skill.json")
		m, err := LoadManifestFile(manifestPath)
		if err != nil {
			continue
		}
		infos = append(infos, SkillInfo{
			Name:        m.Name,
			Version:     m.Version,
			Description: m.Description,
			Author:      m.Author,
			SourceLevel: level,
			Path:        filepath.Join(dir, e.Name()),
		})
	}
	return infos, nil
}

func (l *SkillLoader) Load(name string) (*Skill, error) {
	l.mu.RLock()
	if s, ok := l.cache[name]; ok {
		l.mu.RUnlock()
		return s, nil
	}
	l.mu.RUnlock()

	paths := []struct {
		dir   string
		level int
	}{
		{l.BuiltinDir, 0},
		{l.UserDir, 1},
		{l.ProjectDir, 2},
	}

	// Search in reverse priority order (builtin first, project last)
	// Since project > user > builtin, we want the last found
	var foundPath string
	var foundLevel int
	for _, p := range paths {
		if p.dir == "" {
			continue
		}
		candidate := filepath.Join(p.dir, name, "skill.json")
		if _, err := os.Stat(candidate); err == nil {
			foundPath = filepath.Join(p.dir, name)
			foundLevel = p.level
		}
	}

	if foundPath == "" {
		return nil, fmt.Errorf("skill not found: %s", name)
	}

	m, err := LoadManifestFile(filepath.Join(foundPath, "skill.json"))
	if err != nil {
		return nil, fmt.Errorf("load manifest: %w", err)
	}

	systemPrompt := ""
	promptPath := filepath.Join(foundPath, "prompts", "system.md")
	if data, err := os.ReadFile(promptPath); err == nil {
		systemPrompt = string(data)
	}

	skill := &Skill{
		Manifest:     *m,
		SystemPrompt: systemPrompt,
		SourcePath:   foundPath,
		SourceLevel:  foundLevel,
	}

	l.mu.Lock()
	l.cache[name] = skill
	l.mu.Unlock()
	return skill, nil
}

func (l *SkillLoader) InvalidateCache(name string) {
	l.mu.Lock()
	delete(l.cache, name)
	l.mu.Unlock()
}

func (l *SkillLoader) InvalidateAll() {
	l.mu.Lock()
	l.cache = make(map[string]*Skill)
	l.mu.Unlock()
}
