package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill represents a discovered skill with metadata and lazy-loaded body.
type Skill struct {
	Name        string
	Description string
	Dir         string   // directory containing SKILL.md
	Body        string   // markdown body (loaded on demand)
	path        string   // full path to SKILL.md
}

type frontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// Discover walks the given directories (in order), finds SKILL.md files
// (at any depth), parses their frontmatter, and returns unique skills by name
// (first occurrence wins).
func Discover(dirs []string) ([]Skill, error) {
	seen := make(map[string]bool)
	var skills []Skill

	for _, root := range dirs {
		root = filepath.Clean(root)
		if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if d.IsDir() || d.Name() != "SKILL.md" {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read %s: %w", path, err)
			}
			name, desc, err := parseFrontmatter(data)
			if err != nil {
				return fmt.Errorf("parse %s: %w", path, err)
			}
			if name == "" {
				return nil
			}
			if seen[name] {
				return nil
			}
			seen[name] = true
			skills = append(skills, Skill{
				Name:        name,
				Description: desc,
				Dir:         filepath.Dir(path),
				path:        path,
			})
			return nil
		}); err != nil {
			return nil, err
		}
	}

	return skills, nil
}

// LoadBody reads the full SKILL.md and sets skill.Body to the markdown content
// after the frontmatter (the second --- delimiter and beyond).
func LoadBody(skill *Skill) error {
	data, err := os.ReadFile(skill.path)
	if err != nil {
		return fmt.Errorf("read %s: %w", skill.path, err)
	}
	_, body := splitFrontmatter(data)
	skill.Body = strings.TrimSpace(string(body))
	return nil
}

func parseFrontmatter(data []byte) (name, description string, err error) {
	fm, _ := splitFrontmatter(data)
	if len(fm) == 0 {
		return "", "", nil
	}
	var f frontmatter
	if err := yaml.Unmarshal(fm, &f); err != nil {
		return "", "", err
	}
	return strings.TrimSpace(f.Name), strings.TrimSpace(f.Description), nil
}

// splitFrontmatter returns (yamlBytes, bodyBytes). Frontmatter is between first --- and second ---.
func splitFrontmatter(data []byte) ([]byte, []byte) {
	sep := []byte("---")
	first := indexBytes(data, sep, 0)
	if first < 0 {
		return nil, data
	}
	second := indexBytes(data, sep, first+len(sep))
	if second < 0 {
		return nil, data
	}
	return data[first+len(sep) : second], data[second+len(sep):]
}

func indexBytes(b, sep []byte, from int) int {
	if from >= len(b) || len(sep) == 0 {
		return -1
	}
	for i := from; i <= len(b)-len(sep); i++ {
		if b[i] == sep[0] && (len(sep) == 1 || equalBytes(b[i:i+len(sep)], sep)) {
			return i
		}
	}
	return -1
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
