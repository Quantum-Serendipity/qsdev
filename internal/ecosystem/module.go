package ecosystem

import "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"

// EcosystemModule is the contract that every language/platform ecosystem
// must implement. It drives detection, code generation, security policy,
// and wizard UX for a single ecosystem (e.g. Go, Node, Python).
type EcosystemModule interface {
	// Name returns the canonical identifier (e.g. "go", "javascript", "python").
	Name() string

	// DisplayName returns the human-readable label (e.g. "Go", "JavaScript/TypeScript").
	DisplayName() string

	// Tier returns the implementation priority tier (1 = core, 2 = standard, 3 = extended).
	Tier() int

	// Detect scans projectRoot for ecosystem indicators and returns a DetectionResult.
	Detect(projectRoot string) DetectionResult

	// DevenvNixFragment returns a Nix code fragment to include in devenv.nix.
	DevenvNixFragment(config ModuleConfig) (string, error)

	// DevenvYamlInputs returns additional flake inputs for devenv.yaml.
	DevenvYamlInputs(config ModuleConfig) []DevenvInput

	// SecurityConfigs returns generated security configuration files
	// (e.g. .npmrc, pip.conf, cargo config).
	SecurityConfigs(config ModuleConfig) []types.GeneratedFile

	// PreCommitHooks returns pre-commit hook definitions for this ecosystem.
	PreCommitHooks(config ModuleConfig) []HookConfig

	// DenyRules returns Claude Code deny-rule patterns for this ecosystem
	// (e.g. "npm install --ignore-scripts").
	DenyRules(config ModuleConfig) []string

	// CICommands returns CI pipeline commands for this ecosystem.
	CICommands(config ModuleConfig) []CICommand

	// PackageManagers returns metadata about the ecosystem's package managers.
	PackageManagers() []PackageManagerInfo

	// WizardFields returns additional wizard form fields this ecosystem needs.
	WizardFields() []WizardField

	// VerificationCommands returns the build/test/lint/typecheck/format commands
	// for this ecosystem. Used by agent-postmortem-skill to inject project-specific
	// verification steps.
	VerificationCommands(config ModuleConfig) VerificationCommands

	// ManifestFiles returns metadata about dependency manifest and lock files
	// for this ecosystem. Used by Version-Sentinel integration to determine
	// which files can be guarded.
	ManifestFiles(config ModuleConfig) []ManifestFileInfo
}
