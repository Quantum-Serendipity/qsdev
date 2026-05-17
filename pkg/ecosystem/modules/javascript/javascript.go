package javascript

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Module implements ecosystem.EcosystemModule for the JavaScript/TypeScript ecosystem.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "javascript" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "JavaScript/TypeScript" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 1 }

// Detect scans projectRoot for JavaScript/TypeScript indicators.
// It checks for package.json (Certain confidence), determines the package manager
// from lockfiles, reads Node.js version from .nvmrc or package.json engines,
// and checks for TypeScript via tsconfig.json.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	pkgJSONPath := filepath.Join(projectRoot, "package.json")
	if !fileutil.FileExists(pkgJSONPath) {
		return ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		}
	}

	evidence := []string{"package.json found"}

	// Determine package manager from lockfiles.
	pm := detectPackageManager(projectRoot)
	evidence = append(evidence, fmt.Sprintf("package manager: %s", pm))

	// Determine Node.js version: .nvmrc takes priority over engines.node.
	version := ""
	nvmrcPath := filepath.Join(projectRoot, ".nvmrc")
	if fileutil.FileExists(nvmrcPath) {
		version = strings.TrimPrefix(fileutil.ReadFirstLine(nvmrcPath), "v")
		if version != "" {
			evidence = append(evidence, fmt.Sprintf("node version %s (from .nvmrc)", version))
		}
	}
	if version == "" {
		version = nodeVersionFromPackageJSON(pkgJSONPath)
		if version != "" {
			evidence = append(evidence, fmt.Sprintf("node version %s (from engines.node)", version))
		}
	}

	// Check for TypeScript.
	extras := make(map[string]string)
	tsconfigPath := filepath.Join(projectRoot, "tsconfig.json")
	if fileutil.FileExists(tsconfigPath) {
		extras["typescript"] = "true"
		evidence = append(evidence, "tsconfig.json found")
	}

	return ecosystem.DetectionResult{
		Detected:   true,
		Confidence: ecosystem.ConfidenceCertain,
		Evidence:   evidence,
		SuggestedConfig: ecosystem.ModuleConfig{
			Version:        version,
			PackageManager: pm,
			Extras:         extras,
		},
	}
}

// detectPackageManager determines the package manager by inspecting lockfiles.
// Priority: pnpm-lock.yaml > yarn.lock > bun.lock/bun.lockb > package-lock.json > npm (default).
func detectPackageManager(projectRoot string) string {
	if fileutil.FileExists(filepath.Join(projectRoot, "pnpm-lock.yaml")) {
		return "pnpm"
	}
	if fileutil.FileExists(filepath.Join(projectRoot, "yarn.lock")) {
		return "yarn"
	}
	if fileutil.FileExists(filepath.Join(projectRoot, "bun.lock")) || fileutil.FileExists(filepath.Join(projectRoot, "bun.lockb")) {
		return "bun"
	}
	if fileutil.FileExists(filepath.Join(projectRoot, "package-lock.json")) {
		return "npm"
	}
	return "npm"
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for JavaScript/TypeScript support.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	major := extractMajorVersion(config.Version)
	nodePkg := nodeNixPackage(major)

	pm := config.PackageManager
	if pm == "" {
		pm = "npm"
	}

	var b strings.Builder
	b.WriteString("  languages.javascript = {\n")
	b.WriteString("    enable = true;\n")
	fmt.Fprintf(&b, "    package = %s;\n", nodePkg)

	// npm is enabled alongside Node.js by default.
	if pm == "npm" {
		b.WriteString("    npm.enable = true;\n")
	}

	b.WriteString("  };\n")

	// Package manager specific configuration.
	switch pm {
	case "pnpm":
		b.WriteString("\n  languages.javascript.pnpm.enable = true;\n")
	case "yarn":
		b.WriteString("\n  languages.javascript.yarn.enable = true;\n")
	case "bun":
		b.WriteString("\n  languages.bun.enable = true;\n")
	}

	// TypeScript support.
	if config.Extras["typescript"] == "true" {
		b.WriteString("\n  languages.typescript.enable = true;\n")
	}

	return b.String(), nil
}

// DevenvYamlInputs returns additional flake inputs for devenv.yaml.
// JavaScript does not require any additional inputs.
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return nil
}

