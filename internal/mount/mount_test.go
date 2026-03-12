package mount

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hiragram/agent-workspace/internal/docker"
)

func newTestOpts(homeDir, workDir string) MountOptions {
	return MountOptions{
		HomeDir:             homeDir,
		WorkDir:             workDir,
		ClaudeHome:          filepath.Join(homeDir, ".claude"),
		ContainerClaudeHome: filepath.Join(homeDir, ".agent-workspace"),
		ContainerClaudeJSON: filepath.Join(homeDir, ".agent-workspace.json"),
		VolumeName:          "claude-code-local",
	}
}

func findMount(mounts []docker.Mount, target string) *docker.Mount {
	for _, m := range mounts {
		if m.Target == target {
			return &m
		}
	}
	return nil
}

func TestBuildMounts_FixedMountsAlwaysPresent(t *testing.T) {
	homeDir := t.TempDir()
	workDir := t.TempDir()
	opts := newTestOpts(homeDir, workDir)

	builder := NewBuilder()
	mounts, err := builder.BuildMounts(opts)
	if err != nil {
		t.Fatalf("BuildMounts() error: %v", err)
	}

	// Volume mount
	vol := findMount(mounts, "/home/claude/.local")
	if vol == nil {
		t.Fatal("missing volume mount for /home/claude/.local")
	}
	if vol.Source != "claude-code-local" || !vol.IsVolume {
		t.Errorf("volume mount = %+v, want source=claude-code-local, IsVolume=true", vol)
	}

	// Claude config mount
	cfg := findMount(mounts, "/home/claude/.claude")
	if cfg == nil {
		t.Fatal("missing mount for /home/claude/.claude")
	}
	if cfg.Source != opts.ContainerClaudeHome {
		t.Errorf("claude config source = %q, want %q", cfg.Source, opts.ContainerClaudeHome)
	}

	// Claude JSON mount
	json := findMount(mounts, "/home/claude/.claude.json")
	if json == nil {
		t.Fatal("missing mount for /home/claude/.claude.json")
	}

	// Workspace mount
	ws := findMount(mounts, workDir)
	if ws == nil {
		t.Fatalf("missing workspace mount for %s", workDir)
	}
	if ws.Source != workDir {
		t.Errorf("workspace source = %q, want %q", ws.Source, workDir)
	}
}

func TestBuildMounts_GitconfigWhenPresent(t *testing.T) {
	homeDir := t.TempDir()
	workDir := t.TempDir()

	// Create .gitconfig
	if err := os.WriteFile(filepath.Join(homeDir, ".gitconfig"), []byte("[user]\nname=test"), 0644); err != nil {
		t.Fatalf("writing .gitconfig: %v", err)
	}

	opts := newTestOpts(homeDir, workDir)
	builder := NewBuilder()
	mounts, err := builder.BuildMounts(opts)
	if err != nil {
		t.Fatalf("BuildMounts() error: %v", err)
	}

	m := findMount(mounts, "/home/claude/.gitconfig")
	if m == nil {
		t.Fatal("missing .gitconfig mount")
	}
	if m.Source != filepath.Join(homeDir, ".gitconfig") {
		t.Errorf("source = %q, want %q", m.Source, filepath.Join(homeDir, ".gitconfig"))
	}
}

func TestBuildMounts_NoGitconfigWhenMissing(t *testing.T) {
	homeDir := t.TempDir()
	workDir := t.TempDir()

	opts := newTestOpts(homeDir, workDir)
	builder := NewBuilder()
	mounts, err := builder.BuildMounts(opts)
	if err != nil {
		t.Fatalf("BuildMounts() error: %v", err)
	}

	if findMount(mounts, "/home/claude/.gitconfig") != nil {
		t.Error(".gitconfig mount should not exist when file is missing")
	}
}

func TestBuildMounts_GhConfigWhenPresent(t *testing.T) {
	homeDir := t.TempDir()
	workDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(homeDir, ".config", "gh"), 0755); err != nil {
		t.Fatalf("creating .config/gh: %v", err)
	}

	opts := newTestOpts(homeDir, workDir)
	builder := NewBuilder()
	mounts, err := builder.BuildMounts(opts)
	if err != nil {
		t.Fatalf("BuildMounts() error: %v", err)
	}

	m := findMount(mounts, "/home/claude/.config/gh")
	if m == nil {
		t.Fatal("missing .config/gh mount")
	}
}

