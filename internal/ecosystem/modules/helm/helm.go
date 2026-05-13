// Package helm implements the Helm ecosystem module for gdev-secure-devenv-bootstrap.
// It detects Helm chart projects by scanning for Chart.yaml and Chart.lock,
// then generates devenv.nix fragments with helm and kubeconform packages,
// pre-commit hooks for helm lint, deny rules, and CI commands for a hardened
// Kubernetes Helm chart development environment.
package helm

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/fileutil"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*Module)(nil)

func init() {
	ecosystem.RegisterModule(&Module{})
}

// chartVersionRe matches the "version: X.Y.Z" line in Chart.yaml.
var chartVersionRe = regexp.MustCompile(`^\s*version:\s*(.+)$`)

// Module is the stateless Helm ecosystem module.
type Module struct{}

// Name returns the canonical module identifier.
func (m *Module) Name() string { return "helm" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Helm" }

// Tier returns the implementation priority tier (2 = standard).
func (m *Module) Tier() int { return 2 }

// Detect scans projectRoot for Helm chart indicators.
// Chart.yaml yields Certain confidence; Chart.lock alone yields Probable.
// The chart version is extracted from Chart.yaml when present.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	chartPath := filepath.Join(projectRoot, "Chart.yaml")
	lockPath := filepath.Join(projectRoot, "Chart.lock")

	if fileutil.FileExists(chartPath) {
		evidence := []string{"Chart.yaml found"}
		version := parseChartVersion(chartPath)
		if version != "" {
			evidence = append(evidence, fmt.Sprintf("chart version %s", version))
		}
		return ecosystem.DetectionResult{
			Detected:   true,
			Confidence: ecosystem.ConfidenceCertain,
			Evidence:   evidence,
			SuggestedConfig: ecosystem.ModuleConfig{
				Version: version,
			},
		}
	}

	if fileutil.FileExists(lockPath) {
		return ecosystem.DetectionResult{
			Detected:   true,
			Confidence: ecosystem.ConfidenceProbable,
			Evidence:   []string{"Chart.lock found"},
		}
	}

	return ecosystem.DetectionResult{
		Detected:   false,
		Confidence: ecosystem.ConfidenceAbsent,
	}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Helm support. Helm uses a packages-based approach (no languages.helm module).
func (m *Module) DevenvNixFragment(_ ecosystem.ModuleConfig) (string, error) {
	var b strings.Builder
	b.WriteString("  packages = with pkgs; [ kubernetes-helm kubeconform ];\n")
	return b.String(), nil
}

// DevenvYamlInputs returns additional flake inputs for devenv.yaml.
// Helm does not require any additional inputs.
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return nil
}

// SecurityConfigs returns nil. OCI registry configuration is
// infrastructure-profile dependent and not handled at the module level.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// PreCommitHooks returns pre-commit hook definitions for the Helm ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "helmlint",
			Name:          "helmlint",
			Description:   "Lint Helm charts with helm lint",
			Entry:         "helm lint",
			Language:      "system",
			Types:         []string{"yaml"},
			Stages:        []string{"pre-commit"},
			Files:         `Chart\.yaml$`,
			PassFilenames: false,
			BuiltIn:       true,
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the Helm ecosystem.
// These prevent direct helm install/upgrade outside of controlled workflows.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"Bash(helm install *)",
		"Bash(helm upgrade *)",
	}
}

// CICommands returns CI pipeline commands for the Helm ecosystem.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name:        "helm-dependency-build",
			Command:     "helm dependency build",
			Description: "Build Helm chart dependencies from Chart.lock",
			Phase:       ecosystem.CIPhaseInstall,
		},
		{
			Name:        "helm-lint",
			Command:     "helm lint .",
			Description: "Lint Helm chart for best practices and errors",
			Phase:       ecosystem.CIPhaseTest,
		},
		{
			Name:        "helm-template-validate",
			Command:     "helm template . | kubeconform --strict",
			Description: "Validate rendered Helm templates against Kubernetes schemas",
			Phase:       ecosystem.CIPhaseScan,
		},
	}
}

// PackageManagers returns metadata about the Helm dependency system.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:                 "helm",
			LockFile:             "Chart.lock",
			InstallCommand:       "helm dependency update",
			FrozenInstallCommand: "helm dependency build",
			AgeGatingSupport:     false,
		},
	}
}

// WizardFields returns nil. Helm does not require additional wizard fields.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return nil
}

// VerificationCommands returns an empty set. Helm does not define standard
// verification commands at the module level.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{}
}

// ManifestFiles returns nil. Helm does not use a traditional manifest file.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return nil
}

// parseChartVersion reads Chart.yaml and extracts the version field using a
// simple line-based regex. Returns an empty string if the field is not found
// or the file cannot be read.
func parseChartVersion(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close() //nolint:errcheck // best-effort read

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if matches := chartVersionRe.FindStringSubmatch(scanner.Text()); matches != nil {
			return strings.TrimSpace(matches[1])
		}
	}
	return ""
}
