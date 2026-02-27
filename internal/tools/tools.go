package tools

import (
	"devagent/internal/sandbox"
	"fmt"
	"path/filepath"
	"strings"
)

type Result struct {
	Success bool
	Output  string
}

type Tool interface {
	Name() string
	Execute(args map[string]string) Result
}

// Registry manages tools and applies sandbox + path translation before execution.
type Registry struct {
	tools           map[string]Tool
	sandbox         *sandbox.Sandbox
	hostWorkDir     string
	containerWorkDir string // "/workspace" when Docker is active, empty otherwise
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) SetSandbox(sb *sandbox.Sandbox) {
	r.sandbox = sb
}

// SetContainerPath enables container-to-host path translation for file tools.
func (r *Registry) SetContainerPath(hostWorkDir, containerWorkDir string) {
	r.hostWorkDir = hostWorkDir
	r.containerWorkDir = containerWorkDir
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

var pathArgForTool = map[string]string{
	"read_file": "path", "write_file": "path", "str_replace": "path", "insert_line": "path",
	"list_dir": "path", "search_files": "path", "grep": "path",
}

func (r *Registry) Execute(name string, args map[string]string) Result {
	tool, ok := r.tools[name]
	if !ok {
		return Result{
			Success: false,
			Output:  fmt.Sprintf("unknown command: %s", name),
		}
	}

	if r.containerWorkDir != "" {
		r.translatePaths(name, args)
	}

	if r.sandbox != nil {
		result := r.sandbox.Check(name, args)
		if !result.Allow {
			var out string
			if result.DenyErr != nil {
				out = fmt.Sprintf("blocked by sandbox: %v", result.DenyErr)
			} else {
				out = "blocked by sandbox"
			}
			if result.ApprovalAction != "" {
				out = out + "\n" + result.ApprovalAction
			}
			return Result{Success: false, Output: out}
		}
	}
	return tool.Execute(args)
}

// translatePaths rewrites container paths (/workspace/...) to host paths for file tools.
// Shell commands don't need translation since Docker mounts workDir at /workspace.
func (r *Registry) translatePaths(toolName string, args map[string]string) {
	if r.containerWorkDir == "" {
		return
	}
	argKey, ok := pathArgForTool[toolName]
	if !ok {
		return
	}
	p := args[argKey]
	if p == "" {
		return
	}
	prefix := r.containerWorkDir
	if p == prefix {
		args[argKey] = r.hostWorkDir
	} else if strings.HasPrefix(p, prefix+"/") {
		args[argKey] = filepath.Join(r.hostWorkDir, p[len(prefix)+1:])
	}
}

func (r *Registry) List() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

func DefaultRegistry(workDir string, dockerExec *sandbox.DockerExecutor) *Registry {
	reg := NewRegistry()
	reg.Register(&ReadFileTool{workDir: workDir})
	reg.Register(&WriteFileTool{workDir: workDir})
	reg.Register(&ListDirTool{workDir: workDir})
	reg.Register(&SearchFilesTool{workDir: workDir})
	reg.Register(&GrepTool{workDir: workDir})
	reg.Register(&ShellTool{workDir: workDir, docker: dockerExec})
	reg.Register(&StrReplaceTool{workDir: workDir})
	reg.Register(&InsertLineTool{workDir: workDir})
	reg.Register(&DoneTool{})
	return reg
}
