package devinit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/policy"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/sarif"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

func policyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Security policy management and inspection",
	}
	cmd.AddCommand(policyCheckCmd(), policyListCmd(), policyShowCmd())
	return cmd
}

// policyAuditLevels are the accepted --audit-level values. The posture check
// has a single finding (no enforcing rules), which fails at every level except
// "none"; the severity names are accepted for parity with other gates.
var policyAuditLevels = []string{"critical", "high", "medium", "any", "none"}

// policyCheckOptions holds the flags of `policy check`.
type policyCheckOptions struct {
	sarif      bool
	auditLevel string
	output     string
}

func policyCheckCmd() *cobra.Command {
	var opts policyCheckOptions

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Assess the current security policy posture",
		Long: `Evaluate the loaded security policy and display a posture summary.

By default output is human-readable. Use --sarif to emit SARIF 2.1.0 format.
Use --output to write either format to a file.

Exit code is 1 when no rule is enforcing (every rule is disabled or
monitor-only), unless --audit-level is "none"; otherwise it is 0.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !slices.Contains(policyAuditLevels, opts.auditLevel) {
				return fmt.Errorf("invalid --audit-level %q; valid levels: %s",
					opts.auditLevel, strings.Join(policyAuditLevels, ", "))
			}
			return runPolicyCheck(cmd, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.sarif, "sarif", false, "Output as SARIF 2.1.0")
	cmd.Flags().StringVar(&opts.auditLevel, "audit-level", "any",
		"Minimum severity to fail: critical, high, medium, any; none never fails")
	cmd.Flags().StringVar(&opts.output, "output", "", "Write output to file instead of stdout")

	return cmd
}

func policyListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all rules in the loaded security policy",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPolicyList(cmd)
		},
	}
}

func policyShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <rule-id>",
		Short: "Show full details of a specific policy rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPolicyShow(cmd, args[0])
		},
	}
}

func runPolicyCheck(cmd *cobra.Command, opts policyCheckOptions) error {
	engine, err := loadPolicyEngine()
	if err != nil {
		return err
	}

	orchestrator := newProductionOrchestrator(engine, "", cmd.ErrOrStderr())
	posture, _, _ := orchestrator.PostureSnapshot()

	return evaluatePolicyPosture(cmd, posture, enforcingRuleCount(engine.CurrentRules()), opts)
}

// enforcingRuleCount counts the rules that can act on a tool call: enabled and
// not in monitor mode (a monitor-mode rule only records what it would block).
func enforcingRuleCount(rules []policy.CompiledRule) int {
	n := 0
	for _, r := range rules {
		if r.Rule.IsEnabled() && !r.Rule.MonitorMode {
			n++
		}
	}
	return n
}

// evaluatePolicyPosture renders the posture in the requested format and then
// applies the exit gate. Both the SARIF and human-readable render paths fall
// through to the SAME gate so that --sarif (the machine invocation used in CI)
// honors the identical exit-code contract as the text output: a policy with no
// enforcing rules (none loaded, all disabled, or all monitor-only) fails with
// exit code 1 unless the audit level is "none".
func evaluatePolicyPosture(cmd *cobra.Command, posture *sarif.PolicyPosture, enforcing int, opts policyCheckOptions) error {
	var data []byte
	if opts.sarif {
		var err error
		if data, err = renderPolicySARIF(posture, enforcing); err != nil {
			return err
		}
	} else {
		data = renderPolicyText(posture, enforcing)
	}

	if opts.output != "" {
		if err := writeOutputFile(opts.output, data); err != nil {
			return err
		}
	} else {
		_, _ = cmd.OutOrStdout().Write(data)
	}

	if opts.auditLevel != "none" && enforcing == 0 {
		return &ExitError{Code: 1}
	}

	return nil
}

func renderPolicyText(posture *sarif.PolicyPosture, enforcing int) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "Policy Posture Summary\n")
	fmt.Fprintf(&b, "  Rules active:  %d / %d\n", posture.RulesActive, posture.RulesTotal)
	fmt.Fprintf(&b, "  Monitor-only:  %d\n", posture.MonitorModeCount)
	fmt.Fprintf(&b, "  Enforcing:     %d\n", enforcing)

	if len(posture.BypassTierSummary) > 0 {
		fmt.Fprintf(&b, "\n  Bypass tier distribution:\n")
		for _, tier := range slices.Sorted(maps.Keys(posture.BypassTierSummary)) {
			fmt.Fprintf(&b, "    %-16s %d\n", tier, posture.BypassTierSummary[tier])
		}
	}

	if len(posture.CategoryCoverage) > 0 {
		fmt.Fprintf(&b, "\n  Category coverage:\n")
		for _, cat := range slices.Sorted(maps.Keys(posture.CategoryCoverage)) {
			fmt.Fprintf(&b, "    %s\n", cat)
		}
	}
	return b.Bytes()
}

func renderPolicySARIF(posture *sarif.PolicyPosture, enforcing int) ([]byte, error) {
	b := branding.Get()
	infoURI := fmt.Sprintf("https://github.com/%s/%s", b.GitHubOwner, b.GitHubRepo)

	log := sarif.BuildLog(b.AppName, "", infoURI, posturefindings(posture, enforcing))

	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling SARIF output: %w", err)
	}
	return append(data, '\n'), nil
}

// posturefindings converts the policy posture into SARIF results. A posture
// check evaluates no specific tool call, so it reports a warning-level result
// when no loaded rule is enforcing (an empty, fully disabled or monitor-only
// policy), which is the only posture condition that gates the exit code.
func posturefindings(posture *sarif.PolicyPosture, enforcing int) []sarif.SarifResult {
	if posture == nil || enforcing > 0 {
		return nil
	}
	return []sarif.SarifResult{{
		RuleID:           "qsdev/policy/MONITOR",
		Level:            "warning",
		Message:          "no enforcing security policy rules are loaded (all are disabled or monitor-only)",
		SecuritySeverity: 5.0,
		PartialFingerprints: map[string]string{
			"ruleId": "qsdev/policy/MONITOR",
		},
	}}
}

func runPolicyList(cmd *cobra.Command) error {
	engine, err := loadPolicyEngine()
	if err != nil {
		return err
	}

	rules := engine.CurrentRules()
	if len(rules) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No rules loaded")
		return nil
	}

	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "%-12s  %-30s  %-10s  %-16s  %-8s  %s\n",
		"ID", "Name", "Severity", "Bypass Tier", "Monitor", "Enabled")
	fmt.Fprintln(w, strings.Repeat("-", 100))

	for _, cr := range rules {
		r := cr.Rule
		monitor := "no"
		if r.MonitorMode {
			monitor = "yes"
		}
		enabled := "yes"
		if !r.IsEnabled() {
			enabled = "no"
		}
		fmt.Fprintf(w, "%-12s  %-30s  %-10s  %-16s  %-8s  %s\n",
			r.ID, r.Name, r.Severity, r.BypassTier, monitor, enabled)
	}

	return nil
}

func runPolicyShow(cmd *cobra.Command, ruleID string) error {
	engine, err := loadPolicyEngine()
	if err != nil {
		return err
	}

	for _, cr := range engine.CurrentRules() {
		r := cr.Rule
		if r.ID != ruleID {
			continue
		}

		w := cmd.OutOrStdout()
		fmt.Fprintf(w, "Rule: %s\n", r.ID)
		fmt.Fprintf(w, "  Name:         %s\n", r.Name)
		fmt.Fprintf(w, "  Category:     %s\n", r.Category)
		fmt.Fprintf(w, "  Severity:     %s\n", r.Severity)
		fmt.Fprintf(w, "  Bypass Tier:  %s\n", r.BypassTier)
		fmt.Fprintf(w, "  Monitor Mode: %v\n", r.MonitorMode)
		fmt.Fprintf(w, "  Enabled:      %v\n", r.IsEnabled())
		if r.Description != "" {
			fmt.Fprintf(w, "  Description:  %s\n", r.Description)
		}
		fmt.Fprintf(w, "  Action:       %s\n", r.Action.Type)
		if r.Action.Message != "" {
			fmt.Fprintf(w, "  Message:      %s\n", r.Action.Message)
		}
		return nil
	}

	return fmt.Errorf("rule %q not found", ruleID)
}

func loadPolicyEngine() (*policy.PolicyEngine, error) {
	policyFiles := discoverPolicyFiles()
	if len(policyFiles) == 0 {
		return nil, fmt.Errorf("no policy files found; create .qsdev/policy.yaml or ~/.qsdev/policy.yaml")
	}

	sessionPath, err := sessionStatePath()
	if err != nil {
		return nil, err
	}

	stateReader := policy.NewFileSessionStateReader(sessionPath)
	engine, err := policy.NewPolicyEngine(policyFiles, stateReader, policy.EngineOptions{})
	if err != nil {
		return nil, fmt.Errorf("loading policy engine: %w", err)
	}

	return engine, nil
}

func writeOutputFile(path string, data []byte) error {
	if err := fileutil.WriteFileAtomic(path, data, fileutil.ModeReadWrite); err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}
	return nil
}
