package devinit

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/teardown"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
)

func teardownCmd() *cobra.Command {
	var (
		quick      bool
		compliance bool
		force      bool
		archive    bool
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   "teardown",
		Short: "Remove qsdev configuration from the current project",
		Long: `Remove qsdev-managed files and configuration from the project.

Three profiles control the scope of removal:

  Default:    Remove state directories and unmodified generated files.
              Shared files are surgically cleaned (qsdev sections removed).
              Modified files are preserved with warnings.

  --quick:    Remove state directories only (.devinit/).
              Leave all generated config files in place.

  --compliance: Generate a final posture report, archive all managed
              files, then perform the default teardown.

Use --force to skip interactive confirmation.
Use --archive to create a backup before removal.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTeardown(cmd, quick, compliance, force, archive, dryRun)
		},
	}

	cmd.Flags().BoolVar(&quick, "quick", false, "Remove state directories only")
	cmd.Flags().BoolVar(&compliance, "compliance", false, "Generate posture report and archive before teardown")
	cmd.Flags().BoolVar(&force, "force", false, "Skip interactive confirmation")
	cmd.Flags().BoolVar(&archive, "archive", false, "Create archive of managed files before removal")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview the teardown plan without executing")

	return cmd
}

func runTeardown(cmd *cobra.Command, quick, compliance, force, archive, dryRun bool) error {
	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return err
	}

	profile := teardown.ProfileDefault
	if quick {
		profile = teardown.ProfileQuick
	} else if compliance {
		profile = teardown.ProfileCompliance
	}

	opts := teardown.TeardownOptions{
		Profile:     profile,
		Force:       force || dryRun,
		Archive:     archive || compliance,
		DryRun:      dryRun,
		ProjectRoot: projectRoot,
	}

	registry := toolreg.DefaultRegistry()

	var confirm func(*teardown.TeardownPlan, io.Writer) bool
	if !force && !dryRun {
		confirm = func(_ *teardown.TeardownPlan, w io.Writer) bool {
			fmt.Fprint(w, "Proceed with teardown? [y/N] ")
			reader := bufio.NewReader(os.Stdin)
			line, _ := reader.ReadString('\n')
			return strings.TrimSpace(strings.ToLower(line)) == "y"
		}
	}

	result, err := teardown.Teardown(opts, registry, confirm, cmd.OutOrStdout())
	if err != nil {
		return err
	}

	if dryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "\n[dry-run] No files were modified.")
	}

	if len(result.Errors) > 0 {
		return fmt.Errorf("teardown completed with %d error(s)", len(result.Errors))
	}
	return nil
}
