package skill

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeTestSkill(name, version, desc, prompt string, level int) *Skill {
	return &Skill{
		Manifest: SkillManifest{
			Name: name, Version: version, Description: desc,
		},
		SystemPrompt: prompt,
		SourceLevel:  level,
	}
}

func TestInjectSkillsEmpty(t *testing.T) {
	result := InjectSkills("base prompt", nil)
	assert.Equal(t, "base prompt", result)
}

func TestInjectSkillsNoSkills(t *testing.T) {
	result := InjectSkills("base prompt", []*Skill{})
	assert.Equal(t, "base prompt", result)
}

func TestInjectSkillsSingle(t *testing.T) {
	skills := []*Skill{makeTestSkill("review", "1.0.0", "Code review", "Review the diff.", 0)}
	result := InjectSkills("You are a helpful assistant.", skills)
	assert.Contains(t, result, "You are a helpful assistant.")
	assert.Contains(t, result, "## Active Skills")
	assert.Contains(t, result, "review")
	assert.Contains(t, result, "1.0.0")
	assert.Contains(t, result, "Code review")
	assert.Contains(t, result, "Review the diff.")
}

func TestInjectSkillsMultiple(t *testing.T) {
	skills := []*Skill{
		makeTestSkill("review", "1.0.0", "Review code", "Review the diff.", 0),
		makeTestSkill("test-gen", "2.0.0", "Generate tests", "Write tests.", 0),
	}
	result := InjectSkills("Base.", skills)
	assert.Contains(t, result, "review")
	assert.Contains(t, result, "test-gen")
	assert.Contains(t, result, "---")
}

func TestInjectSkillsDedupByLevel(t *testing.T) {
	skills := []*Skill{
		makeTestSkill("common", "1.0.0", "builtin", "builtin prompt", 0),
		makeTestSkill("common", "2.0.0", "user", "user prompt", 1),
	}
	result := InjectSkills("Base.", skills)
	assert.Contains(t, result, "user prompt")
	assert.NotContains(t, result, "builtin prompt")
}

func TestInjectSkillsOrder(t *testing.T) {
	skills := []*Skill{
		makeTestSkill("z-skill", "1.0.0", "", "z content", 0),
		makeTestSkill("a-skill", "1.0.0", "", "a content", 0),
	}
	result := InjectSkills("Base.", skills)
	aIdx := strings.Index(result, "a-skill")
	zIdx := strings.Index(result, "z-skill")
	assert.GreaterOrEqual(t, aIdx, 0, "a-skill should be found")
	assert.GreaterOrEqual(t, zIdx, 0, "z-skill should be found")
	assert.Less(t, aIdx, zIdx, "a-skill should appear before z-skill")
}

func TestSelectActiveSkills(t *testing.T) {
	all := []*Skill{
		makeTestSkill("a", "1.0.0", "", "", 0),
		makeTestSkill("b", "1.0.0", "", "", 0),
		makeTestSkill("c", "1.0.0", "", "", 0),
	}
	active := SelectActiveSkills(all, []string{"a", "c"})
	assert.Len(t, active, 2)
	assert.Equal(t, "a", active[0].Manifest.Name)
	assert.Equal(t, "c", active[1].Manifest.Name)
}

func TestSelectActiveSkillsNone(t *testing.T) {
	all := []*Skill{makeTestSkill("a", "1.0.0", "", "", 0)}
	active := SelectActiveSkills(all, []string{"nonexistent"})
	assert.Empty(t, active)
}
