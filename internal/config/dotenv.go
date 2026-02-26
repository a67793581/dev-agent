package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadEnv loads environment variables from .env files.
// Priority (highest to lowest):
//  1. Already set environment variables (never overwritten)
//  2. Explicitly specified env file via -env flag
//  3. .env file in the current working directory
//  4. .env file in the user's home directory (~/.devagent.env)
func LoadEnv(envFile string) error {
	var files []string

	if envFile != "" {
		if _, err := os.Stat(envFile); err != nil {
			return fmt.Errorf("env file not found: %s", envFile)
		}
		files = append(files, envFile)
	}

	if cwd, err := os.Getwd(); err == nil {
		local := filepath.Join(cwd, ".env")
		if _, err := os.Stat(local); err == nil {
			files = append(files, local)
		}
	}

	if home, err := os.UserHomeDir(); err == nil {
		global := filepath.Join(home, ".devagent.env")
		if _, err := os.Stat(global); err == nil {
			files = append(files, global)
		}
	}

	if len(files) == 0 {
		return nil
	}

	// godotenv.Load does NOT overwrite existing env vars
	if err := godotenv.Load(files...); err != nil {
		return fmt.Errorf("load env: %w", err)
	}

	return nil
}
