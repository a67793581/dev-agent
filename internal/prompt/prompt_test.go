package prompt

import (
	"strings"
	"testing"
)

func TestBuildSkillsContext_Empty(t *testing.T) {
	got := BuildSkillsContext(nil)
	if got != "" {
		t.Errorf("BuildSkillsContext(nil) = %q, want empty", got)
	}
	got = BuildSkillsContext([]SkillMeta{})
	if got != "" {
		t.Errorf("BuildSkillsContext([]) = %q, want empty", got)
	}
}

func TestBuildSkillsContext_OneSkill(t *testing.T) {
	skills := []SkillMeta{
		{Name: "setup-db", Description: "Set up the database."},
	}
	got := BuildSkillsContext(skills)
	if !strings.Contains(got, "## Available Skills") {
		t.Error("output should contain ## Available Skills")
	}
	if !strings.Contains(got, "read_skill") {
		t.Error("output should mention read_skill")
	}
	if !strings.Contains(got, "**setup-db**") {
		t.Error("output should contain skill name")
	}
	if !strings.Contains(got, "Set up the database.") {
		t.Error("output should contain skill description")
	}
}

func TestBuildSkillsContext_MultipleSkills(t *testing.T) {
	skills := []SkillMeta{
		{Name: "skill-a", Description: "Desc A"},
		{Name: "skill-b", Description: "Desc B"},
	}
	got := BuildSkillsContext(skills)
	if !strings.Contains(got, "**skill-a**") || !strings.Contains(got, "Desc A") {
		t.Error("output should contain first skill")
	}
	if !strings.Contains(got, "**skill-b**") || !strings.Contains(got, "Desc B") {
		t.Error("output should contain second skill")
	}
}
