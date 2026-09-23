package devinit

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/policy"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
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
Overrides are stored in ~/.qsdev/session-state.json, apply to every project
on this machine, and persist until explicitly cleared with 'session clear'.

Only rules loaded from the project or user policy with bypass_tier "session"
or "command" can be bypassed. Granting a bypass requires a human at an
interactive terminal, outside any AI agent session, who confirms the prompt;
it cannot be scripted.

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

// sessionAllowInteractive reports whether the command's input is an
// interactive terminal. A variable so tests can simulate a human at a TTY.
var sessionAllowInteractive = func(in io.Reader) bool {
	f, ok := in.(*os.File)
	return ok && term.IsTerminal(f.Fd())
}

// agentEnvMarkers are environment variables AI coding agents export to the
// shells they run commands in (Claude Code sets CLAUDECODE=1).
var agentEnvMarkers = []string{"CLAUDECODE"}

// agentSessionMarker returns the first agent marker set in the environment,
// or "" when the command is not running inside an agent session.
func agentSessionMarker() string {
	for _, name := range agentEnvMarkers {
		if os.Getenv(name) != "" {
			return name
		}
	}
	return ""
}

func runSessionAllow(cmd *cobra.Command, ruleIDs []string) error {
	// A bypass disables a security control for every project on the machine
	// until cleared, so it must come from a human: an agent's tool calls run
	// inside its session (and usually without a terminal) and cannot answer
	// the confirmation below. A pseudo-terminal wrapper such as script(1)
	// defeats the TTY check alone, hence the agent-environment check too.
	if marker := agentSessionMarker(); marker != "" {
		return fmt.Errorf("'%s session allow' refused inside an AI agent session (%s is set): run it from your own terminal",
			branding.Get().AppName, marker)
	}
	if !sessionAllowInteractive(cmd.InOrStdin()) {
		return fmt.Errorf("'%s session allow' requires an interactive terminal: a policy bypass must be confirmed by a human",
			branding.Get().AppName)
	}

	if err := validateBypassableRules(ruleIDs, discoverPolicyFiles()); err != nil {
		return err
	}

	path, err := sessionStatePath()
	if err != nil {
		return err
	}

	confirmed, err := confirmSessionBypass(cmd, ruleIDs, path)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(cmd.OutOrStdout(), "Aborted; no bypass granted.")
		return nil
	}

	reader := policy.NewFileSessionStateReader(path)
	merged := reader.SessionOverrides()
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

// validateBypassableRules checks every requested ID names a loaded policy rule
// whose bypass tier permits a session override. The evaluator ignores
// overrides for enforce_always rules, so accepting them (or typos) would only
// mislead the user about what is bypassed.
func validateBypassableRules(ruleIDs, policyFiles []string) error {
	if len(policyFiles) == 0 {
		return fmt.Errorf("no policy files found (.qsdev/policy.yaml or ~/.qsdev/policy.yaml); there are no rules to bypass")
	}
	sp, err := policy.LoadPolicyFiles(policyFiles...)
	if err != nil {
		return fmt.Errorf("loading policy to validate rule IDs: %w", err)
	}
	tiers := make(map[string]policy.BypassTier, len(sp.Rules))
	for _, r := range sp.Rules {
		tiers[r.ID] = r.BypassTier
	}

	var problems []string
	for _, id := range ruleIDs {
		tier, ok := tiers[id]
		switch {
		case !ok:
			problems = append(problems, fmt.Sprintf("%s: no such rule in the loaded policy", id))
		case tier != policy.Session && tier != policy.Command:
			problems = append(problems, fmt.Sprintf("%s: bypass_tier %s cannot be bypassed", id, tier))
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("cannot grant session bypass:\n  %s", strings.Join(problems, "\n  "))
	}
	return nil
}

// confirmSessionBypass asks the user to confirm the machine-wide, persistent
// scope of the bypass. Only an explicit "y"/"yes" confirms.
func confirmSessionBypass(cmd *cobra.Command, ruleIDs []string, statePath string) (bool, error) {
	fmt.Fprintf(cmd.OutOrStdout(),
		"This disables policy rule(s) %s for every project on this machine until '%s session clear' (stored in %s).\nContinue? [y/N]: ",
		strings.Join(ruleIDs, ", "), branding.Get().AppName, statePath)
	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("reading confirmation: %w", err)
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}
