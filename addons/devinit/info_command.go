package devinit

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/info"
)

func infoCmd() *cobra.Command {
	var (
		oneline    bool
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show project status at a glance",
		Long: `Displays project name, ecosystems, tool count, security level, and version.
Instant response — reads cached state only, no evaluation.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInfo(cmd, oneline, jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&oneline, "oneline", false, "Single-line output for prompts and scripts")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output for machine consumption")

	return cmd
}

func runInfo(cmd *cobra.Command, oneline, jsonOutput bool) error {
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("determining project root: %w", err)
	}

	projectInfo, err := info.CollectInfo(projectRoot)
	if err != nil {
		if errors.Is(err, info.ErrNotGdevProject) {
			fmt.Fprintln(cmd.ErrOrStderr(), "Not a gdev-managed project. Run 'gdev init' to set up.")
			os.Exit(1)
		}
		return err
	}

	w := cmd.OutOrStdout()
	switch {
	case jsonOutput:
		return info.FormatJSON(projectInfo, w)
	case oneline:
		return info.FormatOneline(projectInfo, w)
	default:
		return info.FormatDefault(projectInfo, w)
	}
}
