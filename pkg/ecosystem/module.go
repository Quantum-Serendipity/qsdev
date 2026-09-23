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

	// CICommands returns CI pipeline commands for this ecosystem. Returning nil
	// means this ecosystem contributes no CI steps.
	CICommands(config ModuleConfig) []CICommand

	// PackageManagers returns metadata about the ecosystem's package managers.
	PackageManagers() []PackageManagerInfo

	// VerificationCommands returns the build/test/lint/typecheck/format commands
	// for this ecosystem. Used by agent-postmortem-skill to inject project-specific
	// verification steps. A zero-value result means no verification commands apply.
	VerificationCommands(config ModuleConfig) VerificationCommands
}

// PackageProvider is an optional interface that ecosystem modules can
// implement to contribute Nix packages to the devenv shell beyond those
// implied by hooks and language fragments. Names are bare (e.g. "gopls"),
// not prefixed with "pkgs.".
type PackageProvider interface {
	DevenvPackages(config ModuleConfig) []string
}

// PackageExprProvider is an optional interface for modules that must add a
// package that cannot be named by a bare nixpkgs attribute, such as a package
// built with extra components. Each entry is a complete Nix expression that
// evaluates to a derivation and may refer to pkgs; it is rendered verbatim as
// one element of the devenv.nix package list, so function applications must be
// parenthesized.
type PackageExprProvider interface {
	DevenvPackageExprs(config ModuleConfig) []string
}

// DevenvYamlInputProvider is an optional interface that ecosystem modules can
// implement to contribute additional flake inputs to devenv.yaml. Modules that
// do not need extra flake inputs simply omit this interface.
type DevenvYamlInputProvider interface {
	DevenvYamlInputs(config ModuleConfig) []DevenvInput
}

// WizardFieldProvider is an optional interface that ecosystem modules can
// implement to contribute additional wizard form fields. Modules that require
// no extra user input simply omit this interface.
type WizardFieldProvider interface {
	WizardFields() []WizardField
}

// ManifestFileProvider is an optional interface that ecosystem modules can
// implement to declare dependency manifest and lock files. Used by
// Version-Sentinel integration to determine which files can be guarded.
type ManifestFileProvider interface {
	ManifestFiles(config ModuleConfig) []ManifestFileInfo
}

// DependencyDeclarer is an optional interface that ecosystem modules can
// implement to report whether the project's manifest declares any external
// dependencies. Package managers write no lock file for a manifest without
// dependencies (a go.mod with no require directives never gets a go.sum), so
// lock file enforcement skips such projects instead of demanding a file that
// cannot be generated. Modules that omit this interface are assumed to
// declare dependencies. An error means the answer is unknown, and callers
// must then enforce the lock file as usual.
type DependencyDeclarer interface {
	DeclaresDependencies(projectRoot string) (bool, error)
}

// DenyRuleProvider is an optional interface that ecosystem modules can
// implement to contribute Claude Code deny-rule patterns. Modules that
// need no deny rules simply omit this interface.
type DenyRuleProvider interface {
	DenyRules(config ModuleConfig) []string
}

// ReadDenyRuleProvider is an optional interface that ecosystem modules can
// implement to contribute Claude Code sandbox ReadDeny patterns. These block
// the agent from reading credential files on the local filesystem. Modules
// that need no read deny rules simply omit this interface.
type ReadDenyRuleProvider interface {
	ReadDenyRules(config ModuleConfig) []string
}

// DoctorCheckProvider is an optional interface that ecosystem modules can
// implement to contribute doctor health checks. Each check is an independent
// validation that runs during "qsdev doctor".
type DoctorCheckProvider interface {
	DoctorChecks(config ModuleConfig) []DoctorCheck
}
