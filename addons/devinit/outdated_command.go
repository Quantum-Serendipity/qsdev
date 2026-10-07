package devinit

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/outdated"
)

// onlineFlag opts outdated into running the ecosystem commands, which takes it
// out of the read-only contract.
const onlineFlag = "online"

func outdatedCmd() *cobra.Command {
	var opts outdated.OutdatedOptions

	cmd := &cobra.Command{
		Use:   "outdated",
		Short: "Check for outdated dependencies across all ecosystems",
		Long: `Runs each ecosystem's native outdated command and reports results.
Output is the native tool format — qsdev does not parse or normalize it.

These commands query package registries over the network, and some (mvn,
gradle, bundle, mix) download plugins or evaluate the project's build files.
Without --online nothing is run: the command each ecosystem would use is
printed and the command fails, so a CI step cannot pass without checking.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runOutdated(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Ecosystem, "ecosystem", "", "Check only a specific ecosystem (e.g., javascript, python, go)")
	cmd.Flags().BoolVar(&opts.Online, onlineFlag, false, "Run the ecosystem commands, which contact package registries")

	return cmdutil.MarkReadOnly(cmd, "", onlineFlag)
}

func runOutdated(cmd *cobra.Command, opts outdated.OutdatedOptions) error {
	pc, err := cmdutil.Project(cmd)
	if err != nil {
		return err
	}
	projectRoot := pc.Root

	// Determine detected ecosystems from answers.
	// A missing answers file is fine (falls back below); a corrupt or
	// unreadable one must be reported rather than silently treated as empty.
	answers, err := loadAnswersOrEmpty(projectRoot)
	if err != nil {
		return fmt.Errorf("loading saved answers: %w", err)
	}
	var ecosystems []string
	for _, lang := range answers.Languages {
		ecosystems = append(ecosystems, lang.Name)
		if lang.PackageManager != "" {
			if opts.PackageManagers == nil {
				opts.PackageManagers = make(map[string]string)
			}
			opts.PackageManagers[lang.Name] = lang.PackageManager
		}
	}

	// If no answers, fall back to checking which ecosystem binaries exist.
	if len(ecosystems) == 0 {
		ecosystems = outdated.SupportedEcosystems()
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := outdated.RunOutdated(ctx, cmd.OutOrStdout(), projectRoot, ecosystems, opts)
	if err != nil {
		return err
	}

	// A failed check leaves that ecosystem's status unknown, which must not read
	// as "up to date": report it distinctly from outdated packages.
	if failed := result.FailedEcosystems(); len(failed) > 0 {
		for _, check := range result.Ecosystems {
			if check.Error != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "%s: %v\n", check.Name, check.Error)
			}
		}
		return fmt.Errorf("outdated check failed for: %s", strings.Join(failed, ", "))
	}

	if planned := result.PlannedEcosystems(); len(planned) > 0 {
		return fmt.Errorf("%w (would check: %s)", outdated.ErrOnlineRequired, strings.Join(planned, ", "))
	}

	if result.HasAnyOutdated() {
		return fmt.Errorf("outdated packages found")
	}

	return nil
}
