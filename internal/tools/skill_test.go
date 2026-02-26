package tools

import (
	"devagent/internal/skill"
	"os"
	"path/filepath"
	"testing"
)

func TestReadSkillTool_Name(t *testing.T) {
	tool := NewReadSkillTool(nil)
	if tool.Name() != "read_skill" {
		t.Errorf("Name() = %q, want read_skill", tool.Name())
	}
}

func TestReadSkillTool_Execute_EmptyName(t *testing.T) {
	tool := NewReadSkillTool([]skill.Skill{})
	result := tool.Execute(map[string]string{})
	if result.Success {
		t.Error("Execute with no name should fail")
	}
	if result.Output != "read_skill requires \"name\" argument" {
		t.Errorf("Output = %q", result.Output)
	}

	result = tool.Execute(map[string]string{"name": ""})
	if result.Success {
		t.Error("Execute with empty name should fail")
	}
}

func TestReadSkillTool_Execute_UnknownSkill(t *testing.T) {
	tool := NewReadSkillTool([]skill.Skill{
		{Name: "known-skill", Description: "A skill"},
	})
	result := tool.Execute(map[string]string{"name": "unknown-skill"})
	if result.Success {
		t.Error("Execute with unknown skill should fail")
	}
	if result.Output == "" {
		t.Error("Output should contain error message")
	}
}

func TestReadSkillTool_Execute_Success(t *testing.T) {
	dir := t.TempDir()
	content := `---
name: test-skill
description: Test skill for unit test.
---

# Test Skill

Follow these steps.
`
	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	skills, err := skill.Discover([]string{dir})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}

	tool := NewReadSkillTool(skills)
	result := tool.Execute(map[string]string{"name": "test-skill"})
	if !result.Success {
		t.Fatalf("Execute failed: %s", result.Output)
	}
	wantBody := "# Test Skill\n\nFollow these steps."
	if result.Output != wantBody {
		t.Errorf("Output = %q, want %q", result.Output, wantBody)
	}
}

func TestReadSkillTool_Execute_EmptySkillsList(t *testing.T) {
	tool := NewReadSkillTool([]skill.Skill{})
	result := tool.Execute(map[string]string{"name": "any"})
	if result.Success {
		t.Error("Execute with empty skills list should fail for any name")
	}
}
