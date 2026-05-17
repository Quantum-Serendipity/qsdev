// Package ansible implements the Ansible ecosystem module for qsdev.
// It detects Ansible projects by scanning for ansible.cfg, galaxy.yml, playbooks/,
// roles/, and requirements.yml, then generates devenv.nix fragments with ansible
// and ansible-lint packages, a security-hardened ansible.cfg with GPG signature
// verification, pre-commit hooks, deny rules, and CI commands.
package ansible

import (
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Module is the stateless Ansible ecosystem module.
type Module struct{}

// Name returns the canonical module identifier.
func (m *Module) Name() string { return "ansible" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Ansible" }

// Tier returns the implementation priority tier (2 = standard).
func (m *Module) Tier() int { return 2 }

// Detect scans projectRoot for Ansible ecosystem indicators.
// ansible.cfg and galaxy.yml yield Certain confidence; playbooks/, roles/,
// and requirements.yml yield Probable confidence. No file content scanning
// is performed for performance.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	result := ecosystem.DetectionResult{}

	// Certain indicators.
	if fileutil.FileExists(projectRoot, "ansible.cfg") {
		result.Detected = true
		result.Confidence = ecosystem.ConfidenceCertain
		result.Evidence = append(result.Evidence, "ansible.cfg found")
	}
	if fileutil.FileExists(projectRoot, "galaxy.yml") {
		result.Detected = true
		result.Confidence = ecosystem.ConfidenceCertain
		result.Evidence = append(result.Evidence, "galaxy.yml found")
	}

	// Probable indicators.
	if fileutil.DirExists(projectRoot, "playbooks") {
		result.Evidence = append(result.Evidence, "playbooks/ directory found")
		if !result.Detected {
			result.Detected = true
			result.Confidence = ecosystem.ConfidenceProbable
		}
	}
	if fileutil.DirExists(projectRoot, "roles") {
		result.Evidence = append(result.Evidence, "roles/ directory found")
		if !result.Detected {
			result.Detected = true
			result.Confidence = ecosystem.ConfidenceProbable
		}
	}
	if fileutil.FileExists(projectRoot, "requirements.yml") {
		result.Evidence = append(result.Evidence, "requirements.yml found")
		if !result.Detected {
			result.Detected = true
			result.Confidence = ecosystem.ConfidenceProbable
		}
	}

	return result
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Ansible support. Uses a packages-based approach.
func (m *Module) DevenvNixFragment(_ ecosystem.ModuleConfig) (string, error) {
	var b strings.Builder
	b.WriteString("  packages = with pkgs; [ ansible ansible-lint ];\n")
	return b.String(), nil
}

// DevenvYamlInputs returns additional flake inputs for devenv.yaml.
// Ansible does not require any additional inputs.
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return nil
}

// SecurityConfigs returns a security-hardened Ansible configuration stored in a
// separate file to avoid overwriting the user's ansible.cfg.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	var b strings.Builder
	b.WriteString("# Security-hardened Ansible configuration (qsdev-managed)\n")
	b.WriteString("# Merge into your ansible.cfg or set ANSIBLE_CONFIG=.ansible-security.cfg\n")
	b.WriteString("# Requires: Ansible >= 2.15 for GPG signature verification of collections.\n")
	b.WriteString("\n")
	b.WriteString("[galaxy]\n")
	b.WriteString("gpg_keyring = ~/.ansible/keyring.gpg\n")
	b.WriteString("required_valid_signature_count = 1\n")

	return []types.GeneratedFile{
		{
			Path:     ".ansible-security.cfg",
			Content:  []byte(b.String()),
			Mode:     0o644,
			Strategy: types.Overwrite,
		},
	}
}

// PreCommitHooks returns pre-commit hook definitions for the Ansible ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "ansible-lint",
			Name:          "ansible-lint",
			Description:   "Lint Ansible playbooks and roles with ansible-lint",
			Entry:         "ansible-lint",
			Language:      "system",
			Types:         []string{"yaml"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       true,
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the Ansible ecosystem.
// These prevent direct galaxy install outside of controlled workflows.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"Bash(ansible-galaxy install *)",
		"Bash(ansible-galaxy collection install *)",
	}
}

// CICommands returns CI pipeline commands for the Ansible ecosystem.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name:        "galaxy-install",
			Command:     "ansible-galaxy install -r requirements.yml --keyring=~/.ansible/keyring.gpg",
			Description: "Install Ansible Galaxy dependencies with keyring verification",
			Phase:       ecosystem.CIPhaseInstall,
		},
		{
			Name:        "ansible-lint",
			Command:     "ansible-lint",
			Description: "Lint Ansible playbooks and roles",
			Phase:       ecosystem.CIPhaseScan,
		},
	}
}

// PackageManagers returns metadata about the Ansible Galaxy dependency system.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:     "ansible-galaxy",
			LockFile: "requirements.yml",
		},
	}
}

// WizardFields returns nil. Ansible does not require additional wizard fields.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return nil
}

// VerificationCommands returns an empty set. Ansible does not define standard
// verification commands at the module level.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{}
}

// ManifestFiles returns nil. Ansible does not use a traditional manifest file.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return nil
}

