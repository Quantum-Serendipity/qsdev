// Package ansible implements the Ansible ecosystem module for qsdev.
// It detects Ansible projects by scanning for ansible.cfg, galaxy.yml, playbooks/,
// roles/, and requirements.yml, then generates devenv.nix fragments with ansible
// and ansible-lint packages, opt-in GPG signature enforcement for Galaxy
// collections, pre-commit hooks, deny rules, and CI commands.
package ansible

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.PackageProvider = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Module is the stateless Ansible ecosystem module.
type Module struct{}

// Name returns the canonical module identifier.
func (m *Module) Name() string { return ecosystem.NameAnsible }

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
	// Galaxy requirements files: the root one and the conventional
	// collections/ and roles/ locations. They are recorded so the CI install
	// uses the files that exist.
	var reqFiles []string
	for _, rel := range requirementsFileCandidates {
		if !fileutil.FileExists(projectRoot, rel) {
			continue
		}
		reqFiles = append(reqFiles, rel)
		result.Evidence = append(result.Evidence, rel+" found")
		if !result.Detected {
			result.Detected = true
			result.Confidence = ecosystem.ConfidenceProbable
		}
	}
	if result.Detected && len(reqFiles) > 0 {
		result.SuggestedConfig.Extras = map[string]string{ExtraRequirementsFiles: strings.Join(reqFiles, ",")}
	}

	return result
}

// requirementsFileCandidates are the Galaxy requirements files Detect looks
// for, relative to the project root.
var requirementsFileCandidates = []string{"requirements.yml", "collections/requirements.yml", "roles/requirements.yml"}

// ExtraRequirementsFiles is the ModuleConfig.Extras key holding the
// comma-separated Galaxy requirements files found by Detect. When it is unset
// the root requirements.yml is assumed.
const ExtraRequirementsFiles = "requirements_files"

// requirementsFiles returns the recorded requirements files, keeping only the
// known candidate paths (the value is embedded in a CI shell command).
func requirementsFiles(config ecosystem.ModuleConfig) []string {
	var files []string
	for _, f := range strings.Split(config.Extra(ExtraRequirementsFiles, ""), ",") {
		if slices.Contains(requirementsFileCandidates, f) {
			files = append(files, f)
		}
	}
	if len(files) == 0 {
		return []string{"requirements.yml"}
	}
	return files
}

// DevenvPackages returns the Nix packages required for the Ansible ecosystem.
func (m *Module) DevenvPackages(_ ecosystem.ModuleConfig) []string {
	return []string{"ansible", "ansible-lint"}
}

// ExtraGalaxyKeyring is the ModuleConfig.Extras key naming the GPG keyring
// that holds the signing keys of the project's Galaxy distribution server
// (Automation Hub or a private Galaxy NG). Setting it turns on signature
// enforcement for collections.
const ExtraGalaxyKeyring = "galaxy_gpg_keyring"

// keyringPathPattern limits the keyring path to characters that are safe in
// a Nix string and an unquoted CI shell argument.
var keyringPathPattern = regexp.MustCompile(`^[A-Za-z0-9._/~+-]+$`)

