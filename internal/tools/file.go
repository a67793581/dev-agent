package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ReadFileTool struct {
	workDir string
}

func (t *ReadFileTool) Name() string { return "read_file" }

func (t *ReadFileTool) Execute(args map[string]string) Result {
	path := t.resolvePath(args["path"])

	info, err := os.Stat(path)
	if err != nil {
		return Result{Success: false, Output: fmt.Sprintf("cannot access %s: %v", path, err)}
	}
	if info.IsDir() {
		return Result{Success: false, Output: fmt.Sprintf("%s is a directory, use list_dir instead", path)}
	}
	if info.Size() > 512*1024 {
		return Result{Success: false, Output: fmt.Sprintf("file too large (%d bytes), consider reading specific sections", info.Size())}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Result{Success: false, Output: fmt.Sprintf("read error: %v", err)}
	}

	lines := strings.Split(string(data), "\n")
	var sb strings.Builder
	for i, line := range lines {
		sb.WriteString(fmt.Sprintf("%4d | %s\n", i+1, line))
	}

	return Result{Success: true, Output: sb.String()}
}

func (t *ReadFileTool) resolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(t.workDir, p)
}

type WriteFileTool struct {
	workDir string
}

func (t *WriteFileTool) Name() string { return "write_file" }

func (t *WriteFileTool) Execute(args map[string]string) Result {
	path := t.resolvePath(args["path"])
	content := args["content"]

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return Result{Success: false, Output: fmt.Sprintf("cannot create directory %s: %v", dir, err)}
	}

	existed := false
	if _, err := os.Stat(path); err == nil {
		existed = true
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return Result{Success: false, Output: fmt.Sprintf("write error: %v", err)}
	}

	action := "Created"
	if existed {
		action = "Updated"
	}
	return Result{
		Success: true,
		Output:  fmt.Sprintf("%s %s (%d bytes)", action, path, len(content)),
	}
}

func (t *WriteFileTool) resolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(t.workDir, p)
}

type ListDirTool struct {
	workDir string
}

func (t *ListDirTool) Name() string { return "list_dir" }

func (t *ListDirTool) Execute(args map[string]string) Result {
	path := args["path"]
	if path == "" || path == "." {
		path = t.workDir
	} else if !filepath.IsAbs(path) {
		path = filepath.Join(t.workDir, path)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return Result{Success: false, Output: fmt.Sprintf("cannot list %s: %v", path, err)}
	}

	var sb strings.Builder
	for _, entry := range entries {
		prefix := "📄"
		if entry.IsDir() {
			prefix = "📁"
		}
		info, _ := entry.Info()
		size := ""
		if info != nil && !entry.IsDir() {
			size = fmt.Sprintf(" (%d bytes)", info.Size())
		}
		sb.WriteString(fmt.Sprintf("%s %s%s\n", prefix, entry.Name(), size))
	}

	if sb.Len() == 0 {
		return Result{Success: true, Output: "(empty directory)"}
	}
	return Result{Success: true, Output: sb.String()}
}

type SearchFilesTool struct {
	workDir string
}

func (t *SearchFilesTool) Name() string { return "search_files" }

func (t *SearchFilesTool) Execute(args map[string]string) Result {
	root := args["path"]
	if root == "" || root == "." {
		root = t.workDir
	} else if !filepath.IsAbs(root) {
		root = filepath.Join(t.workDir, root)
	}

	pattern := args["pattern"]
	if pattern == "" {
		pattern = "*"
	}

	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == "node_modules" || base == "__pycache__" || base == ".venv" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		matched, _ := filepath.Match(pattern, filepath.Base(path))
		if matched {
			rel, _ := filepath.Rel(root, path)
			matches = append(matches, rel)
		}
		if len(matches) >= 200 {
			return fmt.Errorf("too many results")
		}
		return nil
	})

	if err != nil && len(matches) == 0 {
		return Result{Success: false, Output: fmt.Sprintf("search error: %v", err)}
	}

	if len(matches) == 0 {
		return Result{Success: true, Output: "no files found matching pattern: " + pattern}
	}

	return Result{Success: true, Output: strings.Join(matches, "\n")}
}
