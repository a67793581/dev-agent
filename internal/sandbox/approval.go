package sandbox

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

// ApproveFuncFor returns an approval function suitable for the run mode.
// When interactive is true (e.g. no -task), it uses TerminalApproval so the user can confirm in the terminal.
// When interactive is false (-task "..." mode), it denies by default and logs the action so the run does not block on stdin.
func ApproveFuncFor(interactive bool) func(action string) bool {
	if interactive {
		return TerminalApproval()
	}
	return func(action string) bool {
		log.Printf("Sandbox approval required (non-interactive, denying): %s", strings.TrimSpace(action))
		return false
	}
}

// TerminalApproval returns an ApproveFunc that prints the action prompt to stdout
// and reads a line from stdin; it returns true only for "y" or "yes" (case-insensitive).
func TerminalApproval() func(action string) bool {
	return func(action string) bool {
		fmt.Print(action)
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			return false
		}
		t := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return t == "y" || t == "yes"
	}
}
