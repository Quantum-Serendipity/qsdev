package devinit

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/evidence"
	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

func evidenceCmd() *cobra.Command {
	var (
		framework      string
		format         string
		listFrameworks bool
	)

	cmd := &cobra.Command{
		Use:   "evidence",
		Short: "Generate compliance evidence report mapping qsdev controls to a framework",
		Long: `Generate a compliance evidence report that maps qsdev's defense-in-depth
layers to controls in a compliance framework (SOC2, HIPAA, ASVS, etc.).

The report shows which framework controls are addressed, partially addressed,
or not addressed by the current qsdev configuration. Output is available in
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

	// Assess the project's real security posture using the SAME assessor that
	// `qsdev status` uses (posture.Assess). Deriving evidence from the live
	// posture assessment — rather than from mere config-file presence — ensures
	// a control is only reported as "Addressed" when its defense layer is
	// actually enforced, and keeps the evidence report consistent with
	// `qsdev status` (no evidence-vs-status divergence).
	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return err
	}
	projectName := filepath.Base(projectRoot)

	report, err := posture.Assess(projectRoot, posture.AssessOptions{})
	if err != nil {
		if errors.Is(err, posture.ErrNotInitialized) {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
				"Project not initialized. Run '%s init' first.\n", branding.Get().AppName)
			return &ExitError{Code: exitNotInitialized}
		}
		return fmt.Errorf("assessing project posture: %w", err)
	}

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
