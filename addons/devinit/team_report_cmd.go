package devinit

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/internal/teamreport"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

func teamReportCmd() *cobra.Command {
	var (
		inputDir         string
		scopeFile        string
		format           string
		threshold        float64
		trend            bool
		createIssues     bool
		historyFile      string
		generateWorkflow bool
		output           string
	)

	cmd := &cobra.Command{
		Use:   "team-report",
		Short: "Aggregate security posture across multiple projects",
		Long: `Generate a team-level security posture dashboard by aggregating
individual project posture reports.

Reports can be loaded from a directory (--input-dir) or collected from
GitHub repositories defined in a scope file (--scope).

Output formats:
  md    Markdown dashboard (default)
  json  Machine-readable JSON

Use --generate-workflow to emit a GitHub Actions workflow template
for automated team posture aggregation.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if generateWorkflow {
				return runGenerateWorkflow(cmd, output)
			}
			return runTeamReport(cmd, teamReportOptions{
				inputDir:     inputDir,
				scopeFile:    scopeFile,
				format:       format,
				threshold:    threshold,
				trend:        trend,
				createIssues: createIssues,
				historyFile:  historyFile,
				output:       output,
			})
		},
	}

	cmd.Flags().StringVar(&inputDir, "input-dir", "", "Directory containing posture report JSON files")
	cmd.Flags().StringVar(&scopeFile, "scope", "", "Path to scope file defining repositories to include")
	cmd.Flags().StringVar(&format, "format", "md", "Output format: md, json")
	cmd.Flags().Float64Var(&threshold, "threshold", 70, "Score threshold for alerts")
	cmd.Flags().BoolVar(&trend, "trend", false, "Include trend data from history")
	cmd.Flags().BoolVar(&createIssues, "create-issues", false, "Create GitHub issues for degraded projects")
	cmd.Flags().StringVar(&historyFile, "history-file", "team-posture-history.json", "Path to history file for trend tracking")
	cmd.Flags().BoolVar(&generateWorkflow, "generate-workflow", false, "Generate GitHub Actions workflow template")
	cmd.Flags().StringVar(&output, "output", "", "Output file path (default: stdout)")

	return cmd
}

type teamReportOptions struct {
	inputDir     string
	scopeFile    string
	format       string
	threshold    float64
	trend        bool
	createIssues bool
	historyFile  string
	output       string
}

func runTeamReport(cmd *cobra.Command, opts teamReportOptions) error {
	// Validate format.
	switch opts.format {
	case "md", "json":
		// valid
	default:
		return fmt.Errorf("unsupported format %q; use md or json", opts.format)
	}

	// Load reports from input source.
	var reports []*posture.PostureReport
	var warnings []string

	switch {
	case opts.inputDir != "":
		var err error
		reports, warnings, err = teamreport.LoadPostureReports(opts.inputDir)
		if err != nil {
			return fmt.Errorf("loading posture reports: %w", err)
		}
	case opts.scopeFile != "":
		var err error
		reports, warnings, err = teamreport.CollectFromScope(cmdContext(cmd), opts.scopeFile)
		if err != nil {
			return fmt.Errorf("collecting from scope: %w", err)
		}
	default:
		return fmt.Errorf("either --input-dir or --scope is required")
	}

	// Print warnings.
	for _, w := range warnings {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", w)
	}

	if len(reports) == 0 {
		return fmt.Errorf("no valid posture reports found")
	}

	// Aggregate.
	aggOpts := teamreport.AggregateOptions{
		Threshold:     opts.threshold,
		IncludeTrends: opts.trend,
		QsdevVersion:  version.Info().Version,
	}
	if opts.trend {
		aggOpts.HistoryFile = opts.historyFile
	}

	teamReport, err := teamreport.Aggregate(reports, aggOpts)
	if err != nil {
		return fmt.Errorf("aggregating reports: %w", err)
	}

	// Create issues if requested. An issue that could not be filed is reported
	// as an error once the report itself has been written.
	var issuesErr error
	if opts.createIssues {
		var history *teamreport.HistoryStore
		if opts.historyFile != "" {
			history, err = teamreport.LoadHistory(opts.historyFile)
			if err != nil {
				return fmt.Errorf("loading history for issue generation: %w", err)
			}
		}

		issues := teamreport.GenerateIssues(teamReport, history)
		issuesErr = createTeamIssues(cmd.ErrOrStderr(), issues, func(routable []teamreport.IssueSpec) (int, error) {
			return teamreport.CreateIssuesViaCLI(cmdContext(cmd), routable)
		})
	}

	// Render output.
	var rendered []byte
	switch opts.format {
	case "md":
		rendered = []byte(teamreport.RenderMarkdown(teamReport))
	case "json":
		rendered, err = teamreport.RenderJSON(teamReport)
		if err != nil {
			return fmt.Errorf("rendering JSON: %w", err)
		}
	}

	// Write output.
	if opts.output != "" {
		if err := fileutil.WriteFileAtomic(opts.output, rendered, fileutil.ModeReadWrite); err != nil {
			return fmt.Errorf("writing output to %s: %w", opts.output, err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Report written to %s\n", opts.output)
	} else {
		_, _ = cmd.OutOrStdout().Write(rendered)
	}

	return issuesErr
}

// createTeamIssues files the issues that name a repository through create and
// reports how many were actually filed, even when some failed. Issues without
// a repository cannot be filed; they make the command fail instead of being
// counted as created, so a degraded project's alert is never silently lost.
func createTeamIssues(w io.Writer, issues []teamreport.IssueSpec, create func([]teamreport.IssueSpec) (int, error)) error {
	if len(issues) == 0 {
		fmt.Fprintln(w, "No issues to create")
		return nil
	}

	var routable []teamreport.IssueSpec
	var unrouted []string
	for _, issue := range issues {
		if issue.Repo == "" {
			unrouted = append(unrouted, issue.Title)
			continue
		}
		routable = append(routable, issue)
	}

	created := 0
	var createErr error
	if len(routable) > 0 {
		created, createErr = create(routable)
	}
	fmt.Fprintf(w, "Created %d of %d issue(s)\n", created, len(issues))
	if createErr != nil {
		return fmt.Errorf("creating issues: %w", createErr)
	}

	if len(unrouted) > 0 {
		return fmt.Errorf("%d issue(s) not created because the project's repository is unknown: %s",
			len(unrouted), strings.Join(unrouted, "; "))
	}
	return nil
}

// cmdContext returns the command's context, falling back to Background when
// the command runs without one (e.g. RunE invoked directly).
func cmdContext(cmd *cobra.Command) context.Context {
	if ctx := cmd.Context(); ctx != nil {
		return ctx
	}
	return context.Background()
}

func runGenerateWorkflow(cmd *cobra.Command, output string) error {
	workflow := teamreport.GenerateTeamWorkflow()
	perProject := teamreport.GeneratePerProjectSteps()

	content := workflow + "\n---\n\n# Per-project steps (add to each project's CI workflow):\n\n" + perProject

	if output != "" {
		if err := fileutil.WriteFileAtomic(output, []byte(content), fileutil.ModeReadWrite); err != nil {
			return fmt.Errorf("writing workflow to %s: %w", output, err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Workflow written to %s\n", output)
	} else {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), content)
	}

	return nil
}
