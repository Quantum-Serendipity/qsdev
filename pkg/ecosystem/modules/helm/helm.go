// Package helm implements the Helm ecosystem module for qsdev.
// It detects Helm chart projects by scanning for Chart.yaml and Chart.lock,
// then generates devenv.nix fragments with helm and kubeconform packages,
// pre-commit hooks for helm lint, deny rules, and CI commands for a hardened
// Kubernetes Helm chart development environment.
package helm

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.PackageProvider = (*Module)(nil)
var _ ecosystem.DenyRuleProvider = (*Module)(nil)
var _ ecosystem.ReadDenyRuleProvider = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// chartVersionExtra is the ModuleConfig.Extras key holding the chart's own
// version. It is not the Helm tool version, so it does not go in
// ModuleConfig.Version.
const chartVersionExtra = "chart_version"

// Module is the stateless Helm ecosystem module.
type Module struct{}

// Name returns the canonical module identifier.
func (m *Module) Name() string { return ecosystem.NameHelm }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Helm" }

// Tier returns the implementation priority tier (2 = standard).
func (m *Module) Tier() int { return 2 }

// Detect scans projectRoot and its subdirectories (up to
// ecosystem.ProjectScanDepth levels, so the charts/<name>/ layout `helm
// create` and monorepos use is found) for Helm chart indicators.
// Chart.yaml yields Certain confidence; Chart.lock alone yields Probable.
// Chart directories below the root are recorded in Extras[ExtraChartDirs].
// The chart version is extracted from the root Chart.yaml when present and
// reported in Extras["chart_version"].
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	chartDirs := ecosystem.ProjectDirsWith(projectRoot, func(name string) bool { return name == "Chart.yaml" })
	if len(chartDirs) > 0 {
		result := ecosystem.DetectionResult{
			Detected:   true,
			Confidence: ecosystem.ConfidenceCertain,
			Evidence:   []string{"Chart.yaml found"},
		}
		extras := map[string]string{}
		if dirs := ecosystem.ShellSafeDirs(chartDirs); len(dirs) > 0 && !slices.Equal(dirs, []string{"."}) {
			extras[ExtraChartDirs] = strings.Join(dirs, ",")
			result.Evidence = append(result.Evidence, "charts in: "+strings.Join(dirs, ", "))
		}
		if version := parseChartVersion(filepath.Join(projectRoot, "Chart.yaml")); version != "" {
			result.Evidence = append(result.Evidence, fmt.Sprintf("chart version %s", version))
			extras[chartVersionExtra] = version
		}
		if len(extras) > 0 {
			result.SuggestedConfig.Extras = extras
		}
		return result
	}

	if len(ecosystem.ProjectDirsWith(projectRoot, func(name string) bool { return name == "Chart.lock" })) > 0 {
		return ecosystem.DetectionResult{
			Detected:   true,
			Confidence: ecosystem.ConfidenceProbable,
			Evidence:   []string{"Chart.lock found"},
		}
	}

	return ecosystem.DetectionAbsent()
}

// ExtraChartDirs is the ModuleConfig.Extras key holding the comma-separated
// chart directories (relative to the project root, "." for the root). It is
// unset when the only chart is the root one.
const ExtraChartDirs = "chart_dirs"

