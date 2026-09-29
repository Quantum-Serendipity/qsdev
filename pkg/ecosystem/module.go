package ecosystem

import (
	"context"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

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
	// means this ecosystem contributes no CI steps. The generated
	// security-scan workflow runs them in the project's devenv shell, grouped
	// by phase (AggregateCICommands): install commands must enforce the lock
	// file rather than update it, and every command must succeed on a clean
	// checkout without credentials.
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
// evaluates to a derivation and may refer to the devenv.nix module arguments
// (pkgs, lib, config, e.g. config.languages.java.jdk.package to build against
// the project JDK); it is rendered verbatim as
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
// implement to contribute doctor health checks. "qsdev devenv doctor" runs
// the checks of every module configured in .qsdev.yaml statically: an
// EnvCheck is judged from the environment and the project's devenv modules,
// and a Command is never executed, only resolved on PATH and shown to the
// user as the manual verification step.
type DoctorCheckProvider interface {
	DoctorChecks(config ModuleConfig) []DoctorCheck
}

// ToolchainRequirementProvider is an optional interface for modules whose
// generated security settings only take effect with a minimum version of a
// tool: an older tool silently ignores the setting. "qsdev check" probes the
// tool on PATH for each requirement, so an inert setting is reported rather
// than assumed to be enforced.
type ToolchainRequirementProvider interface {
	ToolchainRequirements(config ModuleConfig) []ToolchainRequirement
}

// SetupWarner is an optional interface for modules that can tell when a
// project lacks files its configuration depends on, such as a package
// manager chosen for a project that has no manifest for it yet. The warnings
// are shown when the configuration is generated and never block it; modules
// with nothing to check simply omit this interface.
type SetupWarner interface {
	SetupWarnings(projectRoot string, config ModuleConfig) []string
}

// ToolchainChecker is an optional interface for modules that can tell when
// the toolchain on PATH does not match what the project's own files require,
// such as a GHC other than the one a Stack snapshot pins. It runs during
// "qsdev devenv doctor" and the devenv_doctor MCP tool, against the
// configuration the module's own Detect suggested. A check that cannot run
// (the tool is not on PATH, the file cannot be read) reports nothing; modules
// with nothing to check simply omit this interface.
type ToolchainChecker interface {
	ToolchainWarnings(ctx context.Context, projectRoot string, config ModuleConfig) []string
}
