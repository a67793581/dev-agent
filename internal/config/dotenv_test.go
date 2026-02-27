package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnv_EmptyPath_NoLocalOrGlobal(t *testing.T) {
	// No -env flag, and no .env in cwd or ~/.devagent.env (or they don't exist)
	// Should return nil (no files to load)
	err := LoadEnv("")
	if err != nil {
		t.Fatalf("LoadEnv(\"\") = %v, want nil", err)
	}
}

func TestLoadEnv_ExplicitFileNotFound(t *testing.T) {
	err := LoadEnv("/nonexistent/path/.env")
	if err == nil {
		t.Fatal("LoadEnv(nonexistent) want error, got nil")
	}
	if err.Error() != "env file not found: /nonexistent/path/.env" {
		t.Errorf("LoadEnv error = %q", err.Error())
	}
}

func TestLoadEnv_ExplicitFileExists(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("TEST_KEY=from_file\n"), 0644); err != nil {
		t.Fatal(err)
	}
	err := LoadEnv(envPath)
	if err != nil {
		t.Fatalf("LoadEnv(%q) = %v, want nil", envPath, err)
	}
	if v := os.Getenv("TEST_KEY"); v != "from_file" {
		t.Errorf("TEST_KEY = %q, want from_file", v)
	}
	os.Unsetenv("TEST_KEY")
}

func TestLoadEnv_LoadError(t *testing.T) {
	// Create a directory and pass it as "env file" - godotenv will fail to parse it as .env
	dir := t.TempDir()
	err := LoadEnv(dir)
	if err == nil {
		t.Fatal("LoadEnv(directory) should error")
	}
}
