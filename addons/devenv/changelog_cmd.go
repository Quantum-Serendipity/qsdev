package devenv

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// changelogCmd returns a cobra command that wraps git-cliff for changelog generation.
func changelogCmd() *cobra.Command {
	var (
		output     string
		latest     bool
		unreleased bool
		tag        string
	)

	cmd := &cobra.Command{
		Use:   "changelog",
		Short: "Generate a changelog using git-cliff",
		Long:  "Thin wrapper around git-cliff that generates a changelog from conventional commits.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cliffPath, err := exec.LookPath("git-cliff")
			if err != nil {
				return fmt.Errorf("git-cliff not found in PATH: install it via 'cargo install git-cliff' or add it to devenv.nix packages")
			}

			args := []string{"--output", output}
			if latest {
				args = append(args, "--latest")
			}
			if unreleased {
				args = append(args, "--unreleased")
			}
			if tag != "" {
				args = append(args, "--tag", tag)
			}

			proc := exec.Command(cliffPath, args...)
			proc.Stdout = cmd.OutOrStdout()
			proc.Stderr = cmd.ErrOrStderr()
			proc.Stdin = os.Stdin

			if err := proc.Run(); err != nil {
				return fmt.Errorf("git-cliff failed: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "CHANGELOG.md", "Output file path")
	cmd.Flags().BoolVar(&latest, "latest", false, "Only process the latest tag")
	cmd.Flags().BoolVar(&unreleased, "unreleased", false, "Only process unreleased commits")
	cmd.Flags().StringVar(&tag, "tag", "", "Set the tag for unreleased commits")

	return cmd
}