func TestBuildMounts_GlabConfigWhenPresent(t *testing.T) {
	homeDir := t.TempDir()
	workDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(homeDir, ".config", "glab-cli"), 0755); err != nil {
		t.Fatalf("creating .config/glab-cli: %v", err)
	}

	opts := newTestOpts(homeDir, workDir)
	builder := NewBuilder()
	mounts, err := builder.BuildMounts(opts)
	if err != nil {
		t.Fatalf("BuildMounts() error: %v", err)
	}

	m := findMount(mounts, "/home/claude/.config-glab-cli-host")
	if m == nil {
		t.Fatal("missing .config-glab-cli-host mount")
	}
	if !m.ReadOnly {
		t.Error(".config/glab-cli mount should be read-only")
	}
	if m.Source != filepath.Join(homeDir, ".config", "glab-cli") {
		t.Errorf("source = %q, want %q", m.Source, filepath.Join(homeDir, ".config", "glab-cli"))
	}
}

func TestBuildMounts_NoGlabConfigWhenMissing(t *testing.T) {
	homeDir := t.TempDir()
	workDir := t.TempDir()

	opts := newTestOpts(homeDir, workDir)
	builder := NewBuilder()
	mounts, err := builder.BuildMounts(opts)
	if err != nil {
		t.Fatalf("BuildMounts() error: %v", err)
	}

	if findMount(mounts, "/home/claude/.config-glab-cli-host") != nil {
		t.Error(".config-glab-cli-host mount should not exist when directory is missing")
	}
}

