package prompt

import (
	"strings"
	"testing"
)

func TestBuildSystemPrompt_EmptySoulAndGuidelines(t *testing.T) {
	got := BuildSystemPrompt("", "")
	if !strings.Contains(got, "You are DevAgent") {
		t.Error("BuildSystemPrompt(empty, empty) should contain identity")
	}
	if !strings.Contains(got, "## Available Commands") {
		t.Error("BuildSystemPrompt(empty, empty) should contain body")
	}
	if strings.Contains(got, "custom-soul") || strings.Contains(got, "custom-guidelines") {
		t.Error("BuildSystemPrompt(empty, empty) should not contain custom content")
	}
}

func TestBuildSystemPrompt_SoulOnly(t *testing.T) {
	soul := "I am a custom soul."
	got := BuildSystemPrompt(soul, "")
	if !strings.Contains(got, "You are DevAgent") {
		t.Error("should contain identity")
	}
	if !strings.Contains(got, soul) {
		t.Errorf("should contain soul %q", soul)
	}
	if !strings.Contains(got, "## Available Commands") {
		t.Error("should contain body")
	}
	idxIdentity := strings.Index(got, "You are DevAgent")
	idxSoul := strings.Index(got, soul)
	idxBody := strings.Index(got, "## Available Commands")
	if idxIdentity >= idxSoul || idxSoul >= idxBody {
		t.Errorf("order should be identity < soul < body; got identity=%d soul=%d body=%d", idxIdentity, idxSoul, idxBody)
	}
}

func TestBuildSystemPrompt_GuidelinesOnly(t *testing.T) {
	guidelines := "Always use tabs."
	got := BuildSystemPrompt("", guidelines)
	if !strings.Contains(got, "You are DevAgent") {
		t.Error("should contain identity")
	}
	if !strings.Contains(got, "## Available Commands") {
		t.Error("should contain body")
	}
	if !strings.Contains(got, guidelines) {
		t.Errorf("should contain guidelines %q", guidelines)
	}
	idxBody := strings.Index(got, "## Available Commands")
	idxGuidelines := strings.Index(got, guidelines)
	if idxBody >= idxGuidelines {
		t.Errorf("order should be body before guidelines; got body=%d guidelines=%d", idxBody, idxGuidelines)
	}
}

func TestBuildSystemPrompt_BothSoulAndGuidelines(t *testing.T) {
	soul := "Custom soul."
	guidelines := "Custom guidelines."
	got := BuildSystemPrompt(soul, guidelines)
	if !strings.Contains(got, soul) {
		t.Errorf("should contain soul %q", soul)
	}
	if !strings.Contains(got, guidelines) {
		t.Errorf("should contain guidelines %q", guidelines)
	}
	idxSoul := strings.Index(got, soul)
	idxBody := strings.Index(got, "## Available Commands")
	idxGuidelines := strings.Index(got, guidelines)
	if idxSoul >= idxBody || idxBody >= idxGuidelines {
		t.Errorf("order should be soul < body < guidelines; got soul=%d body=%d guidelines=%d", idxSoul, idxBody, idxGuidelines)
	}
}

func TestBuildSystemPrompt_TrimSpace(t *testing.T) {
	soul := "  trimmed soul  "
	got := BuildSystemPrompt(soul, "")
	if !strings.Contains(got, "trimmed soul") {
		t.Errorf("soul should be trimmed, got %q", got)
	}
	if strings.Contains(got, "  trimmed soul  ") {
		t.Error("soul content in output should not have leading/trailing spaces")
	}
	guidelines := "  trimmed guidelines  "
	got2 := BuildSystemPrompt("", guidelines)
	if !strings.Contains(got2, "trimmed guidelines") {
		t.Errorf("guidelines should be trimmed, got %q", got2)
	}
}

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

func TestBuildProjectContext(t *testing.T) {
	got := BuildProjectContext("/project", "├── main.go\n└── pkg/\n")
	if !strings.Contains(got, "Project path: /project") {
		t.Error("should contain project path")
	}
	if !strings.Contains(got, "├── main.go") {
		t.Error("should contain file tree")
	}
}

func TestBuildUserTask(t *testing.T) {
	got := BuildUserTask("Add tests")
	if !strings.Contains(got, "## User Task") {
		t.Error("should contain section header")
	}
	if !strings.Contains(got, "Add tests") {
		t.Error("should contain task text")
	}
}

func TestBuildObservation_Success(t *testing.T) {
	got := BuildObservation("read_file", true, "line 1\nline 2")
	if !strings.Contains(got, "read_file") || !strings.Contains(got, "SUCCESS") {
		t.Error("should contain command and SUCCESS")
	}
	if !strings.Contains(got, "line 1") {
		t.Error("should contain output")
	}
}

func TestBuildObservation_Failed(t *testing.T) {
	got := BuildObservation("shell", false, "exit code 1")
	if !strings.Contains(got, "FAILED") {
		t.Error("should contain FAILED")
	}
}

func TestBuildObservation_Truncated(t *testing.T) {
	long := strings.Repeat("x", 10000)
	got := BuildObservation("read_file", true, long)
	if !strings.Contains(got, "(output truncated)") {
		t.Error("long output should be truncated")
	}
}

func TestBuildDebugPrompt_WithoutTestCode(t *testing.T) {
	got := BuildDebugPrompt("code", "error", "")
	if !strings.Contains(got, "## Code Repair Task") {
		t.Error("should contain section")
	}
	if !strings.Contains(got, "code") || !strings.Contains(got, "error") {
		t.Error("should contain code and error")
	}
	if strings.Contains(got, "Test Code") {
		t.Error("should not contain Test Code when testCode is empty")
	}
}

func TestBuildDebugPrompt_WithTestCode(t *testing.T) {
	got := BuildDebugPrompt("code", "error", "test code")
	if !strings.Contains(got, "### Test Code") {
		t.Error("should contain Test Code section")
	}
	if !strings.Contains(got, "test code") {
		t.Error("should contain test code content")
	}
}
