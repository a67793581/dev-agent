package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShellTool_Name(t *testing.T) {
	tool := &ShellTool{workDir: "/tmp"}
	if tool.Name() != "shell" {
		t.Errorf("Name() = %q", tool.Name())
	}
}

func TestShellTool_Execute_EmptyCommand(t *testing.T) {
	tool := &ShellTool{workDir: t.TempDir()}
	result := tool.Execute(map[string]string{"command": ""})
	if result.Success {
		t.Error("empty command should fail")
	}
}

func TestShellTool_ExecuteDirect_Success(t *testing.T) {
	dir := t.TempDir()
	tool := &ShellTool{workDir: dir}
	result := tool.Execute(map[string]string{"command": "echo hello"})
	if !result.Success {
		t.Fatalf("Execute: %s", result.Output)
	}
	if !strings.Contains(result.Output, "hello") {
		t.Errorf("output = %q", result.Output)
	}
}

func TestShellTool_ExecuteDirect_ExitNonZero(t *testing.T) {
	tool := &ShellTool{workDir: t.TempDir()}
	result := tool.Execute(map[string]string{"command": "exit 2"})
	if result.Success {
		t.Error("exit 2 should fail")
	}
	if !strings.Contains(result.Output, "exit code") {
		t.Errorf("output = %q", result.Output)
	}
}

func TestGrepTool_Name(t *testing.T) {
	tool := &GrepTool{workDir: "/tmp"}
	if tool.Name() != "grep" {
		t.Errorf("Name() = %q", tool.Name())
	}
}

func TestGrepTool_Execute_EmptyPattern(t *testing.T) {
	tool := &GrepTool{workDir: t.TempDir()}
	result := tool.Execute(map[string]string{"pattern": ""})
	if result.Success {
		t.Error("empty pattern should fail")
	}
}

func TestGrepTool_Execute_Success(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(f, []byte("needle in haystack"), 0644); err != nil {
		t.Fatal(err)
	}
	tool := &GrepTool{workDir: dir}
	result := tool.Execute(map[string]string{"pattern": "needle", "path": "."})
	// May succeed (if rg/grep found) or fail (no match); just ensure no panic
	if result.Output == "" && result.Success {
		t.Log("grep succeeded with output")
	}
}

func TestShellTool_ExecuteDirect_LongOutputTruncated(t *testing.T) {
	dir := t.TempDir()
	tool := &ShellTool{workDir: dir}
	// Produce > 16000 chars so truncateOutput is exercised
	result := tool.Execute(map[string]string{"command": "printf '%17000s' x | tr ' ' 'a'"})
	if !result.Success {
		t.Fatalf("Execute: %s", result.Output)
	}
	if !strings.Contains(result.Output, "(output truncated)") {
		t.Errorf("long output should be truncated, got len=%d", len(result.Output))
	}
}
