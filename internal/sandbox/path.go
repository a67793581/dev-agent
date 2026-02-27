package sandbox

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// ValidatePath ensures targetPath (relative to workDir or absolute) resolves inside workDir.
// It uses path relativity to detect escape so that non-existent paths (e.g. new files) are allowed.
// Optionally respects PathPolicy deny/allow lists.
func ValidatePath(workDir, targetPath string, pathPolicy *PathPolicy) error {
	resolved := resolvePath(workDir, targetPath)
	absWork, err := filepath.Abs(workDir)
	if err != nil {
		return fmt.Errorf("workDir: %w", err)
	}
	absResolved, err := filepath.Abs(resolved)
	if err != nil {
		return fmt.Errorf("path %q: %w", targetPath, err)
	}
	// Rel(workDir, resolved) must not escape (no ".." prefix)
	rel, err := filepath.Rel(absWork, absResolved)
	if err != nil {
		return fmt.Errorf("path %q escapes sandbox (workDir: %s)", targetPath, absWork)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		// Allow if in allow_outside_workdir
		if pathPolicy != nil && len(pathPolicy.AllowOutsideWorkdir) > 0 {
			for _, allowed := range pathPolicy.AllowOutsideWorkdir {
				allowedAbs := expandHome(allowed)
				allowedAbs, _ = filepath.Abs(allowedAbs)
				if allowedAbs != "" && (absResolved == allowedAbs || strings.HasPrefix(absResolved, allowedAbs+string(filepath.Separator))) {
					return nil
				}
			}
		}
		return fmt.Errorf("path %q escapes sandbox (workDir: %s)", targetPath, absWork)
	}
	// Resolve symlinks for deny check so that paths under workDir that are symlinks to
	// sensitive dirs (e.g. project/link_to_etc -> /etc) are still denied.
	canonicalForDeny := absResolved
	if resolved, err := filepath.EvalSymlinks(absResolved); err == nil {
		canonicalForDeny = resolved
	}
	// Deny sensitive paths even inside workDir (e.g. project contains /etc symlink)
	if pathPolicy != nil && len(pathPolicy.Deny) > 0 {
		for _, deny := range pathPolicy.Deny {
			denyAbs := expandHome(deny)
			denyAbs, _ = filepath.Abs(denyAbs)
			if denyAbs != "" && (canonicalForDeny == denyAbs || strings.HasPrefix(canonicalForDeny, denyAbs+string(filepath.Separator))) {
				return fmt.Errorf("path %q is denied by policy", targetPath)
			}
		}
	}
	return nil
}

func resolvePath(workDir, p string) string {
	if p == "" || p == "." {
		return workDir
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(workDir, p)
}

func expandHome(p string) string {
	if p == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Printf("sandbox: cannot expand ~ (UserHomeDir: %v), skipping path", err)
			return ""
		}
		return home
	}
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Printf("sandbox: cannot expand ~ (UserHomeDir: %v), skipping path %q", err, p)
			return ""
		}
		return filepath.Join(home, p[2:])
	}
	return p
}
