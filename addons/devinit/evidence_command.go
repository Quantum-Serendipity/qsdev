package devinit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/evidence"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/posture"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/version"
)

func evidenceCmd() *cobra.Command {
	var (
		framework      string
		format         string
		listFrameworks bool
	)

	cmd := &cobra.Command{
		Use:   "evidence",
		Short: "Generate compliance evidence report mapping gdev controls to a framework",
		Long: `Generate a compliance evidence report that maps gdev's defense-in-depth
layers to controls in a compliance framework (SOC2, HIPAA, ASVS, etc.).

The report shows which framework controls are addressed, partially addressed,
or not addressed by the current gdev configuration. Output is available in
JSON or Markdown format.

Use --list-frameworks to see available compliance frameworks.
Use --framework to select a specific framework (required unless --list-frameworks).
Use --format to select output format (json or md).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if listFrameworks {
				return runListFrameworks(cmd)
			}
			if framework == "" {
				return fmt.Errorf("--framework is required; use --list-frameworks to see available frameworks")
			}
			return runEvidence(cmd, framework, format)
		},
	}

	cmd.Flags().StringVar(&framework, "framework", "", "Compliance framework ID (e.g., soc2, hipaa, asvs)")
	cmd.Flags().StringVar(&format, "format", "json", "Output format: json, md")
	cmd.Flags().BoolVar(&listFrameworks, "list-frameworks", false, "List available compliance frameworks")

	return cmd
}

func runListFrameworks(cmd *cobra.Command) error {
	registry := evidence.DefaultRegistry()
	frameworks := registry.List()

	if len(frameworks) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No compliance frameworks available.")
		return nil
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-10s  %-25s  %-10s  %s\n", "ID", "Name", "Version", "Description")
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), strings.Repeat("-", 90))
	for _, f := range frameworks {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-10s  %-25s  %-10s  %s\n",
			f.ID, f.Name, f.Version, f.Description)
	}
	return nil
}

func runEvidence(cmd *cobra.Command, frameworkID, format string) error {
	// Validate format.
	switch format {
	case "json", "md":
		// valid
	default:
		return fmt.Errorf("unsupported format %q; use json or md", format)
	}

	// Look up framework.
	registry := evidence.DefaultRegistry()
	fw, ok := registry.Get(frameworkID)
	if !ok {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Unknown framework %q. Available frameworks:\n", frameworkID)
		_ = runListFrameworks(cmd)
		return fmt.Errorf("unknown framework %q", frameworkID)
	}

	// Build a posture report for the current project.
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("determining project root: %w", err)
	}

	projectName := filepath.Base(projectRoot)
	report := buildMinimalPostureReport(projectRoot, projectName)

	// Generate evidence report.
	evidenceReport, err := evidence.Generate(fw, report, projectName)
	if err != nil {
		return fmt.Errorf("generating evidence report: %w", err)
	}

	// Render output.
	switch format {
	case "json":
		return evidence.RenderJSON(evidenceReport, cmd.OutOrStdout())
	case "md":
		return evidence.RenderMarkdown(evidenceReport, cmd.OutOrStdout())
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

// buildMinimalPostureReport constructs a PostureReport with defense layers
// assessed from the current project state. This is a lightweight assessment
// that populates the defense coverage needed for evidence generation without
// requiring the full posture assessment pipeline.
func buildMinimalPostureReport(projectRoot, projectName string) *posture.PostureReport {
	// Read state file to determine which tools are enabled.
	enabledTools := discoverEnabledTools(projectRoot)

	// Build defense layers from canonical names, checking tool enablement.
	layers := buildDefenseLayers(enabledTools, projectRoot)

	enabled := 0
	total := 0
	for _, l := range layers {
		if l.Status == posture.LayerNotApplicable {
			continue
		}
		total++
		if l.Status == posture.LayerEnabled {
			enabled++
		}
	}

	score := posture.ComputeDefenseScore(layers)

	return &posture.PostureReport{
		SchemaVersion: posture.SchemaVersion,
		GeneratedAt:   time.Now().UTC(),
		GdevVersion:   version.Info().Version,
		ProjectPath:   projectRoot,
		ProjectName:   projectName,
		Defense: posture.DefenseCoverage{
			Score:   score,
			Enabled: enabled,
			Total:   total,
			Layers:  layers,
		},
	}
}

// discoverEnabledTools reads the project state to determine which security
// tools are enabled. Returns a map of tool name -> enabled.
func discoverEnabledTools(projectRoot string) map[string]bool {
	tools := make(map[string]bool)

	// Check for common configuration files that indicate tool enablement.
	checks := map[string][]string{
		"semgrep":           {".semgrep.yml", ".semgrep.yaml"},
		"gitleaks":          {".gitleaks.toml", ".gitleaks.yaml"},
		"ripsecrets":        {".ripsecrets.toml"},
		"attach-guard":      {".claude/hooks/package-guard.py", ".claude/settings.json"},
		"container-security": {"Dockerfile"},
		"license-compliance": {".licensecompliance.yml", ".license-compliance.yaml"},
		"socket-dev-mcp":    {".socket.yml"},
	}

	for tool, paths := range checks {
		for _, p := range paths {
			if _, err := os.Stat(filepath.Join(projectRoot, p)); err == nil {
				tools[tool] = true
				break
			}
		}
	}

	return tools
}

// buildDefenseLayers creates defense layer assessments for all 10 canonical layers.
func buildDefenseLayers(enabledTools map[string]bool, projectRoot string) []posture.DefenseLayer {
	layers := make([]posture.DefenseLayer, 0, len(posture.DefenseLayerNames))

	for _, name := range posture.DefenseLayerNames {
		layer := assessLayerForEvidence(name, enabledTools, projectRoot)
		layers = append(layers, layer)
	}

	return layers
}

// assessLayerForEvidence performs a simplified layer assessment suitable for
// evidence report generation.
func assessLayerForEvidence(name string, enabledTools map[string]bool, projectRoot string) posture.DefenseLayer {
	layer := posture.DefenseLayer{
		Name: name,
	}

	switch name {
	case "pretooluse-hooks":
		layer.Weight = posture.WeightCritical
		hookPath := filepath.Join(projectRoot, ".claude/hooks/package-guard.py")
		if _, err := os.Stat(hookPath); err == nil {
			layer.Status = posture.LayerEnabled
			layer.Reason = "package-guard.py hook present"
		} else if enabledTools["attach-guard"] {
			layer.Status = posture.LayerPartial
			layer.Score = 5
			layer.Reason = "attach-guard enabled but package-guard.py not found"
		} else {
			layer.Status = posture.LayerDisabled
			layer.Reason = "no PreToolUse hooks configured"
		}

	case "age-gating":
		layer.Weight = posture.WeightHigh
		if enabledTools["attach-guard"] {
			layer.Status = posture.LayerEnabled
			layer.Reason = "age-gating active via attach-guard configuration"
		} else {
			layer.Status = posture.LayerDisabled
			layer.Reason = "age-gating not configured"
		}

	case "install-script-blocking":
		layer.Weight = posture.WeightHigh
		if enabledTools["attach-guard"] {
			layer.Status = posture.LayerEnabled
			layer.Reason = "install script blocking active via attach-guard"
		} else {
			layer.Status = posture.LayerDisabled
			layer.Reason = "install script blocking not configured"
		}

	case "lock-file-enforcement":
		layer.Weight = posture.WeightHigh
		preCommitPath := filepath.Join(projectRoot, ".pre-commit-config.yaml")
		if _, err := os.Stat(preCommitPath); err == nil {
			layer.Status = posture.LayerEnabled
			layer.Reason = "lock file enforcement configured via pre-commit"
		} else {
			layer.Status = posture.LayerDisabled
			layer.Reason = "no lock file enforcement configured"
		}

	case "vulnerability-scanning":
		layer.Weight = posture.WeightHigh
		if enabledTools["socket-dev-mcp"] || enabledTools["container-security"] {
			layer.Status = posture.LayerEnabled
			layer.Reason = "vulnerability scanning configured"
		} else {
			grypeConfig := filepath.Join(projectRoot, ".grype.yaml")
			if _, err := os.Stat(grypeConfig); err == nil {
				layer.Status = posture.LayerEnabled
				layer.Reason = "Grype vulnerability scanner configured"
			} else {
				layer.Status = posture.LayerDisabled
				layer.Reason = "no vulnerability scanning configured"
			}
		}

	case "nix-hardening":
		layer.Weight = posture.WeightMedium
		devenvPath := filepath.Join(projectRoot, "devenv.nix")
		if _, err := os.Stat(devenvPath); err == nil {
			layer.Status = posture.LayerEnabled
			layer.Reason = "devenv.nix present with hardening configuration"
		} else {
			layer.Status = posture.LayerDisabled
			layer.Reason = "devenv.nix not found"
		}

	case "sast":
		layer.Weight = posture.WeightMedium
		if enabledTools["semgrep"] {
			layer.Status = posture.LayerEnabled
			layer.Reason = "semgrep SAST configured"
		} else {
			layer.Status = posture.LayerDisabled
			layer.Reason = "no SAST tool configured"
		}

	case "secrets-scanning":
		layer.Weight = posture.WeightMedium
		hasGitleaks := enabledTools["gitleaks"]
		hasRipsecrets := enabledTools["ripsecrets"]
		if hasGitleaks && hasRipsecrets {
			layer.Status = posture.LayerEnabled
			layer.Score = 10
			layer.Reason = "both gitleaks and ripsecrets enabled"
		} else if hasGitleaks || hasRipsecrets {
			layer.Status = posture.LayerPartial
			layer.Score = 5
			layer.Reason = "partial secrets scanning coverage"
		} else {
			layer.Status = posture.LayerDisabled
			layer.Reason = "no secrets scanning configured"
		}

	case "container-security":
		layer.Weight = posture.WeightMedium
		dockerfile := filepath.Join(projectRoot, "Dockerfile")
		if _, err := os.Stat(dockerfile); err != nil {
			layer.Status = posture.LayerNotApplicable
			layer.Reason = "no Dockerfile detected"
		} else if enabledTools["container-security"] {
			layer.Status = posture.LayerEnabled
			layer.Reason = "container security scanning enabled"
		} else {
			layer.Status = posture.LayerDisabled
			layer.Reason = "Dockerfile present but no container security scanning"
		}

	case "license-compliance":
		layer.Weight = posture.WeightLow
		if enabledTools["license-compliance"] {
			layer.Status = posture.LayerEnabled
			layer.Reason = "license compliance scanning enabled"
		} else {
			layer.Status = posture.LayerDisabled
			layer.Reason = "license compliance not configured"
		}
	}

	return layer
}
