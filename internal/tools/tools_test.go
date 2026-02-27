package tools

import (
	"testing"
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