// DevenvPackages returns the Nix packages required for the Helm ecosystem.
func (m *Module) DevenvPackages(_ ecosystem.ModuleConfig) []string {
	return []string{"kubernetes-helm", "kubeconform"}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Helm support. Packages are provided via DevenvPackages.
func (m *Module) DevenvNixFragment(_ ecosystem.ModuleConfig) (string, error) {
	return "", nil
}

// SecurityConfigs returns nil. OCI registry configuration is
// infrastructure-profile dependent and not handled at the module level.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// PreCommitHooks returns pre-commit hook definitions for the Helm ecosystem.
// `helm lint` checks the recorded chart directories (ExtraChartDirs), or the
// root chart when none are recorded.
func (m *Module) PreCommitHooks(config ecosystem.ModuleConfig) []ecosystem.HookConfig {
	entry := "helm lint"
	if raw := config.Extra(ExtraChartDirs, ""); raw != "" {
		if dirs := ecosystem.ShellSafeDirs(strings.Split(raw, ",")); len(dirs) > 0 {
			entry += " " + strings.Join(dirs, " ")
		}
	}
	return []ecosystem.HookConfig{
		{
			ID:            "helmlint",
			Name:          "helmlint",
			Description:   "Lint Helm charts with helm lint",
			Entry:         entry,
			Language:      "system",
			Types:         []string{"yaml"},
			Stages:        []string{"pre-commit"},
			Files:         `Chart\.yaml$`,
			PassFilenames: false,
			BuiltIn:       false,
			NixPackage:    "kubernetes-helm",
		},
	}
}

// deniedSubcommands are the helm subcommands the agent must not run: ones
// that change cluster state (install, upgrade, uninstall and its aliases
// delete/del/un, rollback), run third-party code (plugin install/add, plugin
// update/up), re-resolve dependencies past Chart.lock (dependency update, in
// every alias spelling), add chart sources (repo add), or print release values
// and manifests that carry secrets (get values/all/manifest).
var deniedSubcommands = []string{
	"install",
	"upgrade",
	"uninstall",
	"delete",
	"del",
	"un",
	"rollback",
	"plugin install",
	"plugin add",
	"plugin update",
	"plugin up",
	"dependency update",
	"dependency up",
	"dep update",
	"dep up",
	"dependencies update",
	"dependencies up",
	"repo add",
	"get values",
	"get all",
	"get manifest",
}

// kubeconfigDenyRules keep the agent from printing the kubeconfig helm uses,
// which carries cluster credentials: `kubectl config view --raw` and reading
// it with cat.
var kubeconfigDenyRules = []string{
	"Bash(kubectl *config *view*--raw*)",
	"Bash(env *kubectl *config *view*--raw*)",
	"Bash(cat ~/.kube/*)",
}

// DenyRules returns Claude Code deny-rule patterns for the Helm ecosystem.
// Each subcommand in deniedSubcommands is denied plain, after global flags
// such as --kube-context or -n, and behind env (denyutil.SubcommandRules);
// kubeconfigDenyRules cover printing the cluster credentials.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return append(denyutil.SubcommandRules("helm", deniedSubcommands...), kubeconfigDenyRules...)
}

// ReadDenyRules returns the credential stores the agent's Read tool must not
// open: the kubeconfig files helm talks to clusters with (~/.kube/config and
// the per-cluster files commonly kept beside it), and helm's OCI registry and
// chart repository credentials.
func (m *Module) ReadDenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"~/.kube/*",
		"~/.config/helm/registry/*",
		"~/.config/helm/repositories.yaml",
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

// VerificationCommands returns an empty set. Helm does not define standard
// verification commands at the module level.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{}
}

// Compile-time check that the Helm module declares its manifest.
var _ ecosystem.ManifestFileProvider = (*Module)(nil)

// ManifestFiles declares Chart.yaml and Chart.lock so Version-Sentinel
// coverage reports list chart dependencies as uncovered instead of omitting
// them.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return []ecosystem.ManifestFileInfo{{
		Path:           "Chart.yaml",
		Ecosystem:      "helm",
		LockFile:       "Chart.lock",
		LockFilePolicy: ecosystem.LockFilePolicyRecommended,
	}}
}

// Compile-time check that the Helm module reports chart dependencies.
var _ ecosystem.DependencyDeclarer = (*Module)(nil)

// DeclaresDependencies reports whether Chart.yaml lists any dependencies.
// `helm dependency update` writes Chart.lock only for charts that have some,
// so a chart without dependencies has no lock file to enforce.
func (m *Module) DeclaresDependencies(projectRoot string) (bool, error) {
	data, err := os.ReadFile(filepath.Join(projectRoot, "Chart.yaml")) //nolint:gosec // project-root manifest
	if err != nil {
		return false, fmt.Errorf("reading Chart.yaml: %w", err)
	}
	var chart struct {
		Dependencies []yaml.Node `yaml:"dependencies"`
	}
	if err := yaml.Unmarshal(data, &chart); err != nil {
		return false, fmt.Errorf("parsing Chart.yaml: %w", err)
	}
	return len(chart.Dependencies) > 0, nil
}

// parseChartVersion reads Chart.yaml and returns its top-level version field.
// Parsing the YAML (rather than matching lines) ignores the version keys of
// entries under dependencies and strips quoting and trailing comments.
// Returns an empty string if the field is missing or the file cannot be read
// or parsed.
func parseChartVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var chart struct {
		Version string `yaml:"version"`
	}
	if err := yaml.Unmarshal(data, &chart); err != nil {
		return ""
	}
	return chart.Version
}
