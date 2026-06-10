package devinit

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/policy"
)

func sessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage security policy session bypass overrides",
	}
	cmd.AddCommand(sessionAllowCmd(), sessionClearCmd(), sessionListCmd())
	return cmd
}

func sessionAllowCmd() *cobra.Command {
	var rules []string

	cmd := &cobra.Command{
		Use:   "allow [rule-ids...]",
		Short: "Enable session bypass for specific policy rules",
		Long: `Add session-level bypass overrides for one or more policy rule IDs.
These overrides persist until explicitly cleared with 'session clear'.

Rule IDs can be passed as positional arguments or via the --rules flag.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			allRules := append(args, rules...)
			if len(allRules) == 0 {
				return fmt.Errorf("at least one rule ID is required")
			}
			return runSessionAllow(cmd, allRules)
		},
	}
	cmd.Flags().StringSliceVar(&rules, "rules", nil, "Comma-separated list of rule IDs")
	return cmd
}

func sessionClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Remove all session bypass overrides",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := sessionStatePath()
			if err != nil {
				return err
			}
			if err := policy.ClearSessionOverrides(path); err != nil {
				return fmt.Errorf("clearing session overrides: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Session bypass overrides cleared")
			return nil
		},
	}
}

func sessionListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show active session bypass overrides",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := sessionStatePath()
			if err != nil {
				return err
			}
			reader := policy.NewFileSessionStateReader(path)
			overrides := reader.SessionOverrides()
			if len(overrides) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No active session bypasses")
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Active session bypasses:")
			for _, id := range overrides {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", id)
			}
			return nil
		},
	}
}

func runSessionAllow(cmd *cobra.Command, ruleIDs []string) error {
	path, err := sessionStatePath()
	if err != nil {
		return err
	}

	reader := policy.NewFileSessionStateReader(path)
	existing := reader.SessionOverrides()

	merged := existing
	for _, id := range ruleIDs {
		if !slices.Contains(merged, id) {
			merged = append(merged, id)
		}
	}

	if err := policy.SaveSessionOverrides(path, merged); err != nil {
		return fmt.Errorf("saving session overrides: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Session bypass enabled for: %s\n", strings.Join(ruleIDs, ", "))
	return nil
}
