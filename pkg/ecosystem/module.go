package ecosystem

import "github.com/Quantum-Serendipity/qsdev/pkg/types"

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

	// SecurityConfigs returns generated security configuration files
	// (e.g. .npmrc, pip.conf, cargo config). Returning nil indicates no
	// security configs are needed for this ecosystem.
	SecurityConfigs(config ModuleConfig) []types.GeneratedFile

	// PreCommitHooks returns pre-commit hook definitions for this ecosystem.
	// Returning nil means no pre-commit hooks are contributed by this module.
	PreCommitHooks(config ModuleConfig) []HookConfig

	// DenyRules returns Claude Code deny-rule patterns for this ecosystem
	// (e.g. "npm install --ignore-scripts"). Returning nil means no deny
	// rules are needed for this ecosystem.
	DenyRules(config ModuleConfig) []string

	// CICommands returns CI pipeline commands for this ecosystem. Returning nil
	// means this ecosystem contributes no CI steps.
	CICommands(config ModuleConfig) []CICommand

	// PackageManagers returns metadata about the ecosystem's package managers.
	PackageManagers() []PackageManagerInfo

	// WizardFields returns additional wizard form fields this ecosystem needs.
	// Returning nil means the ecosystem requires no extra user input beyond
	// language selection.
	WizardFields() []WizardField

	// VerificationCommands returns the build/test/lint/typecheck/format commands
	// for this ecosystem. Used by agent-postmortem-skill to inject project-specific
	// verification steps. A zero-value result means no verification commands apply.
	VerificationCommands(config ModuleConfig) VerificationCommands

	// ManifestFiles returns metadata about dependency manifest and lock files
	// for this ecosystem. Used by Version-Sentinel integration to determine
	// which files can be guarded. Returning nil means there are no manifest
	// files for this ecosystem.
	ManifestFiles(config ModuleConfig) []ManifestFileInfo
}

// PackageProvider is an optional interface that ecosystem modules can
// implement to contribute Nix packages to the devenv shell beyond those
// implied by hooks and language fragments. Names are bare (e.g. "gopls"),
// not prefixed with "pkgs.".
type PackageProvider interface {
	DevenvPackages(config ModuleConfig) []string
}

// DevenvYamlInputProvider is an optional interface that ecosystem modules can
// implement to contribute additional flake inputs to devenv.yaml. Modules that
// do not need extra flake inputs simply omit this interface.
type DevenvYamlInputProvider interface {
	DevenvYamlInputs(config ModuleConfig) []DevenvInput
}