// galaxyKeyring returns the configured keyring path, "" when signature
// enforcement is not configured, or an error for a path it cannot use safely.
func galaxyKeyring(config ecosystem.ModuleConfig) (string, error) {
	keyring := strings.TrimSpace(config.Extra(ExtraGalaxyKeyring, ""))
	if keyring == "" {
		return "", nil
	}
	if !keyringPathPattern.MatchString(keyring) || strings.HasPrefix(keyring, "-") {
		return "", fmt.Errorf("invalid %s %q: want a plain file path", ExtraGalaxyKeyring, keyring)
	}
	return keyring, nil
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Ansible support. Packages are provided via DevenvPackages.
//
// When a Galaxy keyring is configured (ExtraGalaxyKeyring) the fragment
// enforces collection signatures through ANSIBLE_GALAXY_* environment
// variables. Ansible reads exactly one config file and never merges, so env
// vars are the only way to layer this over the user's own ansible.cfg. The
// signature count is "+1": without the "+", Ansible accepts a collection that
// has no signatures at all. It is opt-in because public galaxy.ansible.com
// serves no signatures, so "+1" would make every public install fail.
// Roles are never signature-verified.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	keyring, err := galaxyKeyring(config)
	if err != nil || keyring == "" {
		return "", err
	}
	return fmt.Sprintf("  env.ANSIBLE_GALAXY_GPG_KEYRING = %s;\n  env.ANSIBLE_GALAXY_REQUIRED_VALID_SIGNATURE_COUNT = \"+1\";\n",
		ecosystem.NixString(keyring)), nil
}

// SecurityConfigs returns nil. Galaxy signature enforcement is configured
// through environment variables in DevenvNixFragment; a separate config file
// would never be loaded (Ansible reads only one) and pointing ANSIBLE_CONFIG
// at it would drop the user's own ansible.cfg.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
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
// They block Galaxy installs and downloads of roles and collections (in every
// form, including `collection install`, `role install` and verbose flags
// before the subcommand) outside controlled workflows, and ansible-vault
// commands that print decrypted secrets.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return append(
		denyutil.SubcommandRules("ansible-galaxy", "install", "download"),
		// edit prints the plaintext too when EDITOR is a pager or cat.
		denyutil.SubcommandRules("ansible-vault", "view", "decrypt", "edit")...,
	)
}

// ReadDenyRules returns the vault password files the agent's Read tool must
// not open, by the names Ansible projects conventionally give them (the
// vault_password_file setting has no default).
func (m *Module) ReadDenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"**/.vault_pass*",
		"**/.vault-pass*",
		"**/vault_pass*",
		"**/vault-pass*",
	}
}

// CICommands returns CI pipeline commands for the Ansible ecosystem. The
// Galaxy install verifies collection signatures against the configured
// keyring when one is set (ExtraGalaxyKeyring).
func (m *Module) CICommands(config ecosystem.ModuleConfig) []ecosystem.CICommand {
	install := ecosystem.CICommand{
		Name:        "galaxy-install",
		Description: "Install Ansible Galaxy dependencies",
		Phase:       ecosystem.CIPhaseInstall,
	}
	envPrefix := ""
	if keyring, err := galaxyKeyring(config); err == nil && keyring != "" {
		// The same variables DevenvNixFragment sets, so CI enforces
		// signatures whether or not it runs inside the devenv shell.
		envPrefix = "ANSIBLE_GALAXY_GPG_KEYRING=" + keyring + " ANSIBLE_GALAXY_REQUIRED_VALID_SIGNATURE_COUNT=+1 "
		install.Description = "Install Ansible Galaxy dependencies with collection signature verification"
	}
	var cmds []string
	for _, f := range requirementsFiles(config) {
		cmds = append(cmds, envPrefix+"ansible-galaxy install -r "+f)
	}
	install.Command = strings.Join(cmds, " && ")
	return []ecosystem.CICommand{
		install,
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

// VerificationCommands returns an empty set. Ansible does not define standard
// verification commands at the module level.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{}
}

// Compile-time check that the Ansible module declares its manifest.
var _ ecosystem.ManifestFileProvider = (*Module)(nil)
var _ ecosystem.ReadDenyRuleProvider = (*Module)(nil)

// ManifestFiles declares requirements.yml so Version-Sentinel coverage
// reports list Galaxy dependencies as uncovered instead of omitting them.
// Galaxy has no separate lock file: versions are pinned in the manifest.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return []ecosystem.ManifestFileInfo{{
		Path:           "requirements.yml",
		Ecosystem:      "ansible-galaxy",
		LockFilePolicy: ecosystem.LockFilePolicyNone,
	}}
}
