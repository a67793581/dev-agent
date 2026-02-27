package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileTool_Name(t *testing.T) {
	tool := &ReadFileTool{workDir: "/tmp"}
	if tool.Name() != "read_file" {
		t.Errorf("Name() = %q", tool.Name())
	}
}

func TestReadFileTool_Execute_Success(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f.txt")
	os.WriteFile(f, []byte("line1\nline2"), 0644)
	tool := &ReadFileTool{workDir: dir}
	result := tool.Execute(map[string]string{"path": "f.txt"})
	if !result.Success {
		t.Fatalf("Execute: %s", result.Output)
	}
	if !strings.Contains(result.Output, "1 | line1") || !strings.Contains(result.Output, "2 | line2") {
		t.Errorf("output = %q", result.Output)
	}
}

func TestReadFileTool_Execute_NotFound(t *testing.T) {
	tool := &ReadFileTool{workDir: t.TempDir()}
	result := tool.Execute(map[string]string{"path": "nonexistent.txt"})
	if result.Success {
		t.Error("nonexistent file should fail")
	}
}

func TestReadFileTool_Execute_Dir(t *testing.T) {
	dir := t.TempDir()
	tool := &ReadFileTool{workDir: dir}
	result := tool.Execute(map[string]string{"path": "."})
	if result.Success {
		t.Error("directory should fail")
	}
}

func TestReadFileTool_Execute_TooLarge(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "big.txt")
	// > 512*1024
	data := make([]byte, 600*1024)
	os.WriteFile(f, data, 0644)
	tool := &ReadFileTool{workDir: dir}
	result := tool.Execute(map[string]string{"path": "big.txt"})
	if result.Success {
		t.Error("file too large should fail")
	}
}

func TestWriteFileTool_Name(t *testing.T) {
	tool := &WriteFileTool{workDir: "/tmp"}
	if tool.Name() != "write_file" {
		t.Errorf("Name() = %q", tool.Name())
	}
}

func TestWriteFileTool_Execute_Create(t *testing.T) {
	dir := t.TempDir()
	tool := &WriteFileTool{workDir: dir}
	result := tool.Execute(map[string]string{"path": "new.txt", "content": "hello"})
	if !result.Success {
		t.Fatalf("Execute: %s", result.Output)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "new.txt"))
	if string(data) != "hello" {
		t.Errorf("content = %q", data)
	}
}

func TestWriteFileTool_Execute_Update(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f.txt")
	os.WriteFile(f, []byte("old"), 0644)
	tool := &WriteFileTool{workDir: dir}
	result := tool.Execute(map[string]string{"path": "f.txt", "content": "new"})
	if !result.Success {
		t.Fatalf("Execute: %s", result.Output)
	}
	if !strings.Contains(result.Output, "Updated") {
		t.Errorf("output = %q", result.Output)
	}
}

func TestListDirTool_Name(t *testing.T) {
	tool := &ListDirTool{workDir: "/tmp"}
	if tool.Name() != "list_dir" {
		t.Errorf("Name() = %q", tool.Name())
	}
}

func TestListDirTool_Execute_Success(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), nil, 0644)
	os.Mkdir(filepath.Join(dir, "sub"), 0755)
	tool := &ListDirTool{workDir: dir}
	result := tool.Execute(map[string]string{"path": "."})
	if !result.Success {
		t.Fatalf("Execute: %s", result.Output)
	}
	if !strings.Contains(result.Output, "a.txt") {
		t.Errorf("output = %q", result.Output)
	}
}

func TestSearchFilesTool_Name(t *testing.T) {
	tool := &SearchFilesTool{workDir: "/tmp"}
	if tool.Name() != "search_files" {
		t.Errorf("Name() = %q", tool.Name())
	}
}

func TestSearchFilesTool_Execute_NoMatch(t *testing.T) {
	dir := t.TempDir()
	tool := &SearchFilesTool{workDir: dir}
	result := tool.Execute(map[string]string{"path": ".", "pattern": "*.nonexistent"})
	if !result.Success {
		t.Fatalf("Execute: %s", result.Output)
	}
	if !strings.Contains(result.Output, "no files found") {
		t.Errorf("output = %q", result.Output)
	}
}

func TestSearchFilesTool_Execute_Match(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), nil, 0644)
	tool := &SearchFilesTool{workDir: dir}
	result := tool.Execute(map[string]string{"path": ".", "pattern": "*.go"})
	if !result.Success {
		t.Fatalf("Execute: %s", result.Output)
	}
	if !strings.Contains(result.Output, "main.go") {
		t.Errorf("output = %q", result.Output)
	}
}

func TestDoneTool_Name(t *testing.T) {
	tool := &DoneTool{}
	if tool.Name() != "done" {
		t.Errorf("Name() = %q", tool.Name())
	}
}

func TestDoneTool_Execute(t *testing.T) {
	tool := &DoneTool{}
	result := tool.Execute(map[string]string{"summary": "All done"})
	if !result.Success {
		t.Error("done should succeed")
	}
	if result.Output != "All done" {
		t.Errorf("Output = %q", result.Output)
	}
	result2 := tool.Execute(map[string]string{})
	if result2.Output != "Task completed." {
		t.Errorf("empty summary default = %q", result2.Output)
	}
}
