package sandbox

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	configDir  = ".devagent"
	configFile = "sandbox.yaml"
)

// Config is the root structure for .devagent/sandbox.yaml.
type Config struct {
	Mode   string       `yaml:"mode"`
	Shell  ShellConfig  `yaml:"shell"`
	Paths  PathsConfig  `yaml:"paths"`
	Docker DockerConfig `yaml:"docker"`
}

// ShellConfig holds shell block/approve/allow pattern lists (strings are glob-like patterns).
type ShellConfig struct {
	Block   []string `yaml:"block"`
	Approve []string `yaml:"approve"`
	Allow   []string `yaml:"allow"`
}

// PathsConfig holds path deny list and allow_outside_workdir list.
type PathsConfig struct {
	Deny                []string `yaml:"deny"`
	AllowOutsideWorkdir []string `yaml:"allow_outside_workdir"`
}

// DockerConfig controls the Docker container sandbox for shell commands.
type DockerConfig struct {
	Enabled     *bool         `yaml:"enabled"`
	Image       string        `yaml:"image"`
	Network     string        `yaml:"network"`
	Memory      string        `yaml:"memory"`
	CPUs        string        `yaml:"cpus"`
	ExtraMounts []MountConfig `yaml:"extra_mounts"`
}

// MountConfig describes an additional volume mount for the Docker container.
type MountConfig struct {
	Source   string `yaml:"source"`
	Target   string `yaml:"target"`
	ReadOnly bool   `yaml:"readonly"`
}

// DockerEnabled returns whether Docker sandbox is enabled (defaults to true).
func (c *DockerConfig) DockerEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// LoadConfig looks for <projectDir>/.devagent/sandbox.yaml and loads it.
// If the file does not exist, returns nil, nil (caller should use defaults).
func LoadConfig(projectDir string) (*Config, error) {
	path := filepath.Join(projectDir, configDir, configFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
