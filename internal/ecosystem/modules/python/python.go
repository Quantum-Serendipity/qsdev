// Package python implements the Python ecosystem module for gdev-secure-devenv-bootstrap.
// It detects Python projects by scanning for pyproject.toml, requirements.txt, setup.py,
// and Pipfile, generates devenv.nix fragments with package manager integration, and
// provides security-hardened pip.conf, pre-commit hooks, CI commands, deny rules,
// and wizard fields for the Python toolchain.
package python

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*Module)(nil)

func init() {
	if err := ecosystem.DefaultRegistry().Register(&Module{}); err != nil {
		panic(fmt.Sprintf("python: failed to register ecosystem module: %v", err))
	}
}

// requiresPythonRe matches the requires-python line in pyproject.toml and
// captures the first version number (major.minor).
var requiresPythonRe = regexp.MustCompile(`^\s*requires-python\s*=\s*"[><=!~]*(\d+\.\d+)`)

// Module implements ecosystem.EcosystemModule for the Python programming language.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "python" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Python" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 1 }

// Detect scans projectRoot for Python ecosystem indicators and returns a DetectionResult.
// It checks for pyproject.toml (Certain), requirements.txt (Probable), setup.py (Probable),
// and Pipfile (Probable). The highest confidence level wins. Package manager is inferred
// from lockfiles: uv.lock -> uv, poetry.lock -> poetry, otherwise pip. Version is read
// from .python-version (priority) or pyproject.toml requires-python.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	confidence := ecosystem.ConfidenceAbsent
	var evidence []string

	// Check indicators in order of confidence.
	if fileExists(filepath.Join(projectRoot, "pyproject.toml")) {
		confidence = ecosystem.ConfidenceCertain
		evidence = append(evidence, "pyproject.toml found")
	}
	if fileExists(filepath.Join(projectRoot, "requirements.txt")) {
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		evidence = append(evidence, "requirements.txt found")
	}
	if fileExists(filepath.Join(projectRoot, "setup.py")) {
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		evidence = append(evidence, "setup.py found")
	}
	if fileExists(filepath.Join(projectRoot, "Pipfile")) {
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		evidence = append(evidence, "Pipfile found")
	}

	if confidence == ecosystem.ConfidenceAbsent {
		return ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		}
	}

	// Determine package manager from lockfiles.
	pm := "pip"
	if fileExists(filepath.Join(projectRoot, "uv.lock")) {
		pm = "uv"
		evidence = append(evidence, "uv.lock found")
	} else if fileExists(filepath.Join(projectRoot, "poetry.lock")) {
		pm = "poetry"
		evidence = append(evidence, "poetry.lock found")
	}

	// Determine version: .python-version takes priority over pyproject.toml.
	version := ""
	if v := readFirstLine(filepath.Join(projectRoot, ".python-version")); v != "" {
		version = v
		evidence = append(evidence, fmt.Sprintf("python version %s (from .python-version)", v))
	} else if v := parseRequiresPython(filepath.Join(projectRoot, "pyproject.toml")); v != "" {
		version = v
		evidence = append(evidence, fmt.Sprintf("python version %s (from pyproject.toml requires-python)", v))
	}

	return ecosystem.DetectionResult{
		Detected:   true,
		Confidence: confidence,
		Evidence:   evidence,
		SuggestedConfig: ecosystem.ModuleConfig{
			Version:        version,
			PackageManager: pm,
		},
	}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Python language support with the configured package manager.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	version := config.Version
	if version == "" {
		version = "3.12"
	}

	pm := config.PackageManager
	if pm == "" {
		pm = "pip"
	}

	var b strings.Builder
	b.WriteString("  languages.python = {\n")
	b.WriteString("    enable = true;\n")
	fmt.Fprintf(&b, "    version = %q;\n", version)

	switch pm {
	case "uv":
		b.WriteString("    uv.enable = true;\n")
	case "poetry":
		b.WriteString("    poetry.enable = true;\n")
	}

	b.WriteString("    venv.enable = true;\n")
	b.WriteString("  };\n")
	return b.String(), nil
}

// DevenvYamlInputs returns additional flake inputs for devenv.yaml.
// Python does not require any additional inputs.
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return nil
}

// SecurityConfigs returns generated security configuration files.
// For pip, it generates a security-hardened pip.conf. For uv and poetry,
// security is enforced via CI commands, so no config files are needed.
func (m *Module) SecurityConfigs(config ecosystem.ModuleConfig) []types.GeneratedFile {
	pm := config.PackageManager
	if pm == "" {
		pm = "pip"
	}

	if pm != "pip" {
		return nil
	}

	content := `# Security-hardened pip configuration
# Generated by gdev
# Note: age-gating via uploaded-prior-to requires pip >= 26.0 (Jan 2026)
# For uv, use --exclude-newer=7d (requires uv >= 0.9.17, Dec 2025)
#
# require-hashes: Enforces hash verification for all installed packages.
# only-binary: Blocks source distributions that execute setup.py during install.

[global]
require-hashes = true
only-binary = :all:
`

	return []types.GeneratedFile{
		{
			Path:     "pip.conf",
			Content:  []byte(content),
			Mode:     0o644,
			Strategy: types.Overwrite,
		},
	}
}

