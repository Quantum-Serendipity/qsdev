package devinit

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/policyengine"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/policy"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/sarif"
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

func policyCheckCmd() *cobra.Command {
	var (
		sarifFlag  bool
		auditLevel string
		output     string
	)

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Assess the current security policy posture",
		Long: `Evaluate the loaded security policy and display a posture summary.

By default output is human-readable. Use --sarif to emit SARIF 2.1.0 format.
Exit code is 0 when posture is healthy, 1 when findings exceed the audit level.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPolicyCheck(cmd, sarifFlag, auditLevel, output)
		},
	}

	cmd.Flags().BoolVar(&sarifFlag, "sarif", false, "Output as SARIF 2.1.0")
	cmd.Flags().StringVar(&auditLevel, "audit-level", "any", "Minimum severity to fail: critical, high, medium, any")
	cmd.Flags().StringVar(&output, "output", "", "Write output to file instead of stdout")

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

func runPolicyCheck(cmd *cobra.Command, sarifFlag bool, auditLevel, outputPath string) error {
	engine, err := loadPolicyEngine()
	if err != nil {
		return err
	}

	orchestrator := policyengine.NewSecurityOrchestrator(engine, nil, nil)
	posture, _, _ := orchestrator.PostureSnapshot()

	if sarifFlag {
		return renderPolicySARIF(cmd, posture, outputPath)
	}

	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "Policy Posture Summary\n")
	fmt.Fprintf(w, "  Rules active:  %d / %d\n", posture.RulesActive, posture.RulesTotal)
	fmt.Fprintf(w, "  Monitor-only:  %d\n", posture.MonitorModeCount)

	if len(posture.BypassTierSummary) > 0 {
		fmt.Fprintf(w, "\n  Bypass tier distribution:\n")
		for tier, count := range posture.BypassTierSummary {
			fmt.Fprintf(w, "    %-16s %d\n", tier, count)
		}
	}

	if len(posture.CategoryCoverage) > 0 {
		fmt.Fprintf(w, "\n  Category coverage:\n")
		for cat := range posture.CategoryCoverage {
			fmt.Fprintf(w, "    %s\n", cat)
		}
	}

	if auditLevel != "none" && posture.RulesActive == 0 {
		return &ExitError{Code: 1}
	}

	return nil
}

func renderPolicySARIF(cmd *cobra.Command, posture *sarif.PolicyPosture, outputPath string) error {
	data, err := json.MarshalIndent(posture, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling SARIF output: %w", err)
	}

	if outputPath != "" {
		return writeOutputFile(outputPath, data)
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
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
