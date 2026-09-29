package devinit

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/internal/posture/conformance"
	"github.com/Quantum-Serendipity/qsdev/internal/posture/render"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

const (
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

When the project has a .qsdev-policy.yaml, its custom conformance
requirements are evaluated and reported under Conformance. A failing
requirement fails the audit gate at "high" and every stricter level.
Requirements on dependencies.totals need --scan; without it they fail.

Exit codes:
  0  All checks pass (or audit-level is "none")
  1  Findings above the audit threshold
  2  Project not initialized`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			level, err := normalizeStatusAuditLevel(auditLevel)
			if err != nil {
				return err
			}
			if format != "" && !slices.Contains(statusFormats, render.Format(format)) {
				return fmt.Errorf("invalid --format %q; valid formats: %s", format, joinFormats(statusFormats))
			}
			auditLevel = level
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
	cmd.Flags().StringVar(&format, "format", "", "Output format: text, json, sarif, badge")
	cmd.Flags().StringVar(&badgeType, "badge-type", "score", "Badge variant: score, conformance, defense")
	cmd.Flags().BoolVar(&allBadges, "all-badges", false, "Write all badge variants to --output-dir")
	cmd.Flags().StringVar(&outputDir, "output-dir", ".", "Directory for badge output files")
	cmd.Flags().BoolVar(&fix, "fix", false, "Output only remediation commands, one per line")
	cmd.Flags().BoolVar(&scan, "scan", false, "Force a fresh dependency scan before assessment")
	cmd.Flags().StringVar(&auditLevel, "audit-level", "high",
		"Exit threshold: none|info|low|moderate|high|critical (any = info, medium = moderate); each level includes every check of the levels above it")

	return cmd
}

// statusFormats are the accepted --format values.
var statusFormats = []render.Format{render.Text, render.JSON, render.SARIF, render.Badge}

func joinFormats(formats []render.Format) string {
	names := make([]string, len(formats))
	for i, f := range formats {
		names[i] = string(f)
	}
	return strings.Join(names, ", ")
}

// statusAuditLevels lists the gating audit levels from strictest to most
// permissive. A level fails when its own condition or that of any more
// permissive level fires, so tightening the level never loosens the gate.
var statusAuditLevels = []string{"info", "low", "moderate", "high", "critical"}

// statusAuditAliases maps alternative spellings to their canonical level;
// "medium" is the name `qsdev check` uses.
var statusAuditAliases = map[string]string{"any": "info", "medium": "moderate"}

// normalizeStatusAuditLevel validates an --audit-level value and returns its
// canonical name.
func normalizeStatusAuditLevel(level string) (string, error) {
	if canonical, ok := statusAuditAliases[level]; ok {
		return canonical, nil
	}
	if level == "none" || slices.Contains(statusAuditLevels, level) {
		return level, nil
	}
	return "", fmt.Errorf("invalid --audit-level %q; valid levels: none, %s (aliases: any, medium)",
		level, strings.Join(statusAuditLevels, ", "))
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
	projectDir, err := cmdutil.ProjectRoot()
	if err != nil {
		return err
	}

	// Perform assessment.
	report, err := posture.Assess(projectDir, posture.AssessOptions{FreshScan: opts.scan})
	if err != nil {
		if errors.Is(err, posture.ErrNotInitialized) {
			fmt.Fprintf(cmd.ErrOrStderr(), "Project not initialized. Run '%s init' first.\n", branding.Get().AppName)
			return &ExitError{Code: exitNotInitialized}
		}
		return fmt.Errorf("assessing project posture: %w", err)
	}
	// The project's own requirements (.qsdev-policy.yaml) are judged against
	// the finished report and gate the exit code alongside baseline conformance.
	conformance.Apply(projectDir, report)
	slog.Info("posture assessed",
		"score", report.Score.Total,
		"grade", report.Score.Grade,
		"tools", len(report.Tools))

	// Honest reporting: a vulnerability-gated audit level is meaningless without a
	// scan. Warn (on stderr, so machine-readable stdout stays clean) that the gate
	// cannot fire rather than letting the run pass as if the dependencies were
	// checked and found clean.
	if !opts.scan && vulnAuditLevelNeedsScan(opts.auditLevel) {
		fmt.Fprintf(cmd.ErrOrStderr(),
			"note: dependencies were not scanned for vulnerabilities; the vulnerability portion of --audit-level %s cannot fire — pass --scan to check.\n",
			opts.auditLevel)
	}

	// Detect color support.
	useColor := render.ColorSupported(os.Stdout.Fd())

	// Determine output format.
	outputFormat := resolveFormat(cmd, opts)

	// Handle --all-badges special case.
	if opts.allBadges {
		if err := render.RenderAllBadges(report, opts.outputDir); err != nil {
			return fmt.Errorf("writing badges: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Badges written to %s/\n", opts.outputDir)
		return exitForAudit(report, opts.auditLevel)
	}

	// Build render options.
	renderOpts := render.Options{
		Verbose:   opts.verbose,
		Quiet:     opts.quiet,
		Fix:       opts.fix,
		UseColor:  useColor,
		BadgeType: opts.badgeType,
	}
	if len(args) > 0 {
		renderOpts.Section = args[0]
	}

	// Render the report.
	w := cmd.OutOrStdout()
	if err := render.Report(report, outputFormat, w, renderOpts); err != nil {
		return fmt.Errorf("rendering report: %w", err)
	}

	return exitForAudit(report, opts.auditLevel)
}

// resolveFormat determines the output format based on flags and environment.
func resolveFormat(cmd *cobra.Command, opts postureStatusOptions) render.Format {
	// Explicit flags take priority.
	if opts.jsonFlag {
		return render.JSON
	}
	if opts.sarifFlag {
		return render.SARIF
	}
	if opts.format != "" {
		return render.Format(opts.format)
	}

	// CI detection: if CI=true and no explicit format flag was provided,
	// default to JSON for machine consumption.
	if os.Getenv("CI") == "true" {
		// Only auto-switch if no format-related flag was explicitly set.
		jsonChanged := cmd.Flags().Changed("json")
		sarifChanged := cmd.Flags().Changed("sarif")
		formatChanged := cmd.Flags().Changed("format")
		if !jsonChanged && !sarifChanged && !formatChanged {
			return render.JSON
		}
	}

	return render.Text
}

// vulnAuditLevelNeedsScan reports whether the given audit level's gate can be
// influenced by dependency vulnerability counts. Without --scan those counts
// stay zero, so the gate's vulnerability portion can never fire and a zero
// count must not read as clean — the command surfaces an honest note. Every
// level except "none" (which never gates on anything) qualifies; this
// deliberately includes the DEFAULT level "high" and "info", which an earlier
// enumeration omitted.
func vulnAuditLevelNeedsScan(level string) bool {
	return level != "none"
}

// exitForAudit evaluates the (canonical) audit level and returns an error that
// wraps the appropriate exit code if findings exceed the threshold. The gate is
// cumulative: a level also fails on every condition of the more permissive
// levels, so e.g. "low" never passes a baseline-conformance failure that the
// default "high" catches.
func exitForAudit(report *posture.PostureReport, auditLevel string) error {
	idx := slices.Index(statusAuditLevels, auditLevel)
	if idx < 0 {
		// "none" (or an unvalidated value) is judged on its own terms.
		if posture.ShouldExitNonZero(report, auditLevel) {
			return &ExitError{Code: exitFindings}
		}
		return nil
	}
	for _, level := range statusAuditLevels[idx:] {
		if posture.ShouldExitNonZero(report, level) {
			return &ExitError{Code: exitFindings}
		}
	}
	return nil
}
