package selfupdate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/instance"
)

// Command returns the "self-update" cobra command.
func Command() *cobra.Command {
	var (
		force    bool
		version  string
		strict   bool
		noStrict bool
	)

	cmd := &cobra.Command{
		Use:    "self-update",
		Short:  "Update qsdev to the latest version",
		Hidden: true,
		Long: `Check for and install the latest version of qsdev.

By default, checks GitHub for a newer release and, if found, downloads it,
verifies its checksum and signature, test-runs it, and atomically replaces
the current binary. If anything fails, the current binary is left in place.

Prefer 'qsdev update' which coordinates binary updates with config regeneration.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := DefaultConfig()
			// Signature verification is required by default; --no-strict is the
			// escape hatch for dev/self-built binaries.
			cfg.Strict = strict && !noStrict
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
				if IsDowngrade(release.Version, currentVersion) {
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: v%s is OLDER than the running v%s; this is a downgrade and may reintroduce fixed bugs or vulnerabilities.\n",
						release.Version, strings.TrimPrefix(currentVersion, "v"))
				}
			} else {
				// Check for latest.
				if !force {
					release, err = CheckForUpdate(ctx, cfg, currentVersion)
				} else {
					// Force: skip cache, reinstall latest — but never silently
					// downgrade; --version is the explicit rollback path.
					release, err = ResolveForcedUpdate(ctx, cfg, currentVersion)
				}
				if err != nil {
					return fmt.Errorf("checking for updates: %w", err)
				}
			}

			if release == nil {
				fmt.Fprintln(cmd.OutOrStdout(), "Already up to date.")
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Updating from v%s to v%s...\n",
				strings.TrimPrefix(currentVersion, "v"), release.Version)

			if err := DoUpdate(ctx, cfg, release); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Reinstall the latest release even if already up to date (refuses to downgrade)")
	cmd.Flags().StringVar(&version, "version", "", "Install a specific version (e.g. 1.2.3); may downgrade")
	cmd.Flags().BoolVar(&strict, "strict", true, "Require a verified release signature before updating")
	cmd.Flags().BoolVar(&noStrict, "no-strict", false, "Allow updating without signature verification (escape hatch for dev/self-built binaries)")

	return cmd
}
