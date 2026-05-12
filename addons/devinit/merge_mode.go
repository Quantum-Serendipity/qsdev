package devinit

import (
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

// ExistingConfig summarises which configuration files already exist in the
// project directory. The wizard uses this to decide whether to offer a
// merge-mode prompt.
type ExistingConfig struct {
	HasDevenv bool     // devenv.nix or devenv.yaml
	HasClaude bool     // .claude/, CLAUDE.md, or settings
	HasEnvrc  bool     // .envrc
	HasMcp    bool     // .mcp.json
	Files     []string // human-readable list for display
}

// DetectExistingConfig builds an ExistingConfig from detection results.
func DetectExistingConfig(detected types.DetectedProject) ExistingConfig {
	ec := ExistingConfig{}

	if detected.HasDevenvNix {
		ec.HasDevenv = true
		ec.Files = append(ec.Files, "devenv.nix")
	}
	if detected.HasDevenvYaml {
		ec.HasDevenv = true
		ec.Files = append(ec.Files, "devenv.yaml")
	}

	if detected.HasClaudeDir {
		ec.HasClaude = true
		ec.Files = append(ec.Files, ".claude/")
	}
	if detected.HasClaudeMd {
		ec.HasClaude = true
		ec.Files = append(ec.Files, "CLAUDE.md")
	}
	if detected.HasClaudeSettings {
		ec.HasClaude = true
		ec.Files = append(ec.Files, ".claude/settings.json")
	}

	if detected.HasEnvrc {
		ec.HasEnvrc = true
		ec.Files = append(ec.Files, ".envrc")
	}

	if detected.HasMcpJson {
		ec.HasMcp = true
		ec.Files = append(ec.Files, ".mcp.json")
	}

	return ec
}

// NeedsMergeMode returns true when any existing configuration was detected,
// meaning the wizard should ask how to handle conflicts.
func (ec ExistingConfig) NeedsMergeMode() bool {
	return ec.HasDevenv || ec.HasClaude || ec.HasEnvrc || ec.HasMcp
}
