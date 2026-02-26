package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type StrReplaceTool struct {
	workDir string
}

func (t *StrReplaceTool) Name() string { return "str_replace" }

func (t *StrReplaceTool) Execute(args map[string]string) Result {
	path := t.resolvePath(args["path"])
	oldStr := args["old_str"]
	newStr := args["new_str"]

	if oldStr == "" {
		return Result{Success: false, Output: "old_str cannot be empty"}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Result{Success: false, Output: fmt.Sprintf("read error: %v", err)}
	}

	content := string(data)
	count := strings.Count(content, oldStr)

	if count == 0 {
		preview := oldStr
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		return Result{
			Success: false,
			Output:  fmt.Sprintf("old_str not found in %s. Searched for:\n%s", path, preview),
		}
	}

	if count > 1 {
		return Result{
			Success: false,
			Output:  fmt.Sprintf("old_str found %d times in %s. Provide a more unique string to match exactly once.", count, path),
		}
	}

	newContent := strings.Replace(content, oldStr, newStr, 1)

	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return Result{Success: false, Output: fmt.Sprintf("write error: %v", err)}
	}

	return Result{
		Success: true,
		Output:  fmt.Sprintf("Replaced in %s (%d bytes -> %d bytes)", path, len(content), len(newContent)),
	}
}

func (t *StrReplaceTool) resolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(t.workDir, p)
}

type InsertLineTool struct {
	workDir string
}

func (t *InsertLineTool) Name() string { return "insert_line" }

func (t *InsertLineTool) Execute(args map[string]string) Result {
	path := t.resolvePath(args["path"])
	afterLine := args["after"]
	content := args["content"]

	if afterLine == "" {
		return Result{Success: false, Output: "after (the line after which to insert) cannot be empty"}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Result{Success: false, Output: fmt.Sprintf("read error: %v", err)}
	}

	lines := strings.Split(string(data), "\n")
	var newLines []string
	found := false

	for _, line := range lines {
		newLines = append(newLines, line)
		if !found && strings.TrimSpace(line) == strings.TrimSpace(afterLine) {
			newLines = append(newLines, content)
			found = true
		}
	}

	if !found {
		return Result{
			Success: false,
			Output:  fmt.Sprintf("line not found in %s: %s", path, afterLine),
		}
	}

	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return Result{Success: false, Output: fmt.Sprintf("write error: %v", err)}
	}

	return Result{
		Success: true,
		Output:  fmt.Sprintf("Inserted content after matching line in %s", path),
	}
}

func (t *InsertLineTool) resolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(t.workDir, p)
}
