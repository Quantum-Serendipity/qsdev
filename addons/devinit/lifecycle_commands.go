package devinit

import (
	"github.com/spf13/cobra"
)

// enableCmd creates the `gdev enable <tool>` command.
func enableCmd() *cobra.Command {
	var opts enableOptions

	cmd := &cobra.Command{
		Use:   "enable <tool>",
		Short: "Enable a tool in the current project",
		Long: `Enable a tool and generate its configuration files.

The tool's prerequisites are validated before enabling. Shared files (like
CLAUDE.md or settings.json) are surgically updated; exclusive files are
written fresh. Use 'gdev list' to see available tools.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEnable(cmd, args[0], opts)
		},
	}

	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Preview changes without writing files")

	return cmd
}

// disableCmd creates the `gdev disable <tool>` command.
func disableCmd() *cobra.Command {
	var opts disableOptions

	cmd := &cobra.Command{
		Use:   "disable <tool>",
		Short: "Disable a tool in the current project",
		Long: `Disable a tool and remove its configuration files.

Files exclusively owned by the tool are deleted. Sections contributed to
shared files are surgically removed. If any owned file has been modified by
the user, the command warns and exits unless --force is specified.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDisable(cmd, args[0], opts)
		},
	}

	cmd.Flags().BoolVar(&opts.Force, "force", false, "Remove files even if they have been modified by the user")

	return cmd
}

// listCmd creates the `gdev list` command.
func listCmd() *cobra.Command {
	var opts listOptions

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all available tools",
		Long: `List all registered tools grouped by category.

Use --category to filter by a specific category (security, ai-agent,
devex, infrastructure).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Category, "category", "", "Filter by category (security, ai-agent, devex, infrastructure)")

	return cmd
}

type enableOptions struct {
	DryRun bool
}

type disableOptions struct {
	Force bool
}

type listOptions struct {
	Category string
}
