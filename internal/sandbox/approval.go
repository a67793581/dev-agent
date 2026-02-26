package sandbox

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

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