// PreCommitHooks returns pre-commit hook definitions for the Python ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "ruff",
			Name:          "ruff",
			Description:   "Run ruff linter and formatter for Python",
			Entry:         "ruff check --fix",
			Language:      "python",
			Types:         []string{"python"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       true,
		},
		{
			ID:            "mypy",
			Name:          "mypy",
			Description:   "Run mypy type checker for Python",
			Entry:         "mypy",
			Language:      "python",
			Types:         []string{"python"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       true,
		},
		{
			ID:            "bandit",
			Name:          "bandit",
			Description:   "Run bandit security SAST scanner for Python",
			Entry:         "bandit -r",
			Language:      "python",
			Types:         []string{"python"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       true,
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the Python ecosystem.
// These prevent direct dependency installation outside of controlled workflows.
// Note: uv sync and poetry install (lockfile-based installs) are intentionally allowed.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"Bash(pip install *)",
		"Bash(pip uninstall *)",
		"Bash(pip3 install *)",
		"Bash(pip3 uninstall *)",
		"Bash(python -m pip install *)",
		"Bash(python -m pip uninstall *)",
		"Bash(python3 -m pip install *)",
		"Bash(python3 -m pip uninstall *)",
		"Bash(uv pip install *)",
		"Bash(uv add *)",
		"Bash(poetry add *)",
		"Bash(poetry remove *)",
	}
}

// CICommands returns CI pipeline commands for the Python ecosystem.
// Commands vary based on the configured package manager.
func (m *Module) CICommands(config ecosystem.ModuleConfig) []ecosystem.CICommand {
	pm := config.PackageManager
	if pm == "" {
		pm = "pip"
	}

	var cmds []ecosystem.CICommand

	switch pm {
	case "pip":
		cmds = append(cmds, ecosystem.CICommand{
			Name:        "pip-install",
			Command:     "pip install --require-hashes --only-binary :all: -r requirements.txt",
			Description: "Install Python dependencies with hash verification and binary-only constraint",
			Phase:       ecosystem.CIPhaseInstall,
		})
	case "uv":
		cmds = append(cmds, ecosystem.CICommand{
			Name:        "uv-sync",
			Command:     "uv sync --frozen --exclude-newer=7d",
			Description: "Install Python dependencies from frozen uv lockfile with 7-day age gate",
			Phase:       ecosystem.CIPhaseInstall,
		})
	case "poetry":
		cmds = append(cmds, ecosystem.CICommand{
			Name:        "poetry-install",
			Command:     "poetry install --no-interaction",
			Description: "Install Python dependencies from poetry lockfile",
			Phase:       ecosystem.CIPhaseInstall,
		})
	}

	cmds = append(cmds,
		ecosystem.CICommand{
			Name:        "pip-audit",
			Command:     "pip-audit",
			Description: "Audit Python dependencies for known vulnerabilities",
			Phase:       ecosystem.CIPhaseScan,
		},
		ecosystem.CICommand{
			Name:        "safety-check",
			Command:     "safety check",
			Description: "Run safety check for Python dependency vulnerabilities",
			Phase:       ecosystem.CIPhaseScan,
		},
	)

	return cmds
}

// PackageManagers returns metadata about Python's package managers.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:                 "pip",
			LockFile:             "requirements.txt",
			InstallCommand:       "pip install -r requirements.txt",
			FrozenInstallCommand: "pip install --require-hashes --only-binary :all: -r requirements.txt",
			AuditCommand:         "pip-audit",
			AgeGatingSupport:     false,
		},
		{
			Name:                 "uv",
			LockFile:             "uv.lock",
			InstallCommand:       "uv sync",
			FrozenInstallCommand: "uv sync --frozen",
			AuditCommand:         "pip-audit",
			AgeGatingSupport:     true,
		},
		{
			Name:                 "poetry",
			LockFile:             "poetry.lock",
			InstallCommand:       "poetry install",
			FrozenInstallCommand: "poetry install --no-interaction",
			AuditCommand:         "pip-audit",
			AgeGatingSupport:     false,
		},
	}
}

// WizardFields returns additional wizard form fields for Python configuration.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         "python_package_manager",
			Label:       "Package manager",
			Description: "Select the Python package manager to use",
			Type:        ecosystem.FieldTypeSelect,
			Options: []ecosystem.WizardOption{
				{Label: "pip", Value: "pip"},
				{Label: "uv", Value: "uv"},
				{Label: "poetry", Value: "poetry"},
			},
			Default:  "pip",
			Required: true,
		},
		{
			Key:         "python_venv",
			Label:       "Enable virtual environment",
			Description: "Create and activate a Python virtual environment",
			Type:        ecosystem.FieldTypeConfirm,
			Default:     "true",
		},
	}
}

// fileExists reports whether a file at the given path exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// readFirstLine reads the first non-empty trimmed line from the file at path.
// Returns an empty string if the file cannot be read or is empty.
func readFirstLine(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close() //nolint:errcheck // best-effort read

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			return line
		}
	}
	return ""
}

// parseRequiresPython reads pyproject.toml and extracts the Python version
// from the requires-python field. Returns an empty string if the field
// is not found or the file cannot be read.
func parseRequiresPython(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close() //nolint:errcheck // best-effort read

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if matches := requiresPythonRe.FindStringSubmatch(scanner.Text()); matches != nil {
			return matches[1]
		}
	}
	return ""
}
