package selfupdate

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/instance"
)

// Command returns the "self-update" cobra command.
func Command() *cobra.Command {
	var (
		force   bool
		version string
	)

	cmd := &cobra.Command{
		Use:   "self-update",
		Short: "Update qsdev to the latest version",
		Long: `Check for and install the latest version of qsdev.

By default, checks GitHub for a newer release and, if found, downloads it,
verifies its checksum, and replaces the current binary. A backup is created
during the update and restored if anything goes wrong.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := DefaultConfig()
			currentVersion := instance.Version()

			ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
			defer cancel()

			var release *Release
			var err error

			if version != "" {
				// Fetch a specific version.
				tag := version
				if tag[0] != 'v' {
					tag = "v" + tag
				}
				release, err = FetchRelease(ctx, cfg, tag)
				if err != nil {
					return fmt.Errorf("fetching release %s: %w", version, err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Found release %s\n", release.Version)
			} else {
				// Check for latest.
				if !force {
					release, err = CheckForUpdate(ctx, cfg, currentVersion)
				} else {
					// Force: skip cache, just fetch latest.
					release, err = fetchLatestRelease(ctx, cfg)
				}
				if err != nil {
					return fmt.Errorf("checking for updates: %w", err)
				}
			}

			if release == nil {
				fmt.Fprintln(cmd.OutOrStdout(), "Already up to date.")
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Updating from %s to %s...\n",
				currentVersion, release.Version)

			if err := DoUpdate(ctx, cfg, release); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force update even if already up to date")
	cmd.Flags().StringVar(&version, "version", "", "Install a specific version (e.g. 1.2.3)")

	return cmd
}
