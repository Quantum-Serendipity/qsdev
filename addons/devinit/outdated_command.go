package devinit

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/outdated"
)

func outdatedCmd() *cobra.Command {
	var opts outdated.OutdatedOptions

	cmd := &cobra.Command{
		Use:   "outdated",
		Short: "Check for outdated dependencies across all ecosystems",
		Long: `Runs each ecosystem's native outdated command and reports results.
Output is the native tool format — qsdev does not parse or normalize it.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runOutdated(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Ecosystem, "ecosystem", "", "Check only a specific ecosystem (e.g., javascript, python, go)")

	return cmd
}

func runOutdated(cmd *cobra.Command, opts outdated.OutdatedOptions) error {
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("determining project root: %w", err)
	}

	// Determine detected ecosystems from answers.
	answers, _ := loadAnswersOrEmpty(projectRoot)
	var ecosystems []string
	for _, lang := range answers.Languages {
		ecosystems = append(ecosystems, lang.Name)
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

	if result.HasAnyOutdated() {
		return fmt.Errorf("outdated packages found")
	}

	return nil
}
