package tools

import "fmt"

type Result struct {
	Success bool
	Output  string
}

type Tool interface {
	Name() string
	Execute(args map[string]string) Result
}

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) Execute(name string, args map[string]string) Result {
	tool, ok := r.tools[name]
	if !ok {
		return Result{
			Success: false,
			Output:  fmt.Sprintf("unknown command: %s", name),
		}
	}
	return tool.Execute(args)
}

func (r *Registry) List() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

func DefaultRegistry(workDir string) *Registry {
	reg := NewRegistry()
	reg.Register(&ReadFileTool{workDir: workDir})
	reg.Register(&WriteFileTool{workDir: workDir})
	reg.Register(&ListDirTool{workDir: workDir})
	reg.Register(&SearchFilesTool{workDir: workDir})
	reg.Register(&GrepTool{workDir: workDir})
	reg.Register(&ShellTool{workDir: workDir})
	reg.Register(&StrReplaceTool{workDir: workDir})
	reg.Register(&InsertLineTool{workDir: workDir})
	reg.Register(&DoneTool{})
	return reg
}
