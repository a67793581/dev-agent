package sandbox

import "regexp"

// SandboxMode controls how strict the policy is.
type SandboxMode int

const (
	ModePermissive SandboxMode = iota // Only block RiskBlock
	ModeNormal                        // Block + High requires approval
	ModeStrict                        // Block + High/Medium require approval; read-only tools exempt
)

// ParseMode converts string to SandboxMode. Unknown values default to ModeNormal.
func ParseMode(s string) SandboxMode {
	switch s {
	case "permissive":
		return ModePermissive
	case "strict":
		return ModeStrict
	case "normal", "":
		return ModeNormal
	default:
		return ModeNormal
	}
}

// Policy holds workDir, shell policy, mode, and approval callback.
type Policy struct {
	Mode       SandboxMode
	WorkDir    string
	Shell      *ShellPolicy
	Path       *PathPolicy
	ApproveFunc func(action string) bool
}

// PathPolicy holds path deny list and allow_outside_workdir (used in path.go).
type PathPolicy struct {
	Deny                []string // Path prefixes to deny (after expand)
	AllowOutsideWorkdir []string // Paths outside workDir that are allowed
}

// Default path deny prefixes (sensitive paths). Config can append.
var DefaultPathDeny = []string{
	"/etc/",
	"~/.ssh/",
	"~/.aws/",
}

// Default shell block patterns (regex). Config block list is appended as patterns.
var defaultBlockPatterns = []string{
	`\bsudo\b`,
	`\brm\s+(-[^-\s]*)*\s*-rf\s+[\s/~]`,
	`\brm\s+-rf\s+/`,
	`\brm\s+-rf\s+~\b`,
	`\|\s*(sh|bash|zsh)\s*$`,
	`\|\s*(sh|bash|zsh)\s*\|`,
	`\bchmod\s+[0-7]{3,4}\s`,
	`\bchmod\s+[a-z+]*[0-7]{3,4}`,
	`\bchown\s`,
	`\bssh\s+[^\s]+`,
	`\bscp\s+[^\s]+`,
	`\brsync\s+[^\s]*@`,
	`>\s*/etc/`,
	`>\s*~/.ssh/`,
	`\bgit\s+push\s+[^|]*--force`,
	`\bdd\s+if=`,
	`\bmkfs\b`,
}

// Default shell approve patterns (High risk, need user confirmation in normal/strict).
var defaultApprovePatterns = []string{
	`\brm\s+-rf\s+`,
	`\bchmod\s+`,
	`\bchown\s`,
	`\bgit\s+push\b`,
	`\bnpm\s+publish\b`,
	`\bdocker\s+(rm|run)\s`,
}

func compilePatterns(strs []string) []*regexp.Regexp {
	var out []*regexp.Regexp
	for _, s := range strs {
		r, err := regexp.Compile(s)
		if err != nil {
			continue
		}
		out = append(out, r)
	}
	return out
}

// DefaultShellBlockPatterns returns compiled default block patterns.
func DefaultShellBlockPatterns() []*regexp.Regexp {
	return compilePatterns(defaultBlockPatterns)
}

// DefaultShellApprovePatterns returns compiled default approve patterns.
func DefaultShellApprovePatterns() []*regexp.Regexp {
	return compilePatterns(defaultApprovePatterns)
}
