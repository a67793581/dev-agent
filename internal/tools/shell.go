package tools

import (
	"bytes"
	"context"
	"devagent/internal/sandbox"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type ShellTool struct {
	workDir  string
	docker   *sandbox.DockerExecutor // nil means direct execution
}

func (t *ShellTool) Name() string { return "shell" }

func (t *ShellTool) Execute(args map[string]string) Result {
	command := args["command"]
	if command == "" {
		return Result{Success: false, Output: "empty command"}
	}

	if t.docker != nil {
		return t.executeDocker(command)
	}
	return t.executeDirect(command)
}

func (t *ShellTool) executeDocker(command string) Result {
	output, exitCode, err := t.docker.Execute(command)
	output = truncateOutput(output)

	if err != nil {
		return Result{Success: false, Output: fmt.Sprintf("[docker] %v\n%s", err, output)}
	}
	if exitCode != 0 {
		return Result{Success: false, Output: fmt.Sprintf("[docker] exit code: %d\n%s", exitCode, output)}
	}
	if output == "" {
		output = "(no output)"
	}
	return Result{Success: true, Output: output}
}

func (t *ShellTool) executeDirect(command string) Result {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = t.workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	var sb strings.Builder
	if stdout.Len() > 0 {
		sb.WriteString(stdout.String())
	}
	if stderr.Len() > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("[stderr]\n")
		sb.WriteString(stderr.String())
	}

	output := truncateOutput(sb.String())

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return Result{Success: false, Output: fmt.Sprintf("command timed out after 5 minutes\n%s", output)}
		}
		return Result{Success: false, Output: fmt.Sprintf("exit code: %v\n%s", err, output)}
	}

	if output == "" {
		output = "(no output)"
	}
	return Result{Success: true, Output: output}
}

func truncateOutput(output string) string {
	const maxLen = 16000
	if len(output) > maxLen {
		half := maxLen / 2
		output = output[:half] + "\n\n... (output truncated) ...\n\n" + output[len(output)-half:]
	}
	return output
}

type GrepTool struct {
	workDir string
}

func (t *GrepTool) Name() string { return "grep" }

func (t *GrepTool) Execute(args map[string]string) Result {
	pattern := args["pattern"]
	if pattern == "" {
		return Result{Success: false, Output: "empty pattern"}
	}

	path := args["path"]
	if path == "" || path == "." {
		path = t.workDir
	}

	shell := &ShellTool{workDir: t.workDir}
	grepCmd := fmt.Sprintf("rg --no-heading -n --max-count=100 '%s' '%s' 2>/dev/null || grep -rn --max-count=100 '%s' '%s' 2>/dev/null",
		pattern, path, pattern, path)
	return shell.Execute(map[string]string{"command": grepCmd})
}

type DoneTool struct{}

func (t *DoneTool) Name() string { return "done" }

func (t *DoneTool) Execute(args map[string]string) Result {
	summary := args["summary"]
	if summary == "" {
		summary = "Task completed."
	}
	return Result{Success: true, Output: summary}
}
