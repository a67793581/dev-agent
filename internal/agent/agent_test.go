package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"devagent/internal/llm"
	"devagent/internal/sandbox"
)

func TestNew(t *testing.T) {
	client := llm.NewClient(llm.Config{APIKey: "test"})
	workDir := t.TempDir()
	a := New(client, workDir, false, nil, "", "", nil, nil)
	if a == nil {
		t.Fatal("New should not return nil")
	}
	if a.client != client {
		t.Error("client not set")
	}
	if a.workDir != workDir {
		t.Error("workDir not set")
	}
	if a.displayDir != workDir {
		t.Error("displayDir should equal workDir when no docker")
	}
}

func TestNew_WithDocker(t *testing.T) {
	client := llm.NewClient(llm.Config{APIKey: "test"})
	workDir := t.TempDir()
	docker := sandbox.NewDockerExecutor(workDir, sandbox.DockerConfig{})
	a := New(client, workDir, false, nil, "", "", nil, docker)
	if a.displayDir != "/workspace" {
		t.Errorf("displayDir = %q, want /workspace", a.displayDir)
	}
}

func TestNew_WithSandbox(t *testing.T) {
	client := llm.NewClient(llm.Config{APIKey: "test"})
	workDir := t.TempDir()
	policy := &sandbox.Policy{WorkDir: workDir, Shell: &sandbox.ShellPolicy{}, Path: &sandbox.PathPolicy{}}
	sb := sandbox.NewSandbox(policy)
	a := New(client, workDir, false, nil, "", "", sb, nil)
	if a == nil {
		t.Fatal("New with sandbox should not return nil")
	}
}

func TestAgent_LLMClient(t *testing.T) {
	client := llm.NewClient(llm.Config{APIKey: "k"})
	a := New(client, t.TempDir(), false, nil, "", "", nil, nil)
	if a.LLMClient() != client {
		t.Error("LLMClient() should return same client")
	}
}

func TestAgent_Verbose(t *testing.T) {
	client := llm.NewClient(llm.Config{APIKey: "k"})
	a := New(client, t.TempDir(), true, nil, "", "", nil, nil)
	if !a.Verbose() {
		t.Error("Verbose() should be true")
	}
	a2 := New(client, t.TempDir(), false, nil, "", "", nil, nil)
	if a2.Verbose() {
		t.Error("Verbose() should be false")
	}
}

func TestAgent_Run_ParseErrorThenDone(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "text/event-stream")
		if callCount == 1 {
			// First response: invalid (no valid command block), agent will send parse_error observation
			w.Write([]byte("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"just text\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":1,\"total_tokens\":2}}\n\n"))
		} else {
			// Second: done
			escaped := "<think>Ok.</think>\\n\\n```json\\n{\\\"command\\\": \\\"done\\\", \\\"args\\\": {\\\"summary\\\": \\\"ok\\\"}, \\\"reason\\\": \\\"\\\"}\\n```"
			w.Write([]byte("data: {\"id\":\"2\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"" + escaped + "\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":2,\"total_tokens\":3}}\n\n"))
		}
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()
	client := llm.NewClient(llm.Config{APIKey: "test", BaseURL: server.URL})
	workDir := t.TempDir()
	os.WriteFile(filepath.Join(workDir, "main.go"), []byte("package main"), 0644)
	a := New(client, workDir, false, nil, "", "", nil, nil)
	ctx := context.Background()
	err := a.Run(ctx, "task")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if callCount < 2 {
		t.Errorf("expected at least 2 LLM calls, got %d", callCount)
	}
}

func TestAgent_Run_ToolFailsThenDone(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "text/event-stream")
		if callCount == 1 {
			// read_file nonexistent -> tool fails -> statusIcon(false)
			escaped := "<think>Read file.</think>\\n\\n```json\\n{\\\"command\\\": \\\"read_file\\\", \\\"args\\\": {\\\"path\\\": \\\"nonexistent.txt\\\"}, \\\"reason\\\": \\\"\\\"}\\n```"
			w.Write([]byte("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"" + escaped + "\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":2,\"total_tokens\":3}}\n\n"))
		} else {
			escaped := "<think>Ok.</think>\\n\\n```json\\n{\\\"command\\\": \\\"done\\\", \\\"args\\\": {\\\"summary\\\": \\\"ok\\\"}, \\\"reason\\\": \\\"\\\"}\\n```"
			w.Write([]byte("data: {\"id\":\"2\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"" + escaped + "\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":2,\"total_tokens\":3}}\n\n"))
		}
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()
	client := llm.NewClient(llm.Config{APIKey: "test", BaseURL: server.URL})
	workDir := t.TempDir()
	a := New(client, workDir, false, nil, "", "", nil, nil)
	ctx := context.Background()
	err := a.Run(ctx, "task")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestAgent_Run_CompletesWithDone(t *testing.T) {
	// Mock LLM returns one stream chunk with done command (content must be valid JSON-escaped).
	escaped := "<think>Done.</think>\\n\\n```json\\n{\\\"command\\\": \\\"done\\\", \\\"args\\\": {\\\"summary\\\": \\\"ok\\\"}, \\\"reason\\\": \\\"\\\"}\\n```"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"" + escaped + "\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":2,\"total_tokens\":3}}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := llm.NewClient(llm.Config{APIKey: "test", BaseURL: server.URL})
	workDir := t.TempDir()
	os.WriteFile(filepath.Join(workDir, "main.go"), []byte("package main"), 0644)
	a := New(client, workDir, false, nil, "", "", nil, nil)
	ctx := context.Background()
	err := a.Run(ctx, "test task")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestAgent_Run_DebugCodeThenDone(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// Detect stream vs non-stream by reading body
		var body struct {
			Stream bool `json:"stream"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			if callCount == 1 {
				// debug_code command -> agent will call Chat (non-stream) for fix
				escaped := "<think>Debug.</think>\\n\\n```json\\n{\\\"command\\\": \\\"debug_code\\\", \\\"args\\\": {\\\"code\\\": \\\"x\\\", \\\"error\\\": \\\"err\\\", \\\"test_code\\\": \\\"\\\"}, \\\"reason\\\": \\\"\\\"}\\n```"
				w.Write([]byte("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"" + escaped + "\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":2,\"total_tokens\":3}}\n\n"))
			} else {
				escaped := "<think>Ok.</think>\\n\\n```json\\n{\\\"command\\\": \\\"done\\\", \\\"args\\\": {\\\"summary\\\": \\\"ok\\\"}, \\\"reason\\\": \\\"\\\"}\\n```"
				w.Write([]byte("data: {\"id\":\"2\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"" + escaped + "\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":2,\"total_tokens\":3}}\n\n"))
			}
			w.Write([]byte("data: [DONE]\n\n"))
		} else {
			// Non-stream: handleDebugCode Chat call - return fixed code in code block
			w.Header().Set("Content-Type", "application/json")
			body := `{"id":"debug","choices":[{"message":{"role":"assistant","content":"` + "```go\nfixed code\n```" + `"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`
			w.Write([]byte(body))
		}
	}))
	defer server.Close()
	client := llm.NewClient(llm.Config{APIKey: "test", BaseURL: server.URL})
	workDir := t.TempDir()
	a := New(client, workDir, false, nil, "", "", nil, nil)
	ctx := context.Background()
	err := a.Run(ctx, "task")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}
