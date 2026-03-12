package mount

import (
	"os"
	"path/filepath"

	"github.com/hiragram/agent-workspace/internal/docker"
)

// MountOptions contains the parameters needed to construct Docker mounts.
type MountOptions struct {
	HomeDir             string // host user home directory
	WorkDir             string // host working directory (workspace)
	ClaudeHome          string // host ~/.claude
	ContainerClaudeHome string // host ~/.agent-workspace
	ContainerClaudeJSON string // host ~/.agent-workspace.json
	VolumeName          string // Docker volume name for Claude installation
	SSHKeyPath          string // SSH private key path (optional)
}

// Builder constructs Docker mount arguments.
type Builder interface {
	BuildMounts(opts MountOptions) ([]docker.Mount, error)
}

// DefaultBuilder is the default mount builder that checks the real filesystem.
type DefaultBuilder struct{}

// NewBuilder creates a new DefaultBuilder.
func NewBuilder() *DefaultBuilder {
	return &DefaultBuilder{}
}

// BuildMounts constructs the full list of Docker mounts for the container.
func (b *DefaultBuilder) BuildMounts(opts MountOptions) ([]docker.Mount, error) {
	var mounts []docker.Mount

	// Fixed mounts (always present)
	mounts = append(mounts, docker.Mount{
		Source:   opts.VolumeName,
		Target:   "/home/claude/.local",
		IsVolume: true,
	})
	mounts = append(mounts, docker.Mount{
		Source: opts.ContainerClaudeHome,
		Target: "/home/claude/.claude",
	})
	mounts = append(mounts, docker.Mount{
		Source: opts.ContainerClaudeJSON,
		Target: "/home/claude/.claude.json",
	})
	mounts = append(mounts, docker.Mount{
		Source: opts.WorkDir,
		Target: opts.WorkDir,
	})

	// Optional host mounts
	mounts = append(mounts, optionalMounts(opts.HomeDir, opts.SSHKeyPath)...)

	// Worktree mount
	worktreeMount, err := worktreeMount(opts.WorkDir)
	if err != nil {
		return nil, err
	}
	if worktreeMount != nil {
		mounts = append(mounts, *worktreeMount)
	}

	return mounts, nil
}

// optionalMounts returns mounts for host files that may or may not exist.
func optionalMounts(homeDir, sshKeyPath string) []docker.Mount {
	var mounts []docker.Mount

	// .gitconfig
	gitconfig := filepath.Join(homeDir, ".gitconfig")
	if fileExists(gitconfig) {
		mounts = append(mounts, docker.Mount{
			Source: gitconfig,
			Target: "/home/claude/.gitconfig",
		})
	}

	// .config/gh
	ghConfig := filepath.Join(homeDir, ".config", "gh")
	if dirExists(ghConfig) {
		mounts = append(mounts, docker.Mount{
			Source: ghConfig,
			Target: "/home/claude/.config/gh",
		})
	}

	// .config/glab-cli (mounted read-only, entrypoint copies it)
	glabConfig := filepath.Join(homeDir, ".config", "glab-cli")
	if dirExists(glabConfig) {
		mounts = append(mounts, docker.Mount{
			Source:   glabConfig,
			Target:   "/home/claude/.config-glab-cli-host",
			ReadOnly: true,
		})
	}

	// SSH key files (individual file mounts when ssh_key is specified)
	mounts = append(mounts, sshKeyMounts(homeDir, sshKeyPath)...)

	return mounts
}

// sshKeyMounts returns individual SSH file mounts when sshKeyPath is specified.
func sshKeyMounts(homeDir, sshKeyPath string) []docker.Mount {
	if sshKeyPath == "" {
		return nil
	}

	var mounts []docker.Mount

	// Private key (required)
	if fileExists(sshKeyPath) {
		mounts = append(mounts, docker.Mount{
			Source:   sshKeyPath,
			Target:   "/home/claude/.ssh-host/" + filepath.Base(sshKeyPath),
			ReadOnly: true,
		})
	}

	// Public key (optional)
	pubKey := sshKeyPath + ".pub"
	if fileExists(pubKey) {
		mounts = append(mounts, docker.Mount{
			Source:   pubKey,
			Target:   "/home/claude/.ssh-host/" + filepath.Base(pubKey),
			ReadOnly: true,
		})
	}

	// ~/.ssh/config (optional)
	sshConfig := filepath.Join(homeDir, ".ssh", "config")
	if fileExists(sshConfig) {
		mounts = append(mounts, docker.Mount{
			Source:   sshConfig,
			Target:   "/home/claude/.ssh-host/config",
			ReadOnly: true,
		})
	}

	// ~/.ssh/known_hosts (optional)
	knownHosts := filepath.Join(homeDir, ".ssh", "known_hosts")
	if fileExists(knownHosts) {
		mounts = append(mounts, docker.Mount{
			Source:   knownHosts,
			Target:   "/home/claude/.ssh-host/known_hosts",
			ReadOnly: true,
		})
	}

	return mounts
}

// worktreeMount returns an additional mount for the main .git directory
// if the workspace is a git worktree.
func worktreeMount(workDir string) (*docker.Mount, error) {
	mainGitDir, err := DetectWorktree(workDir)
	if err != nil {
		return nil, err
	}
	if mainGitDir == "" {
		return nil, nil
	}

	// If the main .git dir is already under the workspace, no extra mount needed
	if IsSubpath(workDir, mainGitDir) {
		return nil, nil
	}

	return &docker.Mount{
		Source: mainGitDir,
		Target: mainGitDir,
	}, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
