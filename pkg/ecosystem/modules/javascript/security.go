package javascript

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ExtraYarnClassic is the ModuleConfig.Extras key that marks a Yarn project
// as Yarn Classic (v1). Classic reads .yarnrc (and .npmrc) but never
// .yarnrc.yml, so it needs a different hardening file from Yarn Berry (v2+).
const ExtraYarnClassic = "yarn_classic"

// bunMinimumReleaseAgeSeconds is the Bun install age gate. bunfig.toml's
// install.minimumReleaseAge is an integer number of SECONDS; a string such
// as "7d" makes `bun install` fail with "Invalid Bunfig".
const bunMinimumReleaseAgeSeconds = 7 * 24 * 60 * 60

// SecurityConfigs returns generated security configuration files for the
// detected (or user-selected) package manager. Only one PM-specific config
// is generated per invocation.
//
// Every file returned here is a conventional, often user-maintained package
// manager config (a pnpm monorepo's workspace list, scoped registries and
// auth in .npmrc, yarnPath/plugins in .yarnrc.yml). They therefore use the
// Skip strategy: an existing file is never replaced on first generation.
func (m *Module) SecurityConfigs(config ecosystem.ModuleConfig) []types.GeneratedFile {
	pm := config.PM("npm")

	switch pm {
	case "npm":
		return []types.GeneratedFile{npmSecurityConfig(config.RegistryProxy)}
	case "pnpm":
		return []types.GeneratedFile{pnpmSecurityConfig(config.RegistryProxy)}
	case "yarn":
		if config.Extra(ExtraYarnClassic, "") == "true" {
			return []types.GeneratedFile{yarnClassicSecurityConfig(config.RegistryProxy)}
		}
		return []types.GeneratedFile{yarnSecurityConfig(config.RegistryProxy)}
	case "bun":
		return []types.GeneratedFile{bunSecurityConfig()}
	default:
		return []types.GeneratedFile{npmSecurityConfig(config.RegistryProxy)}
	}
}

// npmSecurityConfig generates a hardened .npmrc in INI format.
func npmSecurityConfig(registryProxy string) types.GeneratedFile {
	var b strings.Builder
	b.WriteString("# Security-hardened npm configuration\n")
	b.WriteString("# " + branding.GeneratedBy() + " - do not remove security settings\n")
	b.WriteString("# Requires: npm >= 11.10.0 for min-release-age support (Feb 2026)\n")
	b.WriteString("\n")
	if registryProxy != "" {
		fmt.Fprintf(&b, "registry=%s\n", ecosystem.INIEscapeValue(registryProxy))
	}
	b.WriteString("# Pin exact versions to prevent unexpected updates\n")
	b.WriteString("save-exact=true\n")
	b.WriteString("# Disable lifecycle scripts to block malicious postinstall hooks\n")
	b.WriteString("ignore-scripts=true\n")
	b.WriteString("# Require packages to be published for at least 3 days\n")
	b.WriteString("min-release-age=3\n")
	b.WriteString("# Enable automatic security auditing on install\n")
	b.WriteString("audit=true\n")
	b.WriteString("# Fail on moderate and above vulnerabilities\n")
	b.WriteString("audit-level=moderate\n")

	return types.GeneratedFile{
		Path:     ".npmrc",
		Content:  []byte(b.String()),
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.Skip,
	}
}

// pnpmSecurityConfig generates a hardened pnpm-workspace.yaml using yaml.Node
// for comment support. Note: pnpm uses MINUTES for minimumReleaseAge (4320 = 3 days).
func pnpmSecurityConfig(registryProxy string) types.GeneratedFile {
	mappingContent := []*yaml.Node{}

	if registryProxy != "" {
		mappingContent = append(mappingContent,
			&yaml.Node{
				Kind:        yaml.ScalarNode,
				Value:       "npmRegistryServer",
				LineComment: "Corporate registry proxy",
			},
			&yaml.Node{Kind: yaml.ScalarNode, Value: registryProxy},
		)
	}

	mappingContent = append(mappingContent,
		&yaml.Node{
			Kind:        yaml.ScalarNode,
			Value:       "strictDepBuilds",
			LineComment: "Prevent dependency build scripts from running without approval",
		},
		&yaml.Node{Kind: yaml.ScalarNode, Value: "true", Tag: "!!bool"},
		&yaml.Node{
			Kind:        yaml.ScalarNode,
			Value:       "minimumReleaseAge",
			LineComment: "Require packages to be published for at least 3 days (4320 minutes)",
		},
		&yaml.Node{Kind: yaml.ScalarNode, Value: "4320", Tag: "!!int"},
		&yaml.Node{
			Kind:        yaml.ScalarNode,
			Value:       "trustPolicy",
			LineComment: "Prevent trust-level downgrades in dependencies",
		},
		&yaml.Node{Kind: yaml.ScalarNode, Value: "no-downgrade"},
		&yaml.Node{
			Kind:        yaml.ScalarNode,
			Value:       "blockExoticSubdeps",
			LineComment: "Block exotic protocols (git:, http:) in transitive dependencies",
		},
		&yaml.Node{Kind: yaml.ScalarNode, Value: "true", Tag: "!!bool"},
	)

	doc := &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{
				Kind:        yaml.MappingNode,
				HeadComment: "Security-hardened pnpm workspace configuration\n" + branding.GeneratedBy() + " - do not remove security settings\nRequires: pnpm >= 10.16 (Sep 2025); these settings are defaults in pnpm 11",
				Content:     mappingContent,
			},
		},
	}

	content, err := yaml.Marshal(doc)
	if err != nil {
		// Fallback to string-built content if marshaling fails.
		content = []byte("# Security-hardened pnpm workspace configuration\nstrictDepBuilds: true\nminimumReleaseAge: 4320\ntrustPolicy: no-downgrade\nblockExoticSubdeps: true\n")
	}

	return types.GeneratedFile{
		Path:     "pnpm-workspace.yaml",
		Content:  content,
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.Skip,
	}
}

