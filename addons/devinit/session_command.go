package devinit

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// sessionAllowOptions holds the flags of `session allow`.
type sessionAllowOptions struct {
	rules     []string
	sessionID string
	project   string
	ttl       time.Duration
}

func sessionAllowCmd() *cobra.Command {
	var opts sessionAllowOptions

	cmd := &cobra.Command{
		Use:   "allow [rule-ids...] --session <claude-session-id>",
		Short: "Bypass specific policy rules for one Claude Code session",
		Long: `Grant bypasses of one or more policy rule IDs to a single Claude Code
session in a single project. Grants are stored in ~/.qsdev/session-state.json.

A rule with bypass_tier "session" is lifted for that session until the grant
expires (default 8h). A rule with bypass_tier "command" gets a one-shot token:
the next tool call the rule matches in that session runs, and the token is
spent (an unused token expires after 1h by default). Use --ttl to shorten or
lengthen a grant, up to 24h. Rules with bypass_tier "enforce_always" cannot be
bypassed.

The session ID is printed in the policy's block message, together with the
command that lifts the rule. The project defaults to the qsdev project containing the current
directory.

Granting a bypass requires a human at an interactive terminal, outside any AI
agent session, who confirms the prompt; it cannot be scripted.

Rule IDs can be passed as positional arguments or via the --rules flag.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.rules = append(args, opts.rules...)
			if len(opts.rules) == 0 {
				return fmt.Errorf("at least one rule ID is required")
			}
			return runSessionAllow(cmd, opts)
		},
	}
	cmd.Flags().StringSliceVar(&opts.rules, "rules", nil, "Comma-separated list of rule IDs")
	cmd.Flags().StringVar(&opts.sessionID, "session", "", "Claude Code session ID the bypass applies to (required)")
	cmd.Flags().StringVar(&opts.project, "project", "", "Project directory the bypass applies to (default: the project containing the current directory)")
	cmd.Flags().DurationVar(&opts.ttl, "ttl", 0, "Grant lifetime (default 8h for session-tier rules, 1h for command-tier tokens; max 24h)")
	_ = cmd.MarkFlagRequired("session")
	return cmd
}

func sessionClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Remove all session bypass grants",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := sessionStatePath()
			if err != nil {
				return err
			}
			if err := policy.NewFileSessionStateStore(path).Clear(); err != nil {
				return fmt.Errorf("clearing session overrides: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Session bypass grants cleared")
			return nil
		},
	}
}

func sessionListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show active session bypass grants",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := sessionStatePath()
			if err != nil {
				return err
			}
			return writeSessionGrants(cmd.OutOrStdout(), policy.NewFileSessionStateStore(path), time.Now())
		},
	}
}

// writeSessionGrants lists the unexpired grants with their scope, and notes
// any pre-version-2 unscoped overrides, which no longer apply.
func writeSessionGrants(w io.Writer, store *policy.FileSessionStateStore, now time.Time) error {
	grants, legacy, err := store.Grants(now)
	if err != nil {
		return fmt.Errorf("reading session bypass grants: %w", err)
	}
	if len(legacy) > 0 {
		fmt.Fprintf(w, "Ignoring unscoped overrides from an older release (they no longer apply): %s\n",
			strings.Join(legacy, ", "))
	}
	if len(grants) == 0 {
		fmt.Fprintln(w, "No active session bypasses")
		return nil
	}
	fmt.Fprintln(w, "Active session bypasses:")
	for _, g := range grants {
		kind := "session"
		if g.Tier == policy.Command.String() {
			kind = "one-shot"
		}
		fmt.Fprintf(w, "  %s (%s) project %s, session %s, expires %s\n",
			g.RuleID, kind, g.ProjectRoot, g.SessionID, g.ExpiresAt.Local().Format(time.RFC3339))
	}
	return nil
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

func runSessionAllow(cmd *cobra.Command, opts sessionAllowOptions) error {
	// A bypass disables a security control, so it must come from a human: an
	// agent's tool calls run inside its session (and usually without a
	// terminal) and cannot answer the confirmation below. A pseudo-terminal
	// wrapper such as script(1) defeats the TTY check alone, hence the
	// agent-environment check too.
	if marker := agentSessionMarker(); marker != "" {
		return fmt.Errorf("'%s session allow' refused inside an AI agent session (%s is set): run it from your own terminal",
			branding.Get().AppName, marker)
	}
	if !sessionAllowInteractive(cmd.InOrStdin()) {
		return fmt.Errorf("'%s session allow' requires an interactive terminal: a policy bypass must be confirmed by a human",
			branding.Get().AppName)
	}
	if err := validateClaudeSessionID(opts.sessionID); err != nil {
		return err
	}

	projectRoot, err := sessionAllowProjectRoot(opts.project)
	if err != nil {
		return err
	}
	tiers, err := validateBypassableRules(opts.rules, policyFilesFor(projectRoot))
	if err != nil {
		return err
	}

	now := time.Now()
	scope := policy.BypassScope{ProjectRoot: projectRoot, SessionID: opts.sessionID}
	grants := make([]policy.BypassGrant, 0, len(opts.rules))
	for _, id := range opts.rules {
		g, err := policy.NewBypassGrant(id, tiers[id], scope, opts.ttl, now)
		if err != nil {
			return fmt.Errorf("cannot grant session bypass: %w", err)
		}
		grants = append(grants, g)
	}

	path, err := sessionStatePath()
	if err != nil {
		return err
	}

	confirmed, err := confirmSessionBypass(cmd, grants)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(cmd.OutOrStdout(), "Aborted; no bypass granted.")
		return nil
	}

	if err := policy.NewFileSessionStateStore(path).AddGrants(grants, now); err != nil {
		return fmt.Errorf("saving session overrides: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Session bypass enabled for: %s\n", strings.Join(opts.rules, ", "))
	return nil
}

// sessionAllowProjectRoot resolves the project a grant is bound to, in the
// canonical form the enforce hook keys grants by: the qsdev project containing
// --project when given, otherwise the one containing the current directory.
func sessionAllowProjectRoot(project string) (string, error) {
	if project == "" {
		return canonicalProjectRoot(policyProjectRoot("")), nil
	}
	abs, err := filepath.Abs(project)
	if err != nil {
		return "", fmt.Errorf("resolving --project %q: %w", project, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("resolving --project %q: %w", project, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("--project %q is not a directory", project)
	}
	return canonicalProjectRoot(projectRootFrom(abs)), nil
}

// maxSessionIDLen bounds a Claude Code session ID (a UUID in practice).
const maxSessionIDLen = 128

// validateClaudeSessionID rejects a session ID that is empty, overlong, or
// contains anything but letters, digits, '-', '_' and '.', so the stored grant
// and the confirmation prompt carry only a plain identifier.
func validateClaudeSessionID(id string) error {
	if id == "" {
		return fmt.Errorf("--session is required: pass the Claude Code session ID shown in the policy's block message")
	}
	if len(id) > maxSessionIDLen {
		return fmt.Errorf("--session %q is longer than %d characters", id, maxSessionIDLen)
	}
	for _, r := range id {
		if !isSessionIDRune(r) {
			return fmt.Errorf("--session %q contains an invalid character %q", id, r)
		}
	}
	return nil
}

func isSessionIDRune(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.'
}

// validateBypassableRules checks every requested ID names a loaded policy rule
// whose bypass tier permits a bypass, and returns each ID's tier. The
// evaluator ignores grants for enforce_always rules, so accepting them (or
// typos) would only mislead the user about what is bypassed.
func validateBypassableRules(ruleIDs, policyFiles []string) (map[string]policy.BypassTier, error) {
	if len(policyFiles) == 0 {
		return nil, fmt.Errorf("no policy files found (.qsdev/policy.yaml or ~/.qsdev/policy.yaml); there are no rules to bypass")
	}
	sp, err := policy.LoadPolicyFiles(policyFiles...)
	if err != nil {
		return nil, fmt.Errorf("loading policy to validate rule IDs: %w", err)
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
		return nil, fmt.Errorf("cannot grant session bypass:\n  %s", strings.Join(problems, "\n  "))
	}
	return tiers, nil
}

// confirmSessionBypass asks the user to confirm exactly what each grant lifts,
// where, and for how long. Only an explicit "y"/"yes" confirms.
func confirmSessionBypass(cmd *cobra.Command, grants []policy.BypassGrant) (bool, error) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "This lifts policy rules for Claude Code session %s in %s:\n", grants[0].SessionID, grants[0].ProjectRoot)
	for _, g := range grants {
		expires := g.ExpiresAt.Local().Format(time.RFC3339)
		if g.Tier == policy.Command.String() {
			fmt.Fprintf(out, "  %s: the next matching tool call only (token expires %s)\n", g.RuleID, expires)
		} else {
			fmt.Fprintf(out, "  %s: every call until %s\n", g.RuleID, expires)
		}
	}
	fmt.Fprint(out, "Continue? [y/N]: ")
	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("reading confirmation: %w", err)
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}
