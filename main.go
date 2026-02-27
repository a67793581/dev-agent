package main

import (
	"context"
	"devagent/internal/agent"
	"devagent/internal/config"
	"devagent/internal/llm"
	"devagent/internal/prompt"
	"devagent/internal/sandbox"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/chzyer/readline"
	"github.com/mattn/go-runewidth"
)

var (
	version = "0.1.0"
)

func main() {
	envFile := flag.String("env", "", "Path to .env file (default: .env in current directory)")
	projectDir := flag.String("project", ".", "Path to the project directory")
	model := flag.String("model", "", "OpenAI model name (default: gpt-4o, or OPENAI_MODEL env)")
	baseURL := flag.String("base-url", "", "OpenAI API base URL (default: https://api.openai.com/v1, or OPENAI_BASE_URL env)")
	apiKey := flag.String("api-key", "", "OpenAI API key (default: OPENAI_API_KEY env)")
	verbose := flag.Bool("verbose", false, "Enable verbose output (show LLM streaming, tool details)")
	showVersion := flag.Bool("version", false, "Show version")
	taskFlag := flag.String("task", "", "Task to execute (if empty, enters interactive mode)")
	skillsFlag := flag.String("skills", "", "Comma-separated paths to additional skill directories (default: <project>/.devagent/skills and ~/.devagent/skills)")
	sandboxFlag := flag.String("sandbox", "normal", "Sandbox mode: permissive / normal / strict")
	noDockerFlag := flag.Bool("no-docker", false, "Disable Docker sandbox for shell commands")
	langFlag := flag.String("lang", "", "UI language: en / zh (default: auto-detect from LANG env)")
	soulFlag := flag.String("soul", "", "Path to custom soul/identity prompt file")
	guidelinesFlag := flag.String("guidelines", "", "Path to custom guidelines prompt file")

	flag.Usage = func() {
		lang := detectLang(*langFlag)
		if lang == "zh" {
			printUsageZh()
		} else {
			printUsageEn()
		}
	}

	flag.Parse()

	lang := detectLang(*langFlag)

	if *showVersion {
		fmt.Printf("DevAgent v%s\n", version)
		os.Exit(0)
	}

	if err := config.LoadEnv(*envFile); err != nil {
		fatalf("%v", err)
	}

	absProject, err := filepath.Abs(*projectDir)
	if err != nil {
		fatalf("invalid project path: %v", err)
	}

	info, err := os.Stat(absProject)
	if err != nil || !info.IsDir() {
		fatalf("project directory does not exist: %s", absProject)
	}

	key := *apiKey
	if key == "" {
		key = os.Getenv("OPENAI_API_KEY")
	}
	if key == "" {
		fatalf("OpenAI API key is required. Set OPENAI_API_KEY env or use -api-key flag.")
	}

	client := llm.NewClient(llm.Config{
		APIKey:  key,
		BaseURL: *baseURL,
		Model:   *model,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var dockerExec *sandbox.DockerExecutor
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n\n⚠️  Interrupted. Shutting down...")
		if dockerExec != nil {
			dockerExec.Stop()
		}
		cancel()
	}()

	sandboxCfg, err := sandbox.LoadConfig(absProject)
	if err != nil {
		log.Printf("Warning: loading sandbox config: %v (using defaults)", err)
		sandboxCfg = nil
	}
	cliMode := ""
	if *sandboxFlag != "" {
		cliMode = *sandboxFlag
	}
	interactive := *taskFlag == ""
	approveFunc := sandbox.ApproveFuncFor(interactive)
	sb := sandbox.NewSandboxFromConfig(absProject, sandboxCfg, cliMode, approveFunc)

	dockerCfg := sandbox.DockerConfig{}
	if sandboxCfg != nil {
		dockerCfg = sandboxCfg.Docker
	}
	if *noDockerFlag {
		enabled := false
		dockerCfg.Enabled = &enabled
	}
	if dockerCfg.DockerEnabled() {
		if sandbox.DockerAvailable() {
			dockerExec = sandbox.NewDockerExecutor(absProject, dockerCfg)
			fmt.Printf("🐳 Docker sandbox enabled (container: %s)\n", dockerExec.ContainerName())
		} else {
			fmt.Fprintln(os.Stderr, "⚠️  Docker not available, falling back to direct shell execution")
		}
	}

	skillDirs := buildSkillDirs(absProject, *skillsFlag)
	soul := prompt.ResolvePromptFile(*soulFlag, absProject, "SOUL.md")
	guidelines := prompt.ResolvePromptFile(*guidelinesFlag, absProject, "GUIDELINES.md")
	if *soulFlag != "" && soul == "" {
		fmt.Fprintf(os.Stderr, "⚠️  Soul file not found or unreadable: %s\n", *soulFlag)
	}
	if *guidelinesFlag != "" && guidelines == "" {
		fmt.Fprintf(os.Stderr, "⚠️  Guidelines file not found or unreadable: %s\n", *guidelinesFlag)
	}
	ag := agent.New(client, absProject, *verbose, skillDirs, soul, guidelines, sb, dockerExec)

	if *taskFlag != "" {
		err := ag.Run(ctx, *taskFlag)
		if dockerExec != nil {
			dockerExec.Stop()
		}
		if err != nil {
			fatalf("agent error: %v", err)
		}
		return
	}

	runInteractive(ctx, ag, absProject, skillDirs, soul, guidelines, sb, dockerExec, lang)
	if dockerExec != nil {
		dockerExec.Stop()
	}
}

func buildSkillDirs(projectDir, skillsFlag string) []string {
	// Priority: project-level, user-level, then custom (--skills)
	var dirs []string
	projectSkills := filepath.Join(projectDir, ".devagent", "skills")
	dirs = append(dirs, projectSkills)
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".devagent", "skills"))
	}
	if skillsFlag != "" {
		for _, p := range strings.Split(skillsFlag, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				dirs = append(dirs, p)
			}
		}
	}
	return dirs
}