// PreCommitHooks returns pre-commit hook definitions for the JavaScript/TypeScript ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "prettier",
			Name:          "prettier",
			Description:   "Format JavaScript/TypeScript code with Prettier",
			Entry:         "prettier --write --list-different",
			Language:      "node",
			Types:         []string{"javascript", "typescript", "json", "css"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       true,
		},
		{
			ID:            "eslint",
			Name:          "eslint",
			Description:   "Lint JavaScript/TypeScript code with ESLint",
			Entry:         "eslint --fix",
			Language:      "node",
			Types:         []string{"javascript", "typescript"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       true,
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the JavaScript/TypeScript ecosystem.
// These cover ALL four package managers regardless of which one is detected,
// plus pipe-to-shell patterns that are common JS supply chain attack vectors.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	// Package install commands (npm/pnpm/yarn/bun add/install) are handled by
	// base ask rules + package-guard hook. Only hard-deny patterns here that
	// must never execute regardless of hook validation.
	return []string{
		"Bash(npx *)",
		"Bash(curl * | sh*)",
		"Bash(curl * | bash*)",
		"Bash(wget * | sh*)",
		"Bash(wget * | bash*)",
	}
}

// CICommands returns CI pipeline commands for the JavaScript/TypeScript ecosystem.
// The frozen install command depends on the detected package manager.
func (m *Module) CICommands(config ecosystem.ModuleConfig) []ecosystem.CICommand {
	pm := config.PackageManager
	if pm == "" {
		pm = "npm"
	}

	var installCmd string
	switch pm {
	case "npm":
		installCmd = "npm ci --ignore-scripts"
	case "pnpm":
		installCmd = "pnpm install --frozen-lockfile"
	case "yarn":
		installCmd = "yarn install --immutable"
	case "bun":
		installCmd = "bun install --frozen-lockfile"
	default:
		installCmd = "npm ci --ignore-scripts"
	}

	return []ecosystem.CICommand{
		{
			Name:        fmt.Sprintf("%s-install", pm),
			Command:     installCmd,
			Description: fmt.Sprintf("Install dependencies using %s with frozen lockfile", pm),
			Phase:       ecosystem.CIPhaseInstall,
		},
	}
}

// PackageManagers returns metadata about all JavaScript package managers.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:                 "npm",
			LockFile:             "package-lock.json",
			InstallCommand:       "npm install",
			FrozenInstallCommand: "npm ci",
			AuditCommand:         "npm audit",
			AgeGatingSupport:     true,
		},
		{
			Name:                 "pnpm",
			LockFile:             "pnpm-lock.yaml",
			InstallCommand:       "pnpm install",
			FrozenInstallCommand: "pnpm install --frozen-lockfile",
			AuditCommand:         "pnpm audit",
			AgeGatingSupport:     true,
		},
		{
			Name:                 "yarn",
			LockFile:             "yarn.lock",
			InstallCommand:       "yarn install",
			FrozenInstallCommand: "yarn install --immutable",
			AuditCommand:         "yarn npm audit",
			AgeGatingSupport:     true,
		},
		{
			Name:                 "bun",
			LockFile:             "bun.lock",
			InstallCommand:       "bun install",
			FrozenInstallCommand: "bun install --frozen-lockfile",
			AuditCommand:         "",
			AgeGatingSupport:     true,
		},
	}
}

// WizardFields returns additional wizard form fields for JavaScript/TypeScript configuration.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         "package_manager",
			Label:       "Package manager",
			Description: "Select the JavaScript package manager for this project",
			Type:        ecosystem.FieldTypeSelect,
			Options: []ecosystem.WizardOption{
				{Label: "npm", Value: "npm"},
				{Label: "pnpm", Value: "pnpm"},
				{Label: "Yarn", Value: "yarn"},
				{Label: "Bun", Value: "bun"},
			},
			Default:  "npm",
			Required: true,
		},
		{
			Key:         "typescript",
			Label:       "TypeScript",
			Description: "Enable TypeScript support",
			Type:        ecosystem.FieldTypeConfirm,
			Default:     "false",
		},
	}
}

// VerificationCommands returns project verification commands for the JavaScript/TypeScript ecosystem.
func (m *Module) VerificationCommands(config ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	pm := config.PackageManager
	if pm == "" {
		pm = "npm"
	}
	vc := ecosystem.VerificationCommands{
		Build:  []string{pm + " run build"},
		Test:   []string{pm + " test"},
		Lint:   []string{pm + " run lint"},
		Format: []string{"prettier --check ."},
	}
	if pm == "bun" {
		vc.Test = []string{"bun run test"}
	}
	return vc
}

// ManifestFiles returns manifest file metadata for the JavaScript/TypeScript ecosystem.
func (m *Module) ManifestFiles(config ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	pm := config.PackageManager
	if pm == "" {
		pm = "npm"
	}
	info := ecosystem.ManifestFileInfo{
		Path:           "package.json",
		Ecosystem:      pm,
		VSSupported:    true,
		LockFilePolicy: ecosystem.LockFilePolicyRequired,
	}
	switch pm {
	case "npm":
		info.LockFile = "package-lock.json"
	case "pnpm":
		info.LockFile = "pnpm-lock.yaml"
	case "yarn":
		info.LockFile = "yarn.lock"
	case "bun":
		info.LockFile = "bun.lock"
	default:
		info.LockFile = "package-lock.json"
	}
	return []ecosystem.ManifestFileInfo{info}
}

// SemgrepRuleSets returns Semgrep rule set identifiers relevant to JavaScript/TypeScript projects.
func (m *Module) SemgrepRuleSets() []string {
	return []string{"p/typescript", "p/javascript", "p/react", "p/nextjs", "p/owasp-top-ten", "p/xss"}
}