func TestBuildMounts_SSHKeyMountsIndividualFiles(t *testing.T) {
	homeDir := t.TempDir()
	workDir := t.TempDir()

	// Create SSH key files
	sshDir := filepath.Join(homeDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("creating .ssh: %v", err)
	}
	keyPath := filepath.Join(sshDir, "id_ed25519")
	if err := os.WriteFile(keyPath, []byte("private-key"), 0600); err != nil {
		t.Fatalf("writing key: %v", err)
	}
	if err := os.WriteFile(keyPath+".pub", []byte("public-key"), 0644); err != nil {
		t.Fatalf("writing pub key: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "config"), []byte("Host *"), 0644); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "known_hosts"), []byte("github.com ..."), 0644); err != nil {
		t.Fatalf("writing known_hosts: %v", err)
	}

	opts := newTestOpts(homeDir, workDir)
	opts.SSHKeyPath = keyPath
	builder := NewBuilder()
	mounts, err := builder.BuildMounts(opts)
	if err != nil {
		t.Fatalf("BuildMounts() error: %v", err)
	}

	// Private key
	m := findMount(mounts, "/home/claude/.ssh-host/id_ed25519")
	if m == nil {
		t.Fatal("missing private key mount")
	}
	if !m.ReadOnly {
		t.Error("private key mount should be read-only")
	}
	if m.Source != keyPath {
		t.Errorf("source = %q, want %q", m.Source, keyPath)
	}

	// Public key
	m = findMount(mounts, "/home/claude/.ssh-host/id_ed25519.pub")
	if m == nil {
		t.Fatal("missing public key mount")
	}
	if !m.ReadOnly {
		t.Error("public key mount should be read-only")
	}

	// config
	m = findMount(mounts, "/home/claude/.ssh-host/config")
	if m == nil {
		t.Fatal("missing config mount")
	}

	// known_hosts
	m = findMount(mounts, "/home/claude/.ssh-host/known_hosts")
	if m == nil {
		t.Fatal("missing known_hosts mount")
	}
}

func TestBuildMounts_NoSSHWhenKeyNotSpecified(t *testing.T) {
	homeDir := t.TempDir()
	workDir := t.TempDir()

	// Even if .ssh directory exists, no mounts when SSHKeyPath is empty
	sshDir := filepath.Join(homeDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("creating .ssh: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "id_ed25519"), []byte("key"), 0600); err != nil {
		t.Fatalf("writing key: %v", err)
	}

	opts := newTestOpts(homeDir, workDir)
	// SSHKeyPath is empty (default)
	builder := NewBuilder()
	mounts, err := builder.BuildMounts(opts)
	if err != nil {
		t.Fatalf("BuildMounts() error: %v", err)
	}

	for _, m := range mounts {
		if filepath.Dir(m.Target) == "/home/claude/.ssh-host" {
			t.Errorf("unexpected SSH mount: %+v", m)
		}
	}
}

func TestBuildMounts_SSHKeyPubMissingOk(t *testing.T) {
	homeDir := t.TempDir()
	workDir := t.TempDir()

	// Create only private key, no .pub
	sshDir := filepath.Join(homeDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("creating .ssh: %v", err)
	}
	keyPath := filepath.Join(sshDir, "id_rsa")
	if err := os.WriteFile(keyPath, []byte("private-key"), 0600); err != nil {
		t.Fatalf("writing key: %v", err)
	}

	opts := newTestOpts(homeDir, workDir)
	opts.SSHKeyPath = keyPath
	builder := NewBuilder()
	mounts, err := builder.BuildMounts(opts)
	if err != nil {
		t.Fatalf("BuildMounts() error: %v", err)
	}

	// Private key should be mounted
	m := findMount(mounts, "/home/claude/.ssh-host/id_rsa")
	if m == nil {
		t.Fatal("missing private key mount")
	}

	// Public key should NOT be mounted (doesn't exist)
	if findMount(mounts, "/home/claude/.ssh-host/id_rsa.pub") != nil {
		t.Error("public key mount should not exist when .pub file is missing")
	}
}

func TestBuildMounts_WorktreeAddsMount(t *testing.T) {
	// Set up a worktree scenario
	baseDir := t.TempDir()

	mainRepo := filepath.Join(baseDir, "main-repo")
	mainGitDir := filepath.Join(mainRepo, ".git")
	if err := os.MkdirAll(filepath.Join(mainGitDir, "worktrees", "wt"), 0755); err != nil {
		t.Fatalf("creating worktree dir: %v", err)
	}

	worktreeDir := filepath.Join(baseDir, "worktree")
	if err := os.MkdirAll(worktreeDir, 0755); err != nil {
		t.Fatalf("creating worktree dir: %v", err)
	}

	gitdirPath := filepath.Join(mainGitDir, "worktrees", "wt")
	if err := os.WriteFile(filepath.Join(worktreeDir, ".git"), []byte("gitdir: "+gitdirPath+"\n"), 0644); err != nil {
		t.Fatalf("writing .git file: %v", err)
	}

	homeDir := t.TempDir()
	opts := newTestOpts(homeDir, worktreeDir)
	builder := NewBuilder()
	mounts, err := builder.BuildMounts(opts)
	if err != nil {
		t.Fatalf("BuildMounts() error: %v", err)
	}

	absMainGitDir, _ := filepath.Abs(mainGitDir)
	m := findMount(mounts, absMainGitDir)
	if m == nil {
		t.Fatalf("missing worktree mount for %s", absMainGitDir)
	}
	if m.Source != absMainGitDir {
		t.Errorf("source = %q, want %q", m.Source, absMainGitDir)
	}
}

func TestBuildMounts_NoWorktreeMount_RegularRepo(t *testing.T) {
	homeDir := t.TempDir()
	workDir := t.TempDir()

	// Regular .git directory
	if err := os.MkdirAll(filepath.Join(workDir, ".git"), 0755); err != nil {
		t.Fatalf("creating .git dir: %v", err)
	}

	opts := newTestOpts(homeDir, workDir)
	builder := NewBuilder()
	mounts, err := builder.BuildMounts(opts)
	if err != nil {
		t.Fatalf("BuildMounts() error: %v", err)
	}

	// Should only have the 4 fixed mounts (no optional ones since homeDir is empty)
	if len(mounts) != 4 {
		t.Errorf("expected 4 mounts (fixed only), got %d: %+v", len(mounts), mounts)
	}
}