func runInteractive(ctx context.Context, ag *agent.Agent, projectDir string, skillDirs []string, soul, guidelines string, sb *sandbox.Sandbox, dockerExec *sandbox.DockerExecutor, lang string) {
	if lang == "zh" {
		fmt.Printf(`
╔══════════════════════════════════════════════════╗
║          DevAgent v%s - 交互模式               ║
╠══════════════════════════════════════════════════╣
║  项目: %-39s ║
║                                                  ║
║  输入任务后按回车执行                              ║
║  输入 quit 或 exit 退出                           ║
║  输入 help 查看帮助                               ║
╚══════════════════════════════════════════════════╝
`, version, truncatePath(projectDir, 39))
	} else {
		fmt.Printf(`
╔══════════════════════════════════════════════════╗
║          DevAgent v%s - Interactive Mode        ║
╠══════════════════════════════════════════════════╣
║  Project: %-38s ║
║                                                  ║
║  Type your task and press Enter.                 ║
║  Type 'quit' or 'exit' to quit.                  ║
║  Type 'help' for available commands.             ║
╚══════════════════════════════════════════════════╝
`, version, truncatePath(projectDir, 38))
	}

	historyFile := ""
	if home, err := os.UserHomeDir(); err == nil {
		historyFile = filepath.Join(home, ".devagent_history")
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "🤖 > ",
		HistoryFile:     historyFile,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		log.Printf("Warning: readline init failed: %v, falling back to basic input", err)
		return
	}
	defer rl.Close()

	for {
		fmt.Println()
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt || err == io.EOF {
				if lang == "zh" {
					fmt.Println("再见!")
				} else {
					fmt.Println("Goodbye!")
				}
			}
			return
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		switch strings.ToLower(input) {
		case "quit", "exit", "q":
			if lang == "zh" {
				fmt.Println("再见!")
			} else {
				fmt.Println("Goodbye!")
			}
			return
		case "help", "h":
			printHelp(lang)
			continue
		}

		newAgent := agent.New(ag.LLMClient(), projectDir, ag.Verbose(), skillDirs, soul, guidelines, sb, dockerExec)

		if err := newAgent.Run(ctx, input); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		}
	}
}

