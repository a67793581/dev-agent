package agent

import (
	"context"
	"devagent/internal/llm"
	"devagent/internal/parser"
	"devagent/internal/prompt"
	"devagent/internal/skill"
	"devagent/internal/tools"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxIterations  = 30
	maxRetries     = 3
	maxHistoryMsgs = 40
)

type Agent struct {
	client    *llm.Client
	registry  *tools.Registry
	workDir   string
	verbose   bool
	skillDirs []string

	messages   []llm.Message
	totalUsage llm.Usage
}

func New(client *llm.Client, workDir string, verbose bool, skillDirs []string) *Agent {
	return &Agent{
		client:    client,
		registry:  tools.DefaultRegistry(workDir),
		workDir:   workDir,
		verbose:   verbose,
		skillDirs: skillDirs,
	}
}

func (a *Agent) LLMClient() *llm.Client { return a.client }
func (a *Agent) Verbose() bool           { return a.verbose }

func (a *Agent) Run(ctx context.Context, task string) error {
	fileTree := a.buildFileTree(a.workDir, "", 0, 3)

	skills, err := skill.Discover(a.skillDirs)
	if err != nil {
		return fmt.Errorf("discover skills: %w", err)
	}
	if len(skills) > 0 {
		a.registry.Register(tools.NewReadSkillTool(skills))
	}

	meta := make([]prompt.SkillMeta, len(skills))
	for i := range skills {
		meta[i] = prompt.SkillMeta{Name: skills[i].Name, Description: skills[i].Description}
	}
	userContent := prompt.BuildProjectContext(a.workDir, fileTree) + "\n\n" + prompt.BuildUserTask(task) + prompt.BuildSkillsContext(meta)
	a.messages = []llm.Message{
		{Role: "system", Content: prompt.SystemPrompt},
		{Role: "user", Content: userContent},
	}

	fmt.Printf("\n🤖 DevAgent started (model: %s)\n", a.client.Model())
	fmt.Printf("📁 Project: %s\n", a.workDir)
	fmt.Printf("📋 Task: %s\n\n", task)

	for i := 0; i < maxIterations; i++ {
		fmt.Printf("━━━ Step %d/%d ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n", i+1, maxIterations)

		response, usage, err := a.callLLM(ctx)
		if err != nil {
			return fmt.Errorf("LLM call failed at step %d: %w", i+1, err)
		}
		a.totalUsage.PromptTokens += usage.PromptTokens
		a.totalUsage.CompletionTokens += usage.CompletionTokens
		a.totalUsage.TotalTokens += usage.TotalTokens

		a.messages = append(a.messages, llm.Message{Role: "assistant", Content: response})

		commands, thinking, err := parser.ParseCommands(response)
		if err != nil {
			fmt.Printf("⚠️  Parse error: %v\n", err)
			a.messages = append(a.messages, llm.Message{
				Role:    "user",
				Content: prompt.BuildObservation("parse_error", false, fmt.Sprintf("Failed to parse your command: %v\nPlease output a valid JSON command block.", err)),
			})
			continue
		}

		if thinking != "" && a.verbose {
			fmt.Printf("💭 Thinking: %s\n\n", truncate(thinking, 500))
		}

		if len(commands) == 0 {
			fmt.Printf("💬 %s\n\n", truncate(response, 1000))
			a.messages = append(a.messages, llm.Message{
				Role:    "user",
				Content: "You did not output a command. Please output a JSON command block to take action, or use the 'done' command if the task is complete.",
			})
			continue
		}

		for _, cmd := range commands {
			fmt.Printf("🔧 Command: %s\n", cmd.Name)
			if cmd.Reason != "" {
				fmt.Printf("   Reason: %s\n", cmd.Reason)
			}
			if a.verbose {
				for k, v := range cmd.Args {
					display := v
					if len(display) > 200 {
						display = display[:200] + "..."
					}
					fmt.Printf("   %s: %s\n", k, display)
				}
			}

			if cmd.Name == "done" {
				fmt.Printf("\n✅ Task completed!\n")
				fmt.Printf("   %s\n", cmd.Args["summary"])
				a.printUsage()
				return nil
			}

			if cmd.Name == "debug_code" {
				result := a.handleDebugCode(ctx, cmd.Args)
				fmt.Printf("   Status: %s\n\n", statusIcon(result.Success))
				a.messages = append(a.messages, llm.Message{
					Role:    "user",
					Content: prompt.BuildObservation(cmd.Name, result.Success, result.Output),
				})
				continue
			}

			result := a.registry.Execute(cmd.Name, cmd.Args)
			fmt.Printf("   Status: %s\n", statusIcon(result.Success))
			if a.verbose || !result.Success {
				fmt.Printf("   Output: %s\n", truncate(result.Output, 500))
			}
			fmt.Println()

			a.messages = append(a.messages, llm.Message{
				Role:    "user",
				Content: prompt.BuildObservation(cmd.Name, result.Success, result.Output),
			})
		}

		a.trimHistory()
	}

	a.printUsage()
	return fmt.Errorf("reached maximum iterations (%d) without completing the task", maxIterations)
}

