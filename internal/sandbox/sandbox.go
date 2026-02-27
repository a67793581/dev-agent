package sandbox

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrBlocked       = errors.New("blocked by sandbox policy")
	ErrNeedsApproval = errors.New("operation requires user approval")
)

// CheckResult is the return value of Sandbox.Check.
type CheckResult struct {
	Allow          bool   // If true, caller may execute.
	DenyErr        error  // If set, operation is blocked (Allow is false).
	ApprovalAction string // If set and Allow is false, caller should prompt user with this; if user approves, then allow.
}

// Sandbox runs path + shell checks and applies policy mode.
type Sandbox struct {
	policy *Policy
}

// NewSandbox builds a Sandbox from a Policy (default rules only).
func NewSandbox(policy *Policy) *Sandbox {
	return &Sandbox{policy: policy}
}

// NewSandboxFromConfig builds a Sandbox merging default rules with optional Config.
// CLI mode overrides config mode when non-empty.
func NewSandboxFromConfig(workDir string, cfg *Config, cliMode string, approveFunc func(action string) bool) *Sandbox {
	mode := ModeNormal
	if cliMode != "" {
		mode = ParseMode(cliMode)
	} else if cfg != nil && cfg.Mode != "" {
		mode = ParseMode(cfg.Mode)
	}

	shellPolicy := &ShellPolicy{
		BlockPatterns:   DefaultShellBlockPatterns(),
		ApprovePatterns: DefaultShellApprovePatterns(),
		AllowPatterns:   nil,
	}
	if cfg != nil && cfg.Shell.Block != nil {
		for _, s := range cfg.Shell.Block {
			r, err := regexpFromGlobLike(s)
			if err == nil {
				shellPolicy.BlockPatterns = append(shellPolicy.BlockPatterns, r)
			}
		}
	}
	if cfg != nil && cfg.Shell.Approve != nil {
		for _, s := range cfg.Shell.Approve {
			r, err := regexpFromGlobLike(s)
			if err == nil {
				shellPolicy.ApprovePatterns = append(shellPolicy.ApprovePatterns, r)
			}
		}
	}
	if cfg != nil && cfg.Shell.Allow != nil {
		for _, s := range cfg.Shell.Allow {
			r, err := regexpFromGlobLike(s)
			if err == nil {
				shellPolicy.AllowPatterns = append(shellPolicy.AllowPatterns, r)
			}
		}
	}

	pathDeny := append([]string{}, DefaultPathDeny...)
	var pathAllowOutside []string
	if cfg != nil {
		if cfg.Paths.Deny != nil {
			for _, p := range cfg.Paths.Deny {
				pathDeny = append(pathDeny, p)
			}
		}
		if cfg.Paths.AllowOutsideWorkdir != nil {
			pathAllowOutside = append(pathAllowOutside, cfg.Paths.AllowOutsideWorkdir...)
		}
	}
	pathPolicy := &PathPolicy{
		Deny:                pathDeny,
		AllowOutsideWorkdir: pathAllowOutside,
	}

	policy := &Policy{
		Mode:       mode,
		WorkDir:    workDir,
		Shell:      shellPolicy,
		Path:       pathPolicy,
		ApproveFunc: approveFunc,
	}
	return NewSandbox(policy)
}

func regexpFromGlobLike(s string) (*regexp.Regexp, error) {
	return regexp.Compile(GlobLikeToRegex(strings.TrimSpace(s)))
}

// Tool names that are read-only (no approval in strict mode).
var readOnlyTools = map[string]bool{
	"read_file": true, "list_dir": true, "search_files": true, "grep": true,
}

// Tools that take a path argument (for path validation).
var pathTools = map[string]string{
	"read_file": "path", "write_file": "path", "str_replace": "path", "insert_line": "path",
	"list_dir": "path", "search_files": "path", "grep": "path",
}

// Check runs policy: path validation for path tools, shell risk for shell tool, and mode-based allow/approve/deny.
func (s *Sandbox) Check(toolName string, args map[string]string) CheckResult {
	if s.policy == nil {
		return CheckResult{Allow: true}
	}

	// Path tools: validate path
	if pathArg, ok := pathTools[toolName]; ok {
		path := args[pathArg]
		if path == "" && (toolName == "list_dir" || toolName == "search_files" || toolName == "grep") {
			path = "."
		}
		if path != "" {
			if err := ValidatePath(s.policy.WorkDir, path, s.policy.Path); err != nil {
				return CheckResult{Allow: false, DenyErr: fmt.Errorf("%w: %v", ErrBlocked, err)}
			}
		}
	}

	// Read-only tools: in strict mode no approval needed
	if s.policy.Mode == ModeStrict && readOnlyTools[toolName] {
		return CheckResult{Allow: true}
	}

	// Shell: evaluate risk
	if toolName == "shell" {
		cmd := args["command"]
		if cmd == "" {
			return CheckResult{Allow: false, DenyErr: fmt.Errorf("%w: empty command", ErrBlocked)}
		}
		risk := s.policy.Shell.Evaluate(cmd)
		switch risk {
		case RiskBlock:
			return CheckResult{Allow: false, DenyErr: fmt.Errorf("%w: dangerous command", ErrBlocked)}
		case RiskHigh:
			if s.policy.Mode == ModePermissive {
				return CheckResult{Allow: true}
			}
			action := fmt.Sprintf("⚠️  Agent wants to execute: %s\n   Risk: HIGH\n   Allow? [y/N]: ", truncateForPrompt(cmd, 200))
			if s.policy.Mode == ModeStrict {
				if s.policy.ApproveFunc != nil && s.policy.ApproveFunc(action) {
					return CheckResult{Allow: true}
				}
				return CheckResult{Allow: false, DenyErr: ErrNeedsApproval, ApprovalAction: action}
			}
			// ModeNormal
			if s.policy.ApproveFunc != nil && s.policy.ApproveFunc(action) {
				return CheckResult{Allow: true}
			}
			return CheckResult{Allow: false, DenyErr: ErrNeedsApproval, ApprovalAction: action}
		case RiskMedium:
			if s.policy.Mode == ModeStrict {
				action := fmt.Sprintf("⚠️  Agent wants to execute: %s\n   Risk: MEDIUM\n   Allow? [y/N]: ", truncateForPrompt(cmd, 200))
				if s.policy.ApproveFunc != nil && s.policy.ApproveFunc(action) {
					return CheckResult{Allow: true}
				}
				return CheckResult{Allow: false, DenyErr: ErrNeedsApproval, ApprovalAction: action}
			}
			return CheckResult{Allow: true}
		default:
			return CheckResult{Allow: true}
		}
	}

	// Other tools (write_file, str_replace, insert_line, done, read_skill, debug_code): strict mode may require approval
	if s.policy.Mode == ModeStrict && !readOnlyTools[toolName] && toolName != "done" && toolName != "read_skill" && toolName != "debug_code" {
		path := args["path"]
		action := fmt.Sprintf("⚠️  Agent wants to run: %s (path: %s)\n   Allow? [y/N]: ", toolName, path)
		if s.policy.ApproveFunc != nil && s.policy.ApproveFunc(action) {
			return CheckResult{Allow: true}
		}
		return CheckResult{Allow: false, DenyErr: ErrNeedsApproval, ApprovalAction: action}
	}

	return CheckResult{Allow: true}
}

func truncateForPrompt(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
