package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"devagent/internal/sandbox"
)

func TestTranslatePaths_DockerMode(t *testing.T) {
	reg := NewRegistry()
	reg.SetContainerPath("/home/user/project", "/workspace")

	reg.Register(&ReadFileTool{workDir: "/home/user/project"})

	tests := []struct {
		name     string
		tool     string
		argKey   string
		input    string
		expected string
	}{
		{"exact workspace", "read_file", "path", "/workspace", "/home/user/project"},
		{"subpath", "read_file", "path", "/workspace/src/main.go", "/home/user/project/src/main.go"},
		{"relative path unchanged", "read_file", "path", "src/main.go", "src/main.go"},
		{"host abs path unchanged", "read_file", "path", "/etc/hosts", "/etc/hosts"},
		{"empty path unchanged", "read_file", "path", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]string{tt.argKey: tt.input}
			reg.translatePaths(tt.tool, args)
			if args[tt.argKey] != tt.expected {
				t.Errorf("got %q, want %q", args[tt.argKey], tt.expected)
			}
		})
	}
}

func TestTranslatePaths_NoDockerMode(t *testing.T) {
	reg := NewRegistry()

	args := map[string]string{"path": "/workspace/file.txt"}
	reg.translatePaths("read_file", args)
	if args["path"] != "/workspace/file.txt" {
		t.Errorf("path should not be translated when containerWorkDir is empty, got %q", args["path"])
	}
}

func TestTranslatePaths_NonFileTool(t *testing.T) {
	reg := NewRegistry()
	reg.SetContainerPath("/home/user/project", "/workspace")

	args := map[string]string{"command": "ls /workspace"}
	reg.translatePaths("shell", args)
	if args["command"] != "ls /workspace" {
		t.Errorf("shell command should not be translated, got %q", args["command"])
	}
}

func TestTranslatePaths_AllFileTools(t *testing.T) {
	reg := NewRegistry()
	reg.SetContainerPath("/host/dir", "/workspace")

	for toolName, argKey := range pathArgForTool {
		t.Run(toolName, func(t *testing.T) {
			args := map[string]string{argKey: "/workspace/test.go"}
			reg.translatePaths(toolName, args)
			if args[argKey] != "/host/dir/test.go" {
				t.Errorf("%s: got %q, want %q", toolName, args[argKey], "/host/dir/test.go")
			}
		})
	}
}

func TestRegistry_Get(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&ReadFileTool{workDir: "/tmp"})
	tool, ok := reg.Get("read_file")
	if !ok || tool == nil {
		t.Fatal("Get(read_file) should return tool")
	}
	if tool.Name() != "read_file" {
		t.Errorf("Name = %q", tool.Name())
	}
	_, ok = reg.Get("unknown")
	if ok {
		t.Error("Get(unknown) should return false")
	}
}

func TestRegistry_Execute_UnknownCommand(t *testing.T) {
	reg := NewRegistry()
	result := reg.Execute("unknown_cmd", map[string]string{})
	if result.Success {
		t.Error("unknown command should fail")
	}
	if result.Output == "" {
		t.Error("Output should contain error message")
	}
}

func TestRegistry_Execute_KnownCommand(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	reg.Register(&ReadFileTool{workDir: dir})
	f := filepath.Join(dir, "x.txt")
	os.WriteFile(f, []byte("hi"), 0644)
	result := reg.Execute("read_file", map[string]string{"path": "x.txt"})
	if !result.Success {
		t.Fatalf("Execute: %s", result.Output)
	}
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&ReadFileTool{workDir: "/tmp"})
	reg.Register(&WriteFileTool{workDir: "/tmp"})
	names := reg.List()
	if len(names) != 2 {
		t.Errorf("List() len = %d, want 2", len(names))
	}
}

func TestDefaultRegistry(t *testing.T) {
	reg := DefaultRegistry("/work", nil)
	if reg == nil {
		t.Fatal("DefaultRegistry should not return nil")
	}
	names := reg.List()
	if len(names) == 0 {
		t.Error("DefaultRegistry should register tools")
	}
	if _, ok := reg.Get("read_file"); !ok {
		t.Error("should have read_file")
	}
	if _, ok := reg.Get("shell"); !ok {
		t.Error("should have shell")
	}
	if _, ok := reg.Get("done"); !ok {
		t.Error("should have done")
	}
}

func TestRegistry_SetSandbox(t *testing.T) {
	reg := NewRegistry()
	sb := sandbox.NewSandbox(&sandbox.Policy{WorkDir: t.TempDir(), Shell: &sandbox.ShellPolicy{}, Path: &sandbox.PathPolicy{}})
	reg.SetSandbox(sb)
	// Execute a path tool that would be checked by sandbox
	result := reg.Execute("read_file", map[string]string{"path": "nonexistent"})
	// Should fail for read error, not sandbox
	if result.Success {
		t.Error("nonexistent path should fail")
	}
}

func TestRegistry_Execute_SandboxDenies(t *testing.T) {
	workDir := t.TempDir()
	reg := NewRegistry()
	reg.Register(&ReadFileTool{workDir: workDir})
	policy := &sandbox.Policy{
		Mode:    sandbox.ModeStrict,
		WorkDir: workDir,
		Shell:   &sandbox.ShellPolicy{},
		Path:    &sandbox.PathPolicy{},
		ApproveFunc: func(action string) bool {
			return false
		},
	}
	sb := sandbox.NewSandbox(policy)
	reg.SetSandbox(sb)
	// In strict mode, write_file requires approval; we deny
	reg.Register(&WriteFileTool{workDir: workDir})
	result := reg.Execute("write_file", map[string]string{"path": "x.go", "content": "x"})
	if result.Success {
		t.Error("sandbox should deny write_file when ApproveFunc returns false")
	}
	if !strings.Contains(result.Output, "blocked") {
		t.Errorf("output should mention blocked: %q", result.Output)
	}
}
