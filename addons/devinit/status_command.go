package devinit

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
)

const (
	exitOK             = 0
	exitFindings       = 1
	exitNotInitialized = 2
)

// statusCmd creates the `qsdev status` command with full posture assessment,
// machine-readable output support, and CI-aware defaults.
func statusCmd() *cobra.Command {
	var (
		verbose    bool
		quiet      bool
		jsonFlag   bool
		sarifFlag  bool
		format     string
		badgeType  string
		allBadges  bool
		outputDir  string
		fix        bool
		scan       bool
		auditLevel string
	)

	cmd := &cobra.Command{
		Use:   "status [section]",
		Short: "Show the security posture of the current project",
		Long: `Assess and display the security posture of the current project.

By default, output is a human-readable summary. Use --json or --sarif for
machine-readable output. In CI environments (CI=true), JSON output is the
default unless an explicit format flag is provided.

Optional positional argument to show a specific section:
  defense   Show defense layer details
  config    Show configuration health
  deps      Show dependency health
  tools     Show tool availability

Exit codes:
  0  All checks pass (or audit-level is "none")
  1  Findings above the audit threshold
  2  Project not initialized`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPostureStatus(cmd, args, postureStatusOptions{
				verbose:    verbose,
				quiet:      quiet,
				jsonFlag:   jsonFlag,
				sarifFlag:  sarifFlag,
				format:     format,
				badgeType:  badgeType,
				allBadges:  allBadges,
				outputDir:  outputDir,
				fix:        fix,
				scan:       scan,
				auditLevel: auditLevel,
			})
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show expanded per-layer detail and remediation hints")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Single-line output: score and grade only")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	cmd.Flags().BoolVar(&sarifFlag, "sarif", false, "Output as SARIF 2.1.0")
	cmd.Flags().StringVar(&format, "format", "", "Output format: badge")
	cmd.Flags().StringVar(&badgeType, "badge-type", "score", "Badge variant: score, conformance, defense")
	cmd.Flags().BoolVar(&allBadges, "all-badges", false, "Write all badge variants to --output-dir")
	cmd.Flags().StringVar(&outputDir, "output-dir", ".", "Directory for badge output files")
	cmd.Flags().BoolVar(&fix, "fix", false, "Output only remediation commands, one per line")
	cmd.Flags().BoolVar(&scan, "scan", false, "Force a fresh dependency scan before assessment")
	cmd.Flags().StringVar(&auditLevel, "audit-level", "high", "Exit threshold: none|info|low|moderate|high|critical")

	return cmd
}

type postureStatusOptions struct {
	verbose    bool
	quiet      bool
	jsonFlag   bool
	sarifFlag  bool
	format     string
	badgeType  string
	allBadges  bool
	outputDir  string
	fix        bool
	scan       bool
	auditLevel string
}

func runPostureStatus(cmd *cobra.Command, args []string, opts postureStatusOptions) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("determining project directory: %w", err)
	}

	// Perform assessment.
	report, err := posture.Assess(projectDir, posture.AssessOptions{
		FreshScan:  opts.scan,
		AuditLevel: opts.auditLevel,
	})
	if err != nil {
		if errors.Is(err, posture.ErrNotInitialized) {
			fmt.Fprintln(cmd.ErrOrStderr(), "Project not initialized. Run 'qsdev init' first.")
			return &ExitError{Code: exitNotInitialized}
		}
		return fmt.Errorf("assessing project posture: %w", err)
	}

	// Detect color support.
	useColor := posture.ColorSupported(os.Stdout.Fd())

	// Determine output format.
	outputFormat := resolveFormat(cmd, opts)

	// Handle --all-badges special case.
	if opts.allBadges {
		if err := posture.RenderAllBadges(report, opts.outputDir); err != nil {
			return fmt.Errorf("writing badges: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Badges written to %s/\n", opts.outputDir)
		return exitForAudit(report, opts.auditLevel)
	}

	// Build render options.
	renderOpts := posture.RenderOptions{
		Verbose:   opts.verbose,
		Quiet:     opts.quiet,
		Fix:       opts.fix,
		UseColor:  useColor,
		BadgeType: opts.badgeType,
	}

	// Render the report.
	w := cmd.OutOrStdout()
	if err := posture.RenderReport(report, outputFormat, w, renderOpts); err != nil {
		return fmt.Errorf("rendering report: %w", err)
	}

	return exitForAudit(report, opts.auditLevel)
}

// resolveFormat determines the output format based on flags and environment.
func resolveFormat(cmd *cobra.Command, opts postureStatusOptions) posture.OutputFormat {
	// Explicit flags take priority.
	if opts.jsonFlag {
		return posture.FormatJSON
	}
	if opts.sarifFlag {
		return posture.FormatSARIF
	}
	if opts.format == "badge" {
		return posture.FormatBadge
	}

	// CI detection: if CI=true and no explicit format flag was provided,
	// default to JSON for machine consumption.
	if os.Getenv("CI") == "true" {
		// Only auto-switch if no format-related flag was explicitly set.
		jsonChanged := cmd.Flags().Changed("json")
		sarifChanged := cmd.Flags().Changed("sarif")
		formatChanged := cmd.Flags().Changed("format")
		if !jsonChanged && !sarifChanged && !formatChanged {
			return posture.FormatJSON
		}
	}

	return posture.FormatText
}

// exitForAudit evaluates the audit level and returns an error that wraps the
// appropriate exit code if findings exceed the threshold.
func exitForAudit(report *posture.PostureReport, auditLevel string) error {
	if posture.ShouldExitNonZero(report, auditLevel) {
		return &ExitError{Code: exitFindings}
	}
	return nil
}
