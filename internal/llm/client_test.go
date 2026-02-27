package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient_Defaults(t *testing.T) {
	c := NewClient(Config{APIKey: "test-key"})
	if c == nil {
		t.Fatal("NewClient should not return nil")
	}
	if c.apiKey != "test-key" {
		t.Errorf("apiKey = %q", c.apiKey)
	}
	if c.baseURL != "https://api.openai.com/v1" {
		t.Errorf("baseURL = %q", c.baseURL)
	}
	if c.model != "gpt-4o" {
		t.Errorf("model = %q", c.model)
	}
	if c.httpClient.Timeout != 120*time.Second {
		t.Errorf("timeout = %v", c.httpClient.Timeout)
	}
}

func TestNewClient_ExplicitConfig(t *testing.T) {
	c := NewClient(Config{
		APIKey:  "key",
		BaseURL: "https://custom.example/v1",
		Model:   "gpt-4",
		Timeout: 10 * time.Second,
	})
	if c.baseURL != "https://custom.example/v1" {
		t.Errorf("baseURL = %q", c.baseURL)
	}
	if c.model != "gpt-4" {
		t.Errorf("model = %q", c.model)
	}
	if c.httpClient.Timeout != 10*time.Second {
		t.Errorf("timeout = %v", c.httpClient.Timeout)
	}
}

func TestClient_Model(t *testing.T) {
	c := NewClient(Config{APIKey: "k", Model: "gpt-4o-mini"})
	if c.Model() != "gpt-4o-mini" {
		t.Errorf("Model() = %q", c.Model())
	}
}

func TestClient_Chat_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"1","choices":[{"index":0,"message":{"role":"assistant","content":"Hello"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		APIKey:  "test",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})
	ctx := context.Background()
	content, usage, err := client.Chat(ctx, []Message{{Role: "user", Content: "Hi"}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if content != "Hello" {
		t.Errorf("content = %q", content)
	}
	if usage.TotalTokens != 2 {
		t.Errorf("usage.TotalTokens = %d", usage.TotalTokens)
	}
}

func TestClient_Chat_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid request"}`))
	}))
	defer server.Close()

	client := NewClient(Config{APIKey: "test", BaseURL: server.URL, Timeout: 5 * time.Second})
	ctx := context.Background()
	_, _, err := client.Chat(ctx, []Message{{Role: "user", Content: "Hi"}})
	if err == nil {
		t.Fatal("expected error on 400")
	}
}

func TestClient_Chat_NoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"1","choices":[],"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`))
	}))
	defer server.Close()

	client := NewClient(Config{APIKey: "test", BaseURL: server.URL, Timeout: 5 * time.Second})
	ctx := context.Background()
	_, _, err := client.Chat(ctx, []Message{{Role: "user", Content: "Hi"}})
	if err == nil {
		t.Fatal("expected error when no choices")
	}
}

func TestClient_ChatStream_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hi\"},\"finish_reason\":null}]}\n\n"))
		w.Write([]byte("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":2,\"total_tokens\":3}}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := NewClient(Config{APIKey: "test", BaseURL: server.URL, Timeout: 5 * time.Second})
	ctx := context.Background()
	var chunks []string
	content, usage, err := client.ChatStream(ctx, []Message{{Role: "user", Content: "Hi"}}, func(s string) { chunks = append(chunks, s) })
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	if content != "Hi" {
		t.Errorf("content = %q", content)
	}
	if len(chunks) != 1 || chunks[0] != "Hi" {
		t.Errorf("chunks = %v", chunks)
	}
	if usage.TotalTokens != 3 {
		t.Errorf("usage.TotalTokens = %d", usage.TotalTokens)
	}
}

func TestClient_ChatStream_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	}))
	defer server.Close()

	client := NewClient(Config{APIKey: "test", BaseURL: server.URL, Timeout: 5 * time.Second})
	ctx := context.Background()
	_, _, err := client.ChatStream(ctx, []Message{{Role: "user", Content: "Hi"}}, nil)
	if err == nil {
		t.Fatal("expected error on 401")
	}
}

func TestClient_ChatStream_NilOnChunk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hi\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":1,\"total_tokens\":2}}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()
	client := NewClient(Config{APIKey: "test", BaseURL: server.URL, Timeout: 5 * time.Second})
	ctx := context.Background()
	content, _, err := client.ChatStream(ctx, []Message{{Role: "user", Content: "Hi"}}, nil)
	if err != nil {
		t.Fatalf("ChatStream with nil onChunk: %v", err)
	}
	if content != "Hi" {
		t.Errorf("content = %q", content)
	}
}
