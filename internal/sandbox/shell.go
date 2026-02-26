package sandbox

import (
	"regexp"
	"strings"
)

// CommandRisk level for shell commands.
type CommandRisk int

const (
	RiskLow CommandRisk = iota
	RiskMedium
	RiskHigh
	RiskBlock
)

// ShellPolicy holds block/approve/allow regex lists.
type ShellPolicy struct {
	BlockPatterns   []*regexp.Regexp
	ApprovePatterns []*regexp.Regexp
	AllowPatterns   []*regexp.Regexp
}

// Evaluate returns the risk level for the given command string.
// Allow wins over block/approve; then block; then approve; then Low.
func (p *ShellPolicy) Evaluate(command string) CommandRisk {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return RiskLow
	}
	for _, r := range p.AllowPatterns {
		if r != nil && r.MatchString(cmd) {
			return RiskLow
		}
	}
	for _, r := range p.BlockPatterns {
		if r != nil && r.MatchString(cmd) {
			return RiskBlock
		}
	}
	for _, r := range p.ApprovePatterns {
		if r != nil && r.MatchString(cmd) {
			return RiskHigh
		}
	}
	return RiskLow
}

// GlobLikeToRegex converts a simple glob-like pattern (e.g. "sudo *", "go test *") into a regex.
// * is treated as .*
func GlobLikeToRegex(pattern string) string {
	s := regexp.QuoteMeta(pattern)
	s = strings.ReplaceAll(s, `\*`, ".*")
	return "(?i)" + s
}
