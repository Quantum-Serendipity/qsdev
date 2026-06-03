package claudecode

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// HookStatus describes a single hook's deployment state for display.
type HookStatus struct {
	Name    string `json:"name"`
	Tier    string `json:"tier"`
	Event   string `json:"event"`
	Matcher string `json:"matcher"`
	Enabled bool   `json:"enabled"`
}

func hooksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage Claude Code hooks",
	}

	cmd.AddCommand(listHooksCmd())
	return cmd
}

func listHooksCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all registered hooks with deployment tier and status",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			answers, err := loadAnswers(projectRoot)
			if err != nil {
				answers = types.WizardAnswers{}
			}

			registry := defaultHookRegistry()
			statuses := buildHookStatuses(registry, answers)

			if jsonOutput {
				return writeHookStatusesJSON(cmd, statuses)
			}
			writeHookStatusesTable(cmd, statuses)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	return cmd
}

func buildHookStatuses(registry *HookRegistry, answers types.WizardAnswers) []HookStatus {
	var statuses []HookStatus
	for _, h := range registry.Definitions() {
		enabled := h.EnabledFunc == nil || h.EnabledFunc(answers)
		statuses = append(statuses, HookStatus{
			Name:    h.Owner,
			Tier:    h.Tier.String(),
			Event:   h.Event,
			Matcher: h.Matcher,
			Enabled: enabled,
		})
	}
	return statuses
}

func writeHookStatusesJSON(cmd *cobra.Command, statuses []HookStatus) error {
	data, err := json.MarshalIndent(statuses, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling hook statuses: %w", err)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

func writeHookStatusesTable(cmd *cobra.Command, statuses []HookStatus) {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "Hook\tTier\tEvent\tMatcher\tStatus")
	_, _ = fmt.Fprintln(w, "----\t----\t-----\t-------\t------")
	for _, s := range statuses {
		status := "disabled"
		if s.Enabled {
			status = "enabled"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", s.Name, s.Tier, s.Event, s.Matcher, status)
	}
	_ = w.Flush()
}