func (a *Agent) callLLM(ctx context.Context) (string, llm.Usage, error) {
	var fullResp string
	var usage llm.Usage
	var err error

	for retry := 0; retry < maxRetries; retry++ {
		fullResp, usage, err = a.client.ChatStream(ctx, a.messages, func(chunk string) {
			if a.verbose {
				fmt.Print(chunk)
			}
		})
		if err == nil {
			if a.verbose {
				fmt.Println()
			}
			return fullResp, usage, nil
		}
		fmt.Printf("⚠️  LLM error (attempt %d/%d): %v\n", retry+1, maxRetries, err)
	}
	return "", usage, err
}

func (a *Agent) handleDebugCode(ctx context.Context, args map[string]string) tools.Result {
	code := args["code"]
	errorMsg := args["error"]
	testCode := args["test_code"]

	debugPrompt := prompt.BuildDebugPrompt(code, errorMsg, testCode)

	debugMsgs := []llm.Message{
		{Role: "system", Content: "You are an expert software engineer. Analyze the code and error, then provide the complete corrected code. Return ONLY the fixed code in a code block, no explanation needed."},
		{Role: "user", Content: debugPrompt},
	}

	resp, _, err := a.client.Chat(ctx, debugMsgs)
	if err != nil {
		return tools.Result{Success: false, Output: fmt.Sprintf("LLM debug call failed: %v", err)}
	}

	fixedCode := parser.ParseCodeBlock(resp, "")
	if fixedCode == "" {
		fixedCode = resp
	}

	return tools.Result{
		Success: true,
		Output:  fmt.Sprintf("## Suggested Fix\n\n```\n%s\n```", fixedCode),
	}
}

func (a *Agent) trimHistory() {
	if len(a.messages) <= maxHistoryMsgs {
		return
	}
	systemMsg := a.messages[0]
	firstUserMsg := a.messages[1]
	remaining := a.messages[len(a.messages)-(maxHistoryMsgs-2):]

	a.messages = make([]llm.Message, 0, maxHistoryMsgs)
	a.messages = append(a.messages, systemMsg, firstUserMsg)
	a.messages = append(a.messages, llm.Message{
		Role:    "user",
		Content: "[Earlier conversation history was trimmed to save context. Continue from where you left off.]",
	})
	a.messages = append(a.messages, remaining...)
}

func (a *Agent) printUsage() {
	fmt.Printf("\n📊 Token Usage: prompt=%d, completion=%d, total=%d\n",
		a.totalUsage.PromptTokens, a.totalUsage.CompletionTokens, a.totalUsage.TotalTokens)
}

func (a *Agent) buildFileTree(dir, prefix string, depth, maxDepth int) string {
	if depth >= maxDepth {
		return ""
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var sb strings.Builder
	skipDirs := map[string]bool{
		".git": true, "node_modules": true, "__pycache__": true,
		".venv": true, "vendor": true, ".idea": true, ".vscode": true,
		"dist": true, "build": true, ".next": true, "target": true,
	}

	filtered := make([]os.DirEntry, 0)
	for _, entry := range entries {
		if entry.IsDir() && skipDirs[entry.Name()] {
			continue
		}
		if strings.HasPrefix(entry.Name(), ".") && entry.Name() != "." {
			continue
		}
		filtered = append(filtered, entry)
	}

	for i, entry := range filtered {
		connector := "├── "
		childPrefix := prefix + "│   "
		if i == len(filtered)-1 {
			connector = "└── "
			childPrefix = prefix + "    "
		}

		if entry.IsDir() {
			sb.WriteString(fmt.Sprintf("%s%s%s/\n", prefix, connector, entry.Name()))
			sub := a.buildFileTree(filepath.Join(dir, entry.Name()), childPrefix, depth+1, maxDepth)
			sb.WriteString(sub)
		} else {
			sb.WriteString(fmt.Sprintf("%s%s%s\n", prefix, connector, entry.Name()))
		}
	}

	return sb.String()
}

func statusIcon(success bool) string {
	if success {
		return "✅"
	}
	return "❌"
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
