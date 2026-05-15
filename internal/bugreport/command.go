package bugreport

import (
	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/logging"
)

// Command returns the "report" cobra command tree.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate reports and file bug reports",
	}

	bug := &cobra.Command{
		Use:   "bug",
		Short: "File a bug report with diagnostic info and optional logs",
		Long: `Walk through an interactive wizard to file a bug report.

Collects environment info, lets you attach privacy-scrubbed log excerpts,
and submits via GitHub CLI, browser, or saves to file.

Logs are scrubbed for secrets, tokens, and credentials before inclusion.
No data is sent without your explicit approval.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot := logging.DetectProjectRoot()
			return RunWizard(projectRoot)
		},
	}

	cmd.AddCommand(bug)
	return cmd
}
