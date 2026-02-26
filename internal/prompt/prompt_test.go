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
