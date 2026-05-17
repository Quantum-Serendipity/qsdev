// Package lua implements the Lua (LuaRocks/Lux) ecosystem module for
// qsdev. It detects Lua projects by scanning for
// rockspec files, lux.toml, and .luarocks/ directories, generates devenv.nix
// fragments with Lua language support, and provides pre-commit hooks, CI
// commands, deny rules, and package manager metadata for the Lua toolchain.
//
// Security limitations: LuaRocks has NO package signing. This is a documented
// gap since the 2019 LuaRocks server compromise incident. There is no
// built-in vulnerability audit for Lua packages. Lux is recommended as a
// more modern alternative with lockfile support (lux.lock). LuaRocks has
// no lockfile mechanism at all.
package lua

import (
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Module implements ecosystem.EcosystemModule for the Lua programming language.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "lua" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Lua" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 4 }

// Detect scans projectRoot for Lua ecosystem indicators: *.rockspec (certain),
// lux.toml (certain, sets PM to lux), and .luarocks/ directory (probable).
// Bare .lua files are intentionally NOT detected because they are too common
// as embedded scripts in other projects.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	var (
		evidence   []string
		confidence = ecosystem.ConfidenceAbsent
		detected   bool
		pm         string
	)

	// Certain indicators.
	if rockspecs, _ := filepath.Glob(filepath.Join(projectRoot, "*.rockspec")); len(rockspecs) > 0 {
		evidence = append(evidence, "*.rockspec files found")
		confidence = ecosystem.ConfidenceCertain
		detected = true
	}
	if fileutil.FileExists(projectRoot, "lux.toml") {
		evidence = append(evidence, "lux.toml found")
		confidence = ecosystem.ConfidenceCertain
		detected = true
		pm = "lux"
	}

	// Probable indicators.
	if fileutil.DirExists(projectRoot, ".luarocks") {
		evidence = append(evidence, ".luarocks/ directory found")
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		detected = true
	}

	if !detected {
		return ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		}
	}

	return ecosystem.DetectionResult{
		Detected:   true,
		Confidence: confidence,
		Evidence:   evidence,
		SuggestedConfig: ecosystem.ModuleConfig{
			PackageManager: pm,
		},
	}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Lua language support.
func (m *Module) DevenvNixFragment(_ ecosystem.ModuleConfig) (string, error) {
	return "  languages.lua.enable = true;\n", nil
}

// DevenvYamlInputs returns additional flake inputs for devenv.yaml.
// Lua does not require any additional inputs.
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return nil
}

// SecurityConfigs returns generated security configuration files.
// LuaRocks has no signing mechanism, so no security configuration files are
// generated. See package documentation for details on the 2019 compromise.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// PreCommitHooks returns pre-commit hook definitions for the Lua ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "stylua",
			Name:          "stylua",
			Description:   "Check Lua source formatting with StyLua",
			Entry:         "stylua --check",
			Language:      "system",
			Types:         []string{"lua"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       false,
		},
		{
			ID:            "luacheck",
			Name:          "luacheck",
			Description:   "Lint Lua source code with luacheck",
			Entry:         "luacheck",
			Language:      "system",
			Types:         []string{"lua"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       false,
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the Lua ecosystem.
// These prevent direct LuaRocks package installation outside of controlled workflows.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"Bash(luarocks install *)",
	}
}

// CICommands returns CI pipeline commands for the Lua ecosystem.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name:        "luarocks-install-deps",
			Command:     "luarocks install --local --only-deps",
			Description: "Install Lua dependencies locally from rockspec",
			Phase:       ecosystem.CIPhaseInstall,
		},
	}
}

// PackageManagers returns metadata about Lua's package managers.
// Both LuaRocks and Lux are listed. LuaRocks has no lockfile mechanism;
// Lux provides lux.lock for reproducible builds.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:             "luarocks",
			LockFile:         "",
			AgeGatingSupport: false,
		},
		{
			Name:             "lux",
			LockFile:         "lux.lock",
			AgeGatingSupport: false,
		},
	}
}

// WizardFields returns additional wizard form fields for Lua configuration.
// Lua does not require any wizard fields.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return nil
}

// VerificationCommands returns an empty set. Lua does not define standard
// verification commands at the module level.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{}
}

// ManifestFiles returns nil. Lua does not use a traditional manifest file.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return nil
}

