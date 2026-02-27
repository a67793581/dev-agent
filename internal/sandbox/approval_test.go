package sandbox

import (
	"strings"
	"testing"
)

func TestApproveFuncFor_NonInteractive(t *testing.T) {
	fn := ApproveFuncFor(false)
	if fn == nil {
		t.Fatal("ApproveFuncFor(false) should return non-nil")
	}
	// Non-interactive: always deny
	if fn("Allow? [y/N]: ") {
		t.Error("non-interactive should deny")
	}
}

func TestApproveFuncFor_Interactive(t *testing.T) {
	fn := ApproveFuncFor(true)
	if fn == nil {
		t.Fatal("ApproveFuncFor(true) should return non-nil")
	}
	// TerminalApproval reads from stdin; we don't feed stdin here, so Scan() fails -> false
	if fn("Allow? [y/N]: ") {
		t.Error("with no stdin, interactive approval should return false")
	}
}

func TestTerminalApproval(t *testing.T) {
	// TerminalApproval returns a function that reads from os.Stdin.
	// Without feeding stdin, scanner.Scan() returns false -> false
	fn := TerminalApproval()
	if fn("prompt") {
		t.Error("TerminalApproval with no stdin should return false")
	}
}

func TestApproveFuncFor_NonInteractive_LogsAction(t *testing.T) {
	// Just ensure the returned function doesn't panic when given action with spaces
	fn := ApproveFuncFor(false)
	action := "  Allow dangerous command? [y/N]:  "
	result := fn(action)
	if result {
		t.Error("should deny")
	}
	_ = strings.TrimSpace(action)
}
