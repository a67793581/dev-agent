package prompt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePromptFile_FlagPathWins(t *testing.T) {
	dir := t.TempDir()
	flagPath := filepath.Join(dir, "flag.md")
	if err := os.WriteFile(flagPath, []byte("from-flag"), 0644); err != nil {
		t.Fatal(err)
	}
	projectDir := t.TempDir()
	projectFile := filepath.Join(projectDir, ".devagent", "SOUL.md")
	if err := os.MkdirAll(filepath.Dir(projectFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectFile, []byte("from-project"), 0644); err != nil {
		t.Fatal(err)
	}
	got := ResolvePromptFile(flagPath, projectDir, "SOUL.md")
	if got != "from-flag" {
		t.Errorf("ResolvePromptFile(flagPath, projectDir, SOUL.md) = %q, want %q", got, "from-flag")
	}
}

func TestResolvePromptFile_ProjectFallback(t *testing.T) {
	projectDir := t.TempDir()
	projectFile := filepath.Join(projectDir, ".devagent", "SOUL.md")
	if err := os.MkdirAll(filepath.Dir(projectFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectFile, []byte("from-project"), 0644); err != nil {
		t.Fatal(err)
	}
	got := ResolvePromptFile("", projectDir, "SOUL.md")
	if got != "from-project" {
		t.Errorf("ResolvePromptFile(empty, projectDir, SOUL.md) = %q, want %q", got, "from-project")
	}
}

func TestResolvePromptFile_MissingReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	nonexistent := filepath.Join(dir, "nonexistent.md")
	got := ResolvePromptFile(nonexistent, dir, "SOUL.md")
	if got != "" {
		t.Errorf("ResolvePromptFile(nonexistent, ...) = %q, want empty", got)
	}
}

func TestResolvePromptFile_TrimSpace(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "soul.md")
	if err := os.WriteFile(f, []byte("  \n  hello world  \n  "), 0644); err != nil {
		t.Fatal(err)
	}
	got := ResolvePromptFile(f, "", "SOUL.md")
	if got != "hello world" {
		t.Errorf("ResolvePromptFile(...) = %q, want %q (trimmed)", got, "hello world")
	}
}

func TestResolvePromptFile_EmptyProjectDir(t *testing.T) {
	got := ResolvePromptFile("", "", "SOUL.md")
	if got != "" {
		t.Errorf("ResolvePromptFile(empty, empty, SOUL.md) = %q, want empty", got)
	}
}
