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

// ExtraESLint and ExtraPrettier record that the project uses ESLint or
// Prettier, and whether the hook should run the project's node_modules copy
// ("node_modules") or the nixpkgs build ("nix"). Unset means the project does
// not use the tool and its pre-commit hook is not enabled.
const (
	ExtraESLint   = "eslint"
	ExtraPrettier = "prettier"
)

// ExtraPnpmVersion is the pnpm version (or range) package.json pins through
// "packageManager" or devEngines.packageManager.
const ExtraPnpmVersion = "pnpm_version"

// pnpmHardeningMinVersion is the first pnpm release that understands every
// setting pnpmSecurityConfig writes.
const pnpmHardeningMinVersion = "10.16"

// pnpmSupportsHardening reports whether the pinned pnpm version (or the lower
// bound of a pinned range) understands the generated hardening settings. An
// unparseable pin is given the benefit of the doubt.
func pnpmSupportsHardening(pin string) bool {
	major, minor, _, ok := parseVersion(pin)
	if !ok {
		return true
	}
	return major > 10 || (major == 10 && minor >= 16)
}

// npmAuditLevel is the minimum severity that makes `npm audit` exit non-zero.
// The generated .npmrc sets it as audit-level and the generated CI passes it
// to the `npm audit` step (CICommands), which is what actually gates on it.
const npmAuditLevel = "moderate"

// bunMinimumReleaseAgeSeconds is the Bun install age gate. bunfig.toml's
// install.minimumReleaseAge is an integer number of SECONDS; a string such
// as "7d" makes `bun install` fail with "Invalid Bunfig".
const bunMinimumReleaseAgeSeconds = 7 * 24 * 60 * 60

// SecurityConfigs returns generated security configuration files for the
// detected (or user-selected) package manager. Only one PM-specific config
// is generated per invocation.
//
// The files are written to the JavaScript project directory (a subproject
// such as frontend/ when that is where package.json lives), where the package
// manager reads them.
//
// Every file returned here is a conventional, often user-maintained package
// manager config (a pnpm monorepo's workspace list, scoped registries and
// auth in .npmrc, yarnPath/plugins in .yarnrc.yml). They therefore use the
// Skip strategy: an existing file is never replaced on first generation.
func (m *Module) SecurityConfigs(config ecosystem.ModuleConfig) []types.GeneratedFile {
	gf := securityConfig(config)
	gf.Path = config.InDirectory(gf.Path)
	return []types.GeneratedFile{gf}
}

// securityConfig returns the hardening file for the configured package
// manager, with a path relative to the JavaScript project directory.
func securityConfig(config ecosystem.ModuleConfig) types.GeneratedFile {
	pm := config.PM("npm")

	switch pm {
	case "npm":
		return npmSecurityConfig(config.RegistryProxy)
	case "pnpm":
		return pnpmSecurityConfig(config.RegistryProxy, config.Extra(ExtraPnpmVersion, ""))
	case "yarn":
		if config.Extra(ExtraYarnClassic, "") == "true" {
			return yarnClassicSecurityConfig(config.RegistryProxy)
		}
		return yarnSecurityConfig(config.RegistryProxy)
	case "bun":
		return bunSecurityConfig(config.RegistryProxy)
	default:
		return npmSecurityConfig(config.RegistryProxy)
	}
}

// npmSecurityConfig generates a hardened .npmrc in INI format.
func npmSecurityConfig(registryProxy string) types.GeneratedFile {
	var b strings.Builder
	b.WriteString("# Security-hardened npm configuration\n")
	b.WriteString("# " + branding.GeneratedBy() + " - do not remove security settings\n")
	fmt.Fprintf(&b, "# Requires: npm >= %s for min-release-age (older npm silently ignores it).\n", npmMinReleaseAgeVersion)
	fmt.Fprintf(&b, "# The devenv shell provides it; `%s check` fails when npm on PATH is older.\n", branding.Get().AppName)
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
	fmt.Fprintf(&b, "# Make `npm audit` exit non-zero on %s and above vulnerabilities.\n", npmAuditLevel)
	b.WriteString("# Installs are not blocked by it: `npm ci`/`npm install` never fail on audit\n")
	b.WriteString("# results. Run `npm audit` in CI to gate on it (the qsdev ecosystem-ci job does).\n")
	fmt.Fprintf(&b, "audit-level=%s\n", npmAuditLevel)

	return types.GeneratedFile{
		Path:     ".npmrc",
		Content:  []byte(b.String()),
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.Skip,
	}
}