func printHelp(lang string) {
	if lang == "zh" {
		fmt.Print(`
可用命令:
  help, h        显示帮助
  quit, exit, q  退出程序

任务示例:
  "分析项目结构并解释架构"
  "修复 main.go 中的错误处理问题"
  "为 utils 包添加单元测试"
  "重构数据库层，使用连接池"
  "安装 golangci-lint 并运行代码检查"
`)
	} else {
		fmt.Print(`
Available commands:
  help, h        Show this help
  quit, exit, q  Exit the program

Task examples:
  "Analyze the project structure and explain the architecture"
  "Fix the bug in main.go where the error handling is missing"
  "Add unit tests for the utils package"
  "Refactor the database layer to use connection pooling"
  "Install golangci-lint and run it on this project"
`)
	}
}

func detectLang(flagVal string) string {
	if flagVal != "" {
		if strings.HasPrefix(strings.ToLower(flagVal), "zh") {
			return "zh"
		}
		return "en"
	}
	envLang := os.Getenv("LANG")
	if strings.HasPrefix(strings.ToLower(envLang), "zh") {
		return "zh"
	}
	return "en"
}

func printUsageEn() {
	fmt.Fprintf(os.Stderr, `DevAgent v%s - AI-powered programming agent

Usage:
  devagent [flags]

Flags:
`, version)
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
Environment Variables (can be set in .env file):
  OPENAI_API_KEY    OpenAI API key (required)
  OPENAI_BASE_URL   API base URL (optional)
  OPENAI_MODEL      Model name (optional, default: gpt-4o)

Sandbox Modes:
  permissive   Block only dangerous commands (sudo, rm -rf /, etc.)
  normal       Block dangerous + require confirmation for high-risk operations (default)
  strict       Require confirmation for all write operations and medium/high-risk commands

Docker Sandbox:
  Shell commands run inside a persistent Docker container per project.
  The container is reused across commands and stopped on exit.
  Use -no-docker to disable, or configure in .devagent/sandbox.yaml.

Examples:
  devagent -project ./myapp -task "add error handling"
  devagent -project ./myapp                               # interactive mode
  devagent -project ./myapp -verbose                      # verbose output
  devagent -sandbox strict                                # strict sandbox
  devagent -no-docker                                     # disable Docker sandbox
  devagent -lang zh                                       # Chinese UI
  devagent -soul ./SOUL.md -guidelines ./GUIDELINES.md    # custom prompts
`)
}

func printUsageZh() {
	fmt.Fprintf(os.Stderr, `DevAgent v%s - AI 驱动的编程 Agent

用法:
  devagent [参数]

参数:
`, version)
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
环境变量 (可在 .env 文件中设置):
  OPENAI_API_KEY    OpenAI API 密钥 (必需)
  OPENAI_BASE_URL   API 基础 URL (可选)
  OPENAI_MODEL      模型名称 (可选, 默认: gpt-4o)

沙箱模式:
  permissive   仅拦截危险命令 (sudo, rm -rf / 等)
  normal       拦截危险命令 + 高风险操作需确认 (默认)
  strict       所有写操作和中/高风险命令均需确认

Docker 沙箱:
  Shell 命令在每个项目独立的持久 Docker 容器内执行。
  容器在命令间复用, 进程退出时停止 (下次自动恢复)。
  使用 -no-docker 禁用, 或在 .devagent/sandbox.yaml 中配置。

示例:
  devagent -project ./myapp -task "添加错误处理"
  devagent -project ./myapp                               # 交互模式
  devagent -project ./myapp -verbose                      # 详细输出
  devagent -sandbox strict                                # 严格沙箱
  devagent -no-docker                                     # 禁用 Docker 沙箱
  devagent -lang en                                       # 英文界面
  devagent -soul ./SOUL.md -guidelines ./GUIDELINES.md    # 自定义提示词
`)
}

func truncatePath(p string, maxWidth int) string {
	if runewidth.StringWidth(p) <= maxWidth {
		return p
	}
	runes := []rune(p)
	prefix := "..."
	prefixW := 3
	for i := len(runes) - 1; i >= 0; i-- {
		tail := string(runes[i:])
		if runewidth.StringWidth(tail)+prefixW > maxWidth {
			tail = string(runes[i+1:])
			w := runewidth.StringWidth(tail) + prefixW
			pad := ""
			if w < maxWidth {
				pad = strings.Repeat(" ", maxWidth-w)
			}
			return prefix + tail + pad
		}
	}
	return p
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}
