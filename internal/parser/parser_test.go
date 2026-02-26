package parser

import (
	"testing"
)

func TestParseCommands_SingleCommand(t *testing.T) {
	input := `<think>
I need to read the main.go file.
</think>

` + "```json" + `
{"command": "read_file", "args": {"path": "main.go"}, "reason": "check entry point"}
` + "```"

	cmds, thinking, err := ParseCommands(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if thinking != "I need to read the main.go file." {
		t.Errorf("thinking = %q, want %q", thinking, "I need to read the main.go file.")
	}
	if len(cmds) != 1 {
		t.Fatalf("got %d commands, want 1", len(cmds))
	}
	if cmds[0].Name != "read_file" {
		t.Errorf("command = %q, want %q", cmds[0].Name, "read_file")
	}
	if cmds[0].Args["path"] != "main.go" {
		t.Errorf("args[path] = %q, want %q", cmds[0].Args["path"], "main.go")
	}
	if cmds[0].Reason != "check entry point" {
		t.Errorf("reason = %q, want %q", cmds[0].Reason, "check entry point")
	}
}

func TestParseCommands_NoCommand(t *testing.T) {
	input := "This is just a plain text response without any commands."
	cmds, thinking, err := ParseCommands(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if thinking != "" {
		t.Errorf("thinking = %q, want empty", thinking)
	}
	if len(cmds) != 0 {
		t.Errorf("got %d commands, want 0", len(cmds))
	}
}

func TestParseCommands_ThinkingOnly(t *testing.T) {
	input := `<think>
Let me analyze the situation.
</think>

I'll look at the code now but I'm still deciding.`

	cmds, thinking, err := ParseCommands(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if thinking != "Let me analyze the situation." {
		t.Errorf("thinking = %q", thinking)
	}
	if len(cmds) != 0 {
		t.Errorf("got %d commands, want 0", len(cmds))
	}
}

func TestParseCommands_ShellCommand(t *testing.T) {
	input := `<think>
I need to install the linter.
</think>

` + "```json" + `
{"command": "shell", "args": {"command": "go install golang.org/x/lint/golint@latest"}, "reason": "install golint"}
` + "```"

	cmds, _, err := ParseCommands(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("got %d commands, want 1", len(cmds))
	}
	if cmds[0].Name != "shell" {
		t.Errorf("command = %q, want %q", cmds[0].Name, "shell")
	}
	if cmds[0].Args["command"] != "go install golang.org/x/lint/golint@latest" {
		t.Errorf("args[command] = %q", cmds[0].Args["command"])
	}
}

func TestParseCommands_DoneCommand(t *testing.T) {
	input := "```json\n" + `{"command": "done", "args": {"summary": "All tests pass"}, "reason": "task complete"}` + "\n```"

	cmds, _, err := ParseCommands(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("got %d commands, want 1", len(cmds))
	}
	if cmds[0].Name != "done" {
		t.Errorf("command = %q, want %q", cmds[0].Name, "done")
	}
	if cmds[0].Args["summary"] != "All tests pass" {
		t.Errorf("summary = %q", cmds[0].Args["summary"])
	}
}

func TestParseCommands_BrokenJSON_TrailingComma(t *testing.T) {
	input := "```json\n" + `{"command": "read_file", "args": {"path": "main.go"},}` + "\n```"

	cmds, _, err := ParseCommands(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("got %d commands, want 1", len(cmds))
	}
	if cmds[0].Name != "read_file" {
		t.Errorf("command = %q, want %q", cmds[0].Name, "read_file")
	}
}

func TestParseCommands_MissingClosingBrace(t *testing.T) {
	input := "```json\n" + `{"command": "shell", "args": {"command": "ls"}` + "\n```"

	cmds, _, err := ParseCommands(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("got %d commands, want 1", len(cmds))
	}
}

func TestParseCodeBlock(t *testing.T) {
	tests := []struct {
		name string
		text string
		lang string
		want string
	}{
		{
			name: "python block",
			text: "Here is the fix:\n```python\ndef hello():\n    print('hello')\n```",
			lang: "python",
			want: "def hello():\n    print('hello')",
		},
		{
			name: "any language",
			text: "```go\npackage main\n```",
			lang: "",
			want: "package main",
		},
		{
			name: "no block",
			text: "no code here",
			lang: "go",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseCodeBlock(tt.text, tt.lang)
			if got != tt.want {
				t.Errorf("ParseCodeBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseSections(t *testing.T) {
	input := `## Summary
This is a summary.

## Details
Here are some details.
More details here.

## Status
PASS`

	sections := ParseSections(input)
	if len(sections) != 3 {
		t.Fatalf("got %d sections, want 3", len(sections))
	}
	if _, ok := sections["Summary"]; !ok {
		t.Error("missing Summary section")
	}
	if _, ok := sections["Status"]; !ok {
		t.Error("missing Status section")
	}
}

func TestExtractTextBetweenTags(t *testing.T) {
	tests := []struct {
		text     string
		open     string
		close    string
		expected string
	}{
		{"<think>hello world</think>", "<think>", "</think>", "hello world"},
		{"no tags here", "<think>", "</think>", ""},
		{"<think>unclosed tag", "<think>", "</think>", "unclosed tag"},
	}
	for _, tt := range tests {
		got := ExtractTextBetweenTags(tt.text, tt.open, tt.close)
		if got != tt.expected {
			t.Errorf("ExtractTextBetweenTags(%q) = %q, want %q", tt.text, got, tt.expected)
		}
	}
}

func TestRepairJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"trailing comma", `{"key": "value",}`},
		{"missing brace", `{"key": "value"`},
		{"clean json", `{"key": "value"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repairJSON(tt.input)
			if result == "" {
				t.Error("repairJSON returned empty string")
			}
		})
	}
}

func TestParseCommands_WriteFileWithNewlines(t *testing.T) {
	input := "```json\n" + `{"command": "write_file", "args": {"path": "test.py", "content": "line1\nline2\nline3"}, "reason": "create test file"}` + "\n```"

	cmds, _, err := ParseCommands(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("got %d commands, want 1", len(cmds))
	}
	if cmds[0].Name != "write_file" {
		t.Errorf("command = %q, want %q", cmds[0].Name, "write_file")
	}
}