// pnpmSecurityConfig generates a hardened pnpm-workspace.yaml using yaml.Node
// for comment support. Note: pnpm uses MINUTES for minimumReleaseAge (4320 = 3 days).
// pinnedVersion is the pnpm version package.json pins, if any.
func pnpmSecurityConfig(registryProxy, pinnedVersion string) types.GeneratedFile {
	mappingContent := []*yaml.Node{}

	if registryProxy != "" {
		mappingContent = append(mappingContent,
			&yaml.Node{
				Kind:        yaml.ScalarNode,
				Value:       "registry",
				LineComment: "Corporate registry proxy",
			},
			&yaml.Node{Kind: yaml.ScalarNode, Value: registryProxy},
		)
	}

	if pinnedVersion != "" && !pnpmSupportsHardening(pinnedVersion) {
		// By default pnpm downloads and switches to the pinned release, which
		// ignores every setting below; keep the Nix-provided pnpm instead.
		mappingContent = append(mappingContent,
			&yaml.Node{
				Kind:        yaml.ScalarNode,
				Value:       "pmOnFail",
				LineComment: fmt.Sprintf("package.json pins pnpm %s, which ignores these settings; warn and keep the devenv pnpm instead of switching (bump the pin to >= %s)", pinnedVersion, pnpmHardeningMinVersion),
			},
			&yaml.Node{Kind: yaml.ScalarNode, Value: "warn"},
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

	// enableImmutableInstalls is deliberately not set: Yarn already turns it
	// on in CI, and committing it makes every local `yarn install` after a
	// manifest change fail with YN0028.
	mappingContent = append(mappingContent,
		&yaml.Node{
			Kind:        yaml.ScalarNode,
			Value:       "enableHardenedMode",
			LineComment: "Validate lockfile resolutions against the registry (slower installs; trade-off accepted for security)",
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
				HeadComment: "Security-hardened Yarn configuration\n" + branding.GeneratedBy() + " - do not remove security settings\nRequires: Yarn >= 4.12 (npmMinimalAgeGate)",
				Content:     mappingContent,
			},
		},
	}

	content, err := yaml.Marshal(doc)
	if err != nil {
		content = []byte("# Security-hardened Yarn configuration\nenableHardenedMode: true\nenableScripts: false\nnpmMinimalAgeGate: 7d\n")
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
	b.WriteString("# Yarn Classic has no package age gate; migrate to Yarn >= 4.12\n")
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
func bunSecurityConfig(registryProxy string) types.GeneratedFile {
	var b strings.Builder
	b.WriteString("# Security-hardened Bun configuration\n")
	b.WriteString("# " + branding.GeneratedBy() + " - do not remove security settings\n")
	b.WriteString("# Requires: Bun >= 1.3 (Oct 2025)\n")
	b.WriteString("\n")
	b.WriteString("[install]\n")
	if registryProxy != "" {
		b.WriteString("# Corporate registry proxy\n")
		fmt.Fprintf(&b, "registry = \"%s\"\n", ecosystem.TOMLEscapeString(ecosystem.INIEscapeValue(registryProxy)))
	}
	b.WriteString("# Disable lifecycle scripts, including Bun's built-in list of\n")
	b.WriteString("# default-trusted packages that otherwise run postinstall hooks\n")
	b.WriteString("ignoreScripts = true\n")
	b.WriteString("# Require packages to be published for at least 7 days\n")
	fmt.Fprintf(&b, "minimumReleaseAge = %d\n", bunMinimumReleaseAgeSeconds)

	return types.GeneratedFile{
		Path:     "bunfig.toml",
		Content:  []byte(b.String()),
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.Skip,
	}
}
