package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStrReplaceTool_Name(t *testing.T) {
	tool := &StrReplaceTool{workDir: "/tmp"}
	if tool.Name() != "str_replace" {
		t.Errorf("Name() = %q", tool.Name())
	}
}

func TestStrReplaceTool_Execute_Success(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(f, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}
	tool := &StrReplaceTool{workDir: dir}
	result := tool.Execute(map[string]string{"path": "f.txt", "old_str": "world", "new_str": "there"})
	if !result.Success {
		t.Fatalf("Execute: %s", result.Output)
	}
	data, _ := os.ReadFile(f)
	if string(data) != "hello there" {
		t.Errorf("file content = %q", data)
	}
}

func TestStrReplaceTool_Execute_EmptyOldStr(t *testing.T) {
	tool := &StrReplaceTool{workDir: "/tmp"}
	result := tool.Execute(map[string]string{"path": "x", "old_str": "", "new_str": "y"})
	if result.Success {
		t.Error("empty old_str should fail")
	}
}

func TestStrReplaceTool_Execute_NotFound(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f.txt")
	os.WriteFile(f, []byte("hello"), 0644)
	tool := &StrReplaceTool{workDir: dir}
	result := tool.Execute(map[string]string{"path": "f.txt", "old_str": "xyz", "new_str": "y"})
	if result.Success {
		t.Error("old_str not found should fail")
	}
}

func TestStrReplaceTool_Execute_MultipleMatches(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f.txt")
	os.WriteFile(f, []byte("a a a"), 0644)
	tool := &StrReplaceTool{workDir: dir}
	result := tool.Execute(map[string]string{"path": "f.txt", "old_str": "a", "new_str": "b"})
	if result.Success {
		t.Error("multiple matches should fail")
	}
}

func TestInsertLineTool_Name(t *testing.T) {
	tool := &InsertLineTool{workDir: "/tmp"}
	if tool.Name() != "insert_line" {
		t.Errorf("Name() = %q", tool.Name())
	}
}

func TestInsertLineTool_Execute_Success(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f.txt")
	os.WriteFile(f, []byte("line1\nline2\nline3"), 0644)
	tool := &InsertLineTool{workDir: dir}
	result := tool.Execute(map[string]string{"path": "f.txt", "after": "line2", "content": "inserted"})
	if !result.Success {
		t.Fatalf("Execute: %s", result.Output)
	}
	data, _ := os.ReadFile(f)
	if string(data) != "line1\nline2\ninserted\nline3" {
		t.Errorf("content = %q", data)
	}
}

func TestInsertLineTool_Execute_EmptyAfter(t *testing.T) {
	tool := &InsertLineTool{workDir: "/tmp"}
	result := tool.Execute(map[string]string{"path": "x", "after": "", "content": "y"})
	if result.Success {
		t.Error("empty after should fail")
	}
}

func TestInsertLineTool_Execute_LineNotFound(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f.txt")
	os.WriteFile(f, []byte("a\nb"), 0644)
	tool := &InsertLineTool{workDir: dir}
	result := tool.Execute(map[string]string{"path": "f.txt", "after": "nonexistent", "content": "y"})
	if result.Success {
		t.Error("line not found should fail")
	}
}
