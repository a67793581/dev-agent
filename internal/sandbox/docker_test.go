package sandbox

import (
	"os"
	"strings"
	"testing"
)

func TestDockerConfig_DockerEnabled_Default(t *testing.T) {
	cfg := DockerConfig{}
	if !cfg.DockerEnabled() {
		t.Error("expected DockerEnabled to default to true")
	}
}

func TestDockerConfig_DockerEnabled_Explicit(t *testing.T) {
	enabled := true
	cfg := DockerConfig{Enabled: &enabled}
	if !cfg.DockerEnabled() {
		t.Error("expected DockerEnabled true when explicitly set")
	}

	disabled := false
	cfg2 := DockerConfig{Enabled: &disabled}
	if cfg2.DockerEnabled() {
		t.Error("expected DockerEnabled false when explicitly disabled")
	}
}

func TestNewDockerExecutor_Defaults(t *testing.T) {
	d := NewDockerExecutor("/tmp/work", DockerConfig{})
	if d.Image != defaultImage {
		t.Errorf("image: got %q, want %q", d.Image, defaultImage)
	}
	if d.Network != defaultNetwork {
		t.Errorf("network: got %q, want %q", d.Network, defaultNetwork)
	}
	if d.Memory != defaultMemory {
		t.Errorf("memory: got %q, want %q", d.Memory, defaultMemory)
	}
	if d.CPUs != defaultCPUs {
		t.Errorf("cpus: got %q, want %q", d.CPUs, defaultCPUs)
	}
	if d.WorkDir != "/tmp/work" {
		t.Errorf("workDir: got %q", d.WorkDir)
	}
}

func TestNewDockerExecutor_CustomConfig(t *testing.T) {
	cfg := DockerConfig{
		Image:   "alpine:3.18",
		Network: "none",
		Memory:  "1g",
		CPUs:    "4",
	}
	d := NewDockerExecutor("/project", cfg)
	if d.Image != "alpine:3.18" {
		t.Errorf("image: got %q", d.Image)
	}
	if d.Network != "none" {
		t.Errorf("network: got %q", d.Network)
	}
	if d.Memory != "1g" {
		t.Errorf("memory: got %q", d.Memory)
	}
	if d.CPUs != "4" {
		t.Errorf("cpus: got %q", d.CPUs)
	}
}

func TestContainerName_Deterministic(t *testing.T) {
	d1 := NewDockerExecutor("/my/project", DockerConfig{})
	d2 := NewDockerExecutor("/my/project", DockerConfig{})
	if d1.ContainerName() != d2.ContainerName() {
		t.Errorf("same workDir should produce same container name: %q vs %q", d1.ContainerName(), d2.ContainerName())
	}
}

func TestContainerName_UniquePerProject(t *testing.T) {
	d1 := NewDockerExecutor("/project/a", DockerConfig{})
	d2 := NewDockerExecutor("/project/b", DockerConfig{})
	if d1.ContainerName() == d2.ContainerName() {
		t.Errorf("different workDirs should produce different container names: both %q", d1.ContainerName())
	}
}

func TestContainerName_HasPrefix(t *testing.T) {
	d := NewDockerExecutor("/any/path", DockerConfig{})
	if !strings.HasPrefix(d.ContainerName(), containerPrefix) {
		t.Errorf("container name %q should start with %q", d.ContainerName(), containerPrefix)
	}
}

func TestCreateArgs_Basic(t *testing.T) {
	d := NewDockerExecutor("/my/project", DockerConfig{})
	args := d.CreateArgs()

	joined := strings.Join(args, " ")
	expected := []string{
		"create",
		"--name", d.ContainerName(),
		"--network", defaultNetwork,
		"--memory", defaultMemory,
		"--cpus", defaultCPUs,
		"--security-opt", "no-new-privileges",
		"-v", "/my/project:/workspace",
		"-w", "/workspace",
		defaultImage,
		"sleep", "infinity",
	}
	for _, part := range expected {
		if !strings.Contains(joined, part) {
			t.Errorf("expected args to contain %q, got: %s", part, joined)
		}
	}
	if strings.Contains(joined, "--read-only") {
		t.Error("CreateArgs should NOT contain --read-only")
	}
	if strings.Contains(joined, "--rm") {
		t.Error("CreateArgs should NOT contain --rm")
	}
}

func TestCreateArgs_ExtraMounts(t *testing.T) {
	home, _ := os.UserHomeDir()
	cfg := DockerConfig{
		ExtraMounts: []MountConfig{
			{Source: "~/go/pkg/mod", Target: "/go/pkg/mod", ReadOnly: true},
			{Source: "/usr/local/bin", Target: "/usr/local/bin", ReadOnly: false},
		},
	}
	d := NewDockerExecutor("/project", cfg)
	args := d.CreateArgs()

	joined := strings.Join(args, " ")
	expectedRO := home + "/go/pkg/mod:/go/pkg/mod:ro"
	expectedRW := "/usr/local/bin:/usr/local/bin"
	if !strings.Contains(joined, expectedRO) {
		t.Errorf("expected readonly mount %q in args: %s", expectedRO, joined)
	}
	if !strings.Contains(joined, expectedRW) {
		t.Errorf("expected rw mount %q in args: %s", expectedRW, joined)
	}
}

func TestCreateArgs_NetworkNone(t *testing.T) {
	cfg := DockerConfig{Network: "none"}
	d := NewDockerExecutor("/project", cfg)
	args := d.CreateArgs()

	found := false
	for i, a := range args {
		if a == "--network" && i+1 < len(args) && args[i+1] == "none" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected --network none in args: %v", args)
	}
}

func TestLoadConfig_WithDocker(t *testing.T) {
	dir := t.TempDir()
	cfgDir := dir + "/.devagent"
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := []byte(`mode: normal
docker:
  enabled: true
  image: "node:20"
  network: "none"
  memory: "1g"
  cpus: "4"
  extra_mounts:
    - source: "~/npm-cache"
      target: "/root/.npm"
      readonly: true
`)
	if err := os.WriteFile(cfgDir+"/sandbox.yaml", content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if !cfg.Docker.DockerEnabled() {
		t.Error("expected docker enabled")
	}
	if cfg.Docker.Image != "node:20" {
		t.Errorf("docker.image: got %q", cfg.Docker.Image)
	}
	if cfg.Docker.Network != "none" {
		t.Errorf("docker.network: got %q", cfg.Docker.Network)
	}
	if cfg.Docker.Memory != "1g" {
		t.Errorf("docker.memory: got %q", cfg.Docker.Memory)
	}
	if cfg.Docker.CPUs != "4" {
		t.Errorf("docker.cpus: got %q", cfg.Docker.CPUs)
	}
	if len(cfg.Docker.ExtraMounts) != 1 {
		t.Fatalf("docker.extra_mounts: got %d", len(cfg.Docker.ExtraMounts))
	}
	m := cfg.Docker.ExtraMounts[0]
	if m.Source != "~/npm-cache" || m.Target != "/root/.npm" || !m.ReadOnly {
		t.Errorf("extra_mount[0]: %+v", m)
	}
}

func TestLoadConfig_DockerDisabled(t *testing.T) {
	dir := t.TempDir()
	cfgDir := dir + "/.devagent"
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := []byte(`docker:
  enabled: false
`)
	if err := os.WriteFile(cfgDir+"/sandbox.yaml", content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Docker.DockerEnabled() {
		t.Error("expected docker disabled")
	}
}
