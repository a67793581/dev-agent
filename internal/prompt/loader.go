package prompt

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolvePromptFile returns the trimmed content of the first existing prompt file.
// Lookup order: flagPath (if non-empty), projectDir/.devagent/filename, ~/.devagent/filename.
// Returns empty string if none found.
func ResolvePromptFile(flagPath, projectDir, filename string) string {
	var candidates []string
	if flagPath != "" {
		candidates = append(candidates, flagPath)
	}
	if projectDir != "" {
		candidates = append(candidates, filepath.Join(projectDir, ".devagent", filename))
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".devagent", filename))
	}
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		return strings.TrimSpace(string(data))
	}
	return ""
}
