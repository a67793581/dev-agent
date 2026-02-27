package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	defaultImage   = "ubuntu:22.04"
	defaultNetwork = "host"
	defaultMemory  = "512m"
	defaultCPUs    = "2"
	containerPrefix = "devagent-"
)

// DockerExecutor runs shell commands inside a persistent Docker container.
// One container is created per project (identified by workDir hash) and reused
// across commands. The container stays alive between commands via "sleep infinity".
type DockerExecutor struct {
	Image         string
	WorkDir       string // host path, mounted at /workspace
	Network       string
	Memory        string
	CPUs          string
	ExtraMounts   []MountConfig
	Timeout       time.Duration
	containerName string
	mu            sync.Mutex
}

// NewDockerExecutor creates a DockerExecutor from DockerConfig and workDir.
// Fields not set in cfg fall back to defaults.
func NewDockerExecutor(workDir string, cfg DockerConfig) *DockerExecutor {
	d := &DockerExecutor{
		Image:       defaultImage,
		WorkDir:     workDir,
		Network:     defaultNetwork,
		Memory:      defaultMemory,
		CPUs:        defaultCPUs,
		ExtraMounts: cfg.ExtraMounts,
		Timeout:     5 * time.Minute,
	}
	if cfg.Image != "" {
		d.Image = cfg.Image
	}
	if cfg.Network != "" {
		d.Network = cfg.Network
	}
	if cfg.Memory != "" {
		d.Memory = cfg.Memory
	}
	if cfg.CPUs != "" {
		d.CPUs = cfg.CPUs
	}
	d.containerName = containerPrefix + hashWorkDir(workDir)
	return d
}

func hashWorkDir(workDir string) string {
	h := sha256.Sum256([]byte(workDir))
	return fmt.Sprintf("%x", h[:6])
}

// ContainerName returns the deterministic container name for this project.
func (d *DockerExecutor) ContainerName() string {
	return d.containerName
}

// DockerAvailable checks whether docker is installed and the daemon is reachable.
func DockerAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// containerStatus returns "running", "exited", or "" (not found).
func (d *DockerExecutor) containerStatus() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Status}}", d.containerName)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(out.String())
}

// CreateArgs returns the docker create argument list (without "docker" itself).
func (d *DockerExecutor) CreateArgs() []string {
	args := []string{
		"create",
		"--name", d.containerName,
		"--network", d.Network,
		"--memory", d.Memory,
		"--cpus", d.CPUs,
		"--security-opt", "no-new-privileges",
		"-v", d.WorkDir + ":/workspace",
		"-w", "/workspace",
	}
	for _, m := range d.ExtraMounts {
		src := expandHome(m.Source)
		if src == "" {
			continue
		}
		mount := src + ":" + m.Target
		if m.ReadOnly {
			mount += ":ro"
		}
		args = append(args, "-v", mount)
	}
	args = append(args, d.Image, "sleep", "infinity")
	return args
}

// EnsureRunning makes sure the project container exists and is running.
// If it doesn't exist, it creates and starts it. If it exists but is stopped, it starts it.
func (d *DockerExecutor) EnsureRunning() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	status := d.containerStatus()
	switch status {
	case "running":
		return nil
	case "exited", "created":
		return d.startContainer()
	case "":
		if err := d.createContainer(); err != nil {
			return err
		}
		return d.startContainer()
	default:
		// Unknown state — remove and recreate
		_ = d.removeContainer()
		if err := d.createContainer(); err != nil {
			return err
		}
		return d.startContainer()
	}
}

func (d *DockerExecutor) createContainer() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	args := d.CreateArgs()
	cmd := exec.CommandContext(ctx, "docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker create failed: %v: %s", err, stderr.String())
	}
	return nil
}

func (d *DockerExecutor) startContainer() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "start", d.containerName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker start failed: %v: %s", err, stderr.String())
	}
	return nil
}

func (d *DockerExecutor) removeContainer() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "rm", "-f", d.containerName)
	return cmd.Run()
}

// Stop gracefully stops the container but does not remove it, so it can be reused next time.
func (d *DockerExecutor) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "stop", d.containerName)
	_ = cmd.Run()
}

// Cleanup stops and removes the container entirely.
func (d *DockerExecutor) Cleanup() {
	d.mu.Lock()
	defer d.mu.Unlock()
	_ = d.removeContainer()
}

// Execute runs a command inside the persistent container via docker exec.
func (d *DockerExecutor) Execute(command string) (output string, exitCode int, err error) {
	if err := d.EnsureRunning(); err != nil {
		return "", -1, err
	}

	timeout := d.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := []string{"exec", "-w", "/workspace", d.containerName, "bash", "-c", command}
	cmd := exec.CommandContext(ctx, "docker", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	var sb strings.Builder
	if stdout.Len() > 0 {
		sb.WriteString(stdout.String())
	}
	if stderr.Len() > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("[stderr]\n")
		sb.WriteString(stderr.String())
	}
	output = sb.String()

	if runErr != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return output, -1, fmt.Errorf("docker command timed out after %v", timeout)
		}
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			return output, exitErr.ExitCode(), nil
		}
		return output, -1, runErr
	}
	return output, 0, nil
}
