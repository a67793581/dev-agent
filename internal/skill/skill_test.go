package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscover_EmptyDirs(t *testing.T) {
	skills, err := Discover(nil)
	if err != nil {
		t.Fatalf("Discover(nil) err = %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("Discover(nil) got %d skills, want 0", len(skills))
	}

	skills, err = Discover([]string{})
	if err != nil {
		t.Fatalf("Discover([]) err = %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("Discover([]) got %d skills, want 0", len(skills))
	}
}

func TestDiscover_NonExistentDir(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "nonexistent")
	skills, err := Discover([]string{missing})
	if err != nil {
		t.Fatalf("Discover(non-existent) should skip or succeed, got err = %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("got %d skills, want 0", len(skills))
	}
}

func TestDiscover_OneSkill(t *testing.T) {
	dir := t.TempDir()
	content := `---
name: my-skill
description: Does something useful.
---

# My Skill

Step 1. Do this.
Step 2. Do that.
`
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	skills, err := Discover([]string{dir})
	if err != nil {
		t.Fatalf("Discover err = %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("got %d skills, want 1", len(skills))
	}
	if skills[0].Name != "my-skill" {
		t.Errorf("Name = %q, want my-skill", skills[0].Name)
	}
	if skills[0].Description != "Does something useful." {
		t.Errorf("Description = %q, want %q", skills[0].Description, "Does something useful.")
	}
	if skills[0].Body != "" {
		t.Errorf("Body should be empty before LoadBody, got %q", skills[0].Body)
	}

	if err := LoadBody(&skills[0]); err != nil {
		t.Fatalf("LoadBody err = %v", err)
	}
	wantBody := "# My Skill\n\nStep 1. Do this.\nStep 2. Do that."
	if skills[0].Body != wantBody {
		t.Errorf("Body = %q, want %q", skills[0].Body, wantBody)
	}
}

func TestDiscover_SkillInSubdir(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "setup-db")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := `---
name: setup-db
description: Set up database.
---
# Setup DB
Instructions here.
`
	if err := os.WriteFile(filepath.Join(subdir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	skills, err := Discover([]string{root})
	if err != nil {
		t.Fatalf("Discover err = %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("got %d skills, want 1", len(skills))
	}
	if skills[0].Name != "setup-db" {
		t.Errorf("Name = %q, want setup-db", skills[0].Name)
	}
	if skills[0].Dir != subdir {
		t.Errorf("Dir = %q, want %q", skills[0].Dir, subdir)
	}
}

func TestDiscover_DedupeByName_FirstWins(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	writeSkill := func(dir, name, desc string) {
		content := "---\nname: " + name + "\ndescription: " + desc + "\n---\nbody"
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	writeSkill(dir1, "same-name", "First description")
	writeSkill(dir2, "same-name", "Second description")

	skills, err := Discover([]string{dir1, dir2})
	if err != nil {
		t.Fatalf("Discover err = %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("got %d skills (dedupe by name), want 1", len(skills))
	}
	if skills[0].Description != "First description" {
		t.Errorf("first dir should win: Description = %q, want First description", skills[0].Description)
	}
}

func TestDiscover_SkipsEmptyName(t *testing.T) {
	dir := t.TempDir()
	content := `---
description: No name here.
---

Body
`
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	skills, err := Discover([]string{dir})
	if err != nil {
		t.Fatalf("Discover err = %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("skill with empty name should be skipped, got %d skills", len(skills))
	}
}

func TestLoadBody_InvalidPath(t *testing.T) {
	s := Skill{path: filepath.Join(t.TempDir(), "nonexistent.md")}
	err := LoadBody(&s)
	if err == nil {
		t.Error("LoadBody with missing file should return error")
	}
}
