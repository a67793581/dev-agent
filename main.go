package main

import (
	"bufio"
	"context"
	"devagent/internal/agent"
	"devagent/internal/config"
	"devagent/internal/llm"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
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

	flag.Usage = func() {
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

.env file lookup order (first found wins, existing env vars are never overwritten):
  1. File specified by -env flag
  2. .env in current working directory
  3. ~/.devagent.env in home directory

Examples:
  devagent -project ./myapp -task "add error handling to all API endpoints"
  devagent -project ./myapp                       # interactive mode
  devagent -project ./myapp -verbose              # verbose output
  devagent -env /path/to/.env -project ./myapp    # custom env file
`)
	}

	flag.Parse()

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

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n\n⚠️  Interrupted. Shutting down...")
		cancel()
	}()

	ag := agent.New(client, absProject, *verbose)

	if *taskFlag != "" {
		if err := ag.Run(ctx, *taskFlag); err != nil {
			fatalf("agent error: %v", err)
		}
		return
	}

	runInteractive(ctx, ag, absProject)
}

func runInteractive(ctx context.Context, ag *agent.Agent, projectDir string) {
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

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\n🤖 > ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch strings.ToLower(input) {
		case "quit", "exit", "q":
			fmt.Println("Goodbye!")
			return
		case "help", "h":
			printHelp()
			continue
		}

		newAgent := agent.New(
			ag.LLMClient(),
			projectDir,
			ag.Verbose(),
		)

		if err := newAgent.Run(ctx, input); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		}
	}
}

func printHelp() {
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

func truncatePath(p string, maxLen int) string {
	if len(p) <= maxLen {
		return p
	}
	return "..." + p[len(p)-maxLen+3:]
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}
