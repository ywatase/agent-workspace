package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Validate checks that a profile configuration is semantically valid.
func Validate(p Profile) error {
	// Validate environment
	switch p.Environment {
	case EnvironmentHost, EnvironmentDocker:
		// ok
	case "":
		return fmt.Errorf("environment is required (\"host\" or \"docker\")")
	default:
		return fmt.Errorf("unknown environment: %q (must be \"host\" or \"docker\")", p.Environment)
	}

	// Validate launch mode
	switch p.Launch {
	case LaunchShell, LaunchClaude, LaunchZellij:
		// ok
	case "":
		return fmt.Errorf("launch is required (\"shell\", \"claude\", or \"zellij\")")
	default:
		return fmt.Errorf("unknown launch mode: %q (must be \"shell\", \"claude\", or \"zellij\")", p.Launch)
	}

	// Validate zellij config is only used with launch: zellij
	if p.Zellij != nil && p.Launch != LaunchZellij {
		return fmt.Errorf("zellij config is only valid with launch: zellij")
	}

	// Validate dockerfile is only used with environment: docker
	if p.Dockerfile != "" && p.Environment != EnvironmentDocker {
		return fmt.Errorf("dockerfile is only valid with environment: docker")
	}

	// Validate ssh_key is only used with environment: docker
	if p.SSHKey != "" && p.Environment != EnvironmentDocker {
		return fmt.Errorf("ssh_key is only valid with environment: docker")
	}

	// Warn if ssh_key path does not exist
	if p.SSHKey != "" {
		sshKeyPath := p.SSHKey
		if strings.HasPrefix(sshKeyPath, "~/") {
			if homeDir, err := os.UserHomeDir(); err == nil {
				sshKeyPath = filepath.Join(homeDir, sshKeyPath[2:])
			}
		}
		if _, err := os.Stat(sshKeyPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "warning: ssh_key path does not exist: %s\n", p.SSHKey)
		}
	}

	return nil
}

// ValidateConfig checks the entire config for errors.
func ValidateConfig(cfg *Config) error {
	if len(cfg.Profiles) == 0 {
		return fmt.Errorf("no profiles defined")
	}

	// Check that default profile exists if specified
	if cfg.Default != "" {
		if _, ok := cfg.Profiles[cfg.Default]; !ok {
			return fmt.Errorf("default profile %q not found in profiles", cfg.Default)
		}
	}

	// Validate each profile
	var errs []string
	for name, p := range cfg.Profiles {
		if err := Validate(p); err != nil {
			errs = append(errs, fmt.Sprintf("profile %q: %v", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation errors:\n  %s", strings.Join(errs, "\n  "))
	}

	return nil
}
