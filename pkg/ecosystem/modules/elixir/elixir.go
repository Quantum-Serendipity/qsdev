// Package elixir implements the Elixir (Mix) ecosystem module for
// qsdev. It detects Elixir projects by scanning for
// mix.exs and mix.lock, generates devenv.nix fragments with Elixir language
// support, and provides pre-commit hooks, CI commands, deny rules, and package
// manager metadata for the Elixir toolchain.
package elixir

import (
	"path/filepath"
	"regexp"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.DenyRuleProvider = (*Module)(nil)
var _ ecosystem.ManifestFileProvider = (*Module)(nil)
var _ ecosystem.SetupWarner = (*Module)(nil)

// extraMixAudit is the Extras key Detect sets, to "true", when mix.exs
// declares the mix_audit dependency that provides `mix deps.audit`. It is
// absent otherwise, so a later re-init merges it in once the project adds it.
const extraMixAudit = "mix_audit"

// mixAuditDep matches the :mix_audit atom as a whole token.
var mixAuditDep = regexp.MustCompile(`(?:^|[\s,{\[(]):mix_audit(?:[\s,}\])]|$)`)

// declaresMixAudit reports whether the project's mix.exs declares
// mix_audit, ignoring comments and strings.
func declaresMixAudit(projectRoot string) bool {
	return ecosystem.FileDeclares(filepath.Join(projectRoot, "mix.exs"), '#', mixAuditDep)
}

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Module implements ecosystem.EcosystemModule for the Elixir programming language.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "elixir" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Elixir" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 3 }

// Detect scans projectRoot for mix.exs and mix.lock files, and records
// whether mix.exs declares mix_audit (Extras mix_audit).
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	hasMixExs := fileutil.FileExists(projectRoot, "mix.exs")
	hasMixLock := fileutil.FileExists(projectRoot, "mix.lock")

	if !hasMixExs && !hasMixLock {
		return ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		}
	}

	confidence := ecosystem.ConfidenceProbable
	var evidence []string
	var suggested ecosystem.ModuleConfig

	if hasMixExs {
		confidence = ecosystem.ConfidenceCertain
		evidence = append(evidence, "mix.exs found")
		if declaresMixAudit(projectRoot) {
			suggested.Extras = map[string]string{extraMixAudit: "true"}
		}
	}
	if hasMixLock {
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		evidence = append(evidence, "mix.lock found")
	}

	return ecosystem.DetectionResult{
		Detected:        true,
		Confidence:      confidence,
		Evidence:        evidence,
		SuggestedConfig: suggested,
	}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Elixir language support. Mix needs Hex to fetch packages and rebar3 to
// build Erlang dependencies, and otherwise offers to download them into
// ~/.mix (which prompts, or fails on a clean CI runner): point it at the
// Nix-built ones instead.
func (m *Module) DevenvNixFragment(_ ecosystem.ModuleConfig) (string, error) {
	return `  languages.elixir.enable = true;
  env.MIX_PATH = "${pkgs.beamPackages.hex}/lib/erlang/lib/hex/ebin";
  env.MIX_REBAR3 = "${pkgs.rebar3}/bin/rebar3";
`, nil
}

// SecurityConfigs returns generated security configuration files.
// Elixir relies on mix.lock for integrity; no additional config files are needed.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// PreCommitHooks returns pre-commit hook definitions for the Elixir ecosystem.
// mix-format runs the Elixir languages.elixir provides; devenv's elixir module
// sets the same hook's package to that toolchain.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:              "mix-format",
			Name:            "mix-format",
			Description:     "Check Elixir code formatting with mix format",
			Entry:           "mix format --check-formatted",
			Language:        "system",
			Types:           []string{"elixir"},
			Stages:          []string{"pre-commit"},
			PassFilenames:   false,
			BuiltIn:         false,
			LanguagePackage: "elixir",
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the Elixir ecosystem.
// These prevent direct dependency fetching outside of controlled workflows.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	// mix deps.get only fetches what mix.lock pins (it is the frozen restore,
	// with --check-locked), so it stays open. These change the supply chain:
	// deps.update/deps.unlock re-resolve and rewrite mix.lock, and
	// archive.install/escript.install put code in ~/.mix that every later mix
	// invocation (in any project) loads or runs.
	return []string{
		"Bash(mix deps.update*)",
		"Bash(mix deps.unlock*)",
		"Bash(mix archive.install*)",
		"Bash(mix escript.install*)",
		// igniter.install adds a dependency to mix.exs and fetches it.
		"Bash(mix igniter.install*)",
	}
}

// CICommands returns CI pipeline commands for the Elixir ecosystem: the
// locked install, and `mix deps.audit` only when the project declares
// mix_audit (see Detect), since the task comes from that dependency.
func (m *Module) CICommands(config ecosystem.ModuleConfig) []ecosystem.CICommand {
	cmds := []ecosystem.CICommand{
		{
			Name:        "mix-deps-get-locked",
			Command:     "mix deps.get --check-locked",
			Description: "Install Elixir dependencies with lockfile verification",
			Phase:       ecosystem.CIPhaseInstall,
		},
	}
	if config.Extra(extraMixAudit, "") == "true" {
		cmds = append(cmds, ecosystem.CICommand{
			Name:        "mix-deps-audit",
			Command:     "mix deps.audit",
			Description: "Audit Elixir dependencies for known vulnerabilities",
			Phase:       ecosystem.CIPhaseScan,
		})
	}
	return cmds
}

// SetupWarnings reports a project that does not declare mix_audit, so
// CICommands emits no audit step. mix.exs is read as well as config, since an
// answers file saved before the project added mix_audit lacks the extra that
// generation fills in from detection.
func (m *Module) SetupWarnings(projectRoot string, config ecosystem.ModuleConfig) []string {
	if config.Extra(extraMixAudit, "") == "true" || declaresMixAudit(projectRoot) {
		return nil
	}
	return []string{"Elixir dependency security scan not run: add {:mix_audit, \"~> <version>\", only: [:dev, :test], runtime: false} " +
		"to the deps in mix.exs, then run `qsdev init --update`"}
}

// PackageManagers returns metadata about the Elixir Mix package manager.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:     "mix",
			LockFile: "mix.lock",
		},
	}
}

// VerificationCommands returns build, test, and format commands for Elixir projects.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{
		Build:  []string{"mix compile"},
		Test:   []string{"mix test"},
		Format: []string{"mix format --check-formatted"},
	}
}

// ManifestFiles returns the mix.exs manifest file for Elixir projects.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return []ecosystem.ManifestFileInfo{{Path: "mix.exs", Ecosystem: "hex", LockFile: "mix.lock", LockFilePolicy: ecosystem.LockFilePolicyRequired}}
}