// yarnSecurityConfig generates a hardened .yarnrc.yml using yaml.Node
// for comment support.
func yarnSecurityConfig(registryProxy string) types.GeneratedFile {
	mappingContent := []*yaml.Node{}

	if registryProxy != "" {
		mappingContent = append(mappingContent,
			&yaml.Node{
				Kind:        yaml.ScalarNode,
				Value:       "npmRegistryServer",
				LineComment: "Corporate registry proxy",
			},
			&yaml.Node{Kind: yaml.ScalarNode, Value: registryProxy},
		)
	}

	mappingContent = append(mappingContent,
		&yaml.Node{
			Kind:        yaml.ScalarNode,
			Value:       "enableImmutableInstalls",
			LineComment: "Prevent lockfile modifications during install",
		},
		&yaml.Node{Kind: yaml.ScalarNode, Value: "true", Tag: "!!bool"},
		&yaml.Node{
			Kind:        yaml.ScalarNode,
			Value:       "enableHardenedMode",
			LineComment: "Enable Yarn's hardened security mode",
		},
		&yaml.Node{Kind: yaml.ScalarNode, Value: "true", Tag: "!!bool"},
		&yaml.Node{
			Kind:        yaml.ScalarNode,
			Value:       "enableScripts",
			LineComment: "Disable lifecycle scripts to block malicious hooks",
		},
		&yaml.Node{Kind: yaml.ScalarNode, Value: "false", Tag: "!!bool"},
		&yaml.Node{
			Kind:        yaml.ScalarNode,
			Value:       "npmMinimalAgeGate",
			LineComment: "Require packages to be published for at least 7 days",
		},
		&yaml.Node{Kind: yaml.ScalarNode, Value: "7d"},
	)

	doc := &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{
				Kind:        yaml.MappingNode,
				HeadComment: "Security-hardened Yarn configuration\n" + branding.GeneratedBy() + " - do not remove security settings\nRequires: Yarn >= 4.10.0 (Sep 2025)",
				Content:     mappingContent,
			},
		},
	}

	content, err := yaml.Marshal(doc)
	if err != nil {
		content = []byte("# Security-hardened Yarn configuration\nenableImmutableInstalls: true\nenableHardenedMode: true\nenableScripts: false\nnpmMinimalAgeGate: 7d\n")
	}

	return types.GeneratedFile{
		Path:     ".yarnrc.yml",
		Content:  content,
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.Skip,
	}
}

// yarnClassicSecurityConfig generates a hardened .yarnrc for Yarn Classic
// (v1), which ignores .yarnrc.yml. Classic has no release-age gate, so only
// lifecycle scripts and the registry can be controlled here.
func yarnClassicSecurityConfig(registryProxy string) types.GeneratedFile {
	var b strings.Builder
	b.WriteString("# Security-hardened Yarn Classic (v1) configuration\n")
	b.WriteString("# " + branding.GeneratedBy() + " - do not remove security settings\n")
	b.WriteString("# Yarn Classic has no package age gate; migrate to Yarn >= 4.10\n")
	b.WriteString("# (Berry, .yarnrc.yml) to enforce one.\n")
	b.WriteString("\n")
	if registryProxy != "" {
		fmt.Fprintf(&b, "registry %s\n", strconv.Quote(registryProxy))
	}
	b.WriteString("# Disable lifecycle scripts to block malicious postinstall hooks\n")
	b.WriteString("ignore-scripts true\n")

	return types.GeneratedFile{
		Path:     ".yarnrc",
		Content:  []byte(b.String()),
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.Skip,
	}
}

// bunSecurityConfig generates a hardened bunfig.toml in string-built TOML format.
func bunSecurityConfig() types.GeneratedFile {
	var b strings.Builder
	b.WriteString("# Security-hardened Bun configuration\n")
	b.WriteString("# " + branding.GeneratedBy() + " - do not remove security settings\n")
	b.WriteString("# Requires: Bun >= 1.3 (Oct 2025)\n")
	b.WriteString("\n")
	b.WriteString("[install]\n")
	b.WriteString("# Require packages to be published for at least 7 days\n")
	fmt.Fprintf(&b, "minimumReleaseAge = %d\n", bunMinimumReleaseAgeSeconds)

	return types.GeneratedFile{
		Path:     "bunfig.toml",
		Content:  []byte(b.String()),
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.Skip,
	}
}
