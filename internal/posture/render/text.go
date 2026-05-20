package render

import (
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
)

// RenderText renders a PostureReport as human-readable text.
// Behavior varies based on Options: quiet, verbose, and fix modes.
func RenderText(report *posture.PostureReport, w io.Writer, opts Options) error {
	if opts.Fix {
		return renderFix(report, w)
	}
	if opts.Quiet {
		return renderQuiet(report, w)
	}
	if opts.Verbose {
		return renderVerbose(report, w, opts)
	}
	return renderDefault(report, w, opts)
}

// renderQuiet outputs a single compact line: "82/100 B+\n"
func renderQuiet(report *posture.PostureReport, w io.Writer) error {
	score := int(math.Round(report.Score.Total))
	_, err := fmt.Fprintf(w, "%d/100 %s\n", score, report.Score.Grade)
	return err
}

// sectionMatch returns true if the given section name matches the filter, or if
// the filter is empty (show all).
func sectionMatch(filter, name string) bool {
	return filter == "" || strings.EqualFold(filter, name)
}

// renderDefault outputs the standard multi-section text report.
func renderDefault(report *posture.PostureReport, w io.Writer, opts Options) error {
	pass, partial, skip, fail := Indicators(opts.UseColor)
	sec := strings.ToLower(opts.Section)

	// Header (always shown)
	fmt.Fprintf(w, "Security Posture: %s (%s)\n", report.ProjectName, report.ProjectPath)
	fmt.Fprintf(w, "Score: %d/100 (%s)\n", int(math.Round(report.Score.Total)), report.Score.Grade)
	if report.Tier.Current != "" {
		tierLine := fmt.Sprintf("Tier: %s (%d/%d)", report.Tier.Current, report.Tier.Position, report.Tier.Total)
		if report.Tier.NextTier != "" {
			tierLine += fmt.Sprintf(" | Next: qsdev init --tier %s --dry-run", report.Tier.NextTier)
		}
		fmt.Fprintln(w, tierLine)
	}
	fmt.Fprintln(w)

	// Conformance (shown when no section filter or "conformance")
	if sec == "" || sec == "conformance" {
		baselineStatus := pass
		if !report.Conformance.Baseline.Pass {
			baselineStatus = fail
		}
		enhancedStatus := pass
		if !report.Conformance.Enhanced.Pass {
			enhancedStatus = fail
		}
		fmt.Fprintf(w, "Conformance: %s Baseline  %s Enhanced\n", baselineStatus, enhancedStatus)
		fmt.Fprintln(w)
	}

	// Defense Coverage
	if sectionMatch(sec, "defense") {
		fmt.Fprintf(w, "Defense Coverage: %d/%d layers (%.0f%%)\n",
			report.Defense.Enabled, report.Defense.Total, report.Defense.Score)
		for _, l := range report.Defense.Layers {
			ind := indicatorForLayer(l.Status, pass, partial, skip, fail)
			fmt.Fprintf(w, "  %s %-30s %s\n", ind, l.Name, l.Status)
		}
		fmt.Fprintln(w)
	}

	// Config Health
	if sectionMatch(sec, "config") {
		fmt.Fprintf(w, "Config Health: %.0f%% (%d/%d files current)\n",
			report.Config.Score, report.Config.Current, report.Config.Total)
		if report.Config.Modified > 0 {
			fmt.Fprintf(w, "  Modified: %d\n", report.Config.Modified)
		}
		if report.Config.Outdated > 0 {
			fmt.Fprintf(w, "  Outdated: %d\n", report.Config.Outdated)
		}
		if report.Config.Missing > 0 {
			fmt.Fprintf(w, "  Missing:  %d\n", report.Config.Missing)
		}
		fmt.Fprintln(w)
	}

	// Dependency Health
	if sectionMatch(sec, "deps") || sectionMatch(sec, "dependencies") {
		totals := report.Dependencies.Totals
		fmt.Fprintf(w, "Dependency Health: %.0f%%\n", report.Dependencies.Score)
		if totals.Total() > 0 {
			fmt.Fprintf(w, "  Vulnerabilities: %d critical, %d high, %d moderate, %d low\n",
				totals.Critical, totals.High, totals.Moderate, totals.Low)
		} else {
			fmt.Fprintf(w, "  No vulnerabilities detected\n")
		}
		if report.Dependencies.Stale {
			fmt.Fprintf(w, "  %s Scan data may be stale; re-run with --scan\n", fail)
		}
		fmt.Fprintln(w)
	}

	// Drift summary
	if sec == "" && report.Drift.TotalFindings > 0 {
		fmt.Fprintf(w, "Drift: %d finding(s)\n", report.Drift.TotalFindings)
		for sev, count := range report.Drift.BySeverity {
			fmt.Fprintf(w, "  %s: %d\n", sev, count)
		}
		fmt.Fprintln(w)
	}

	// Tools (shown when "tools" section requested)
	if sectionMatch(sec, "tools") && sec != "" && len(report.Tools) > 0 {
		fmt.Fprintln(w, "Tools:")
		for _, t := range report.Tools {
			ind := pass
			if !t.Enabled {
				ind = skip
			} else if !t.Available {
				ind = fail
			}
			fmt.Fprintf(w, "  %s %-25s enabled=%v available=%v\n",
				ind, t.Name, t.Enabled, t.Available)
		}
		fmt.Fprintln(w)
	}

	// Footer hint (only when showing full report)
	if sec == "" {
		fmt.Fprintf(w, "Run 'qsdev status --verbose' for details or 'qsdev status --fix' for remediation commands.\n")
		if report.Tier.Current != "" && report.Tier.NextTier != "" {
			fmt.Fprintf(w, "Upgrade tier: qsdev init --tier %s --dry-run\n", report.Tier.NextTier)
		}
	}
	return nil
}

// renderVerbose outputs the expanded report with per-layer detail and
// remediation hints.
func renderVerbose(report *posture.PostureReport, w io.Writer, opts Options) error {
	pass, partial, skip, fail := Indicators(opts.UseColor)

	// Header
	fmt.Fprintf(w, "Security Posture: %s (%s)\n", report.ProjectName, report.ProjectPath)
	fmt.Fprintf(w, "Score: %d/100 (%s)  Defense: %.0f%%  Config: %.0f%%  Deps: %.0f%%\n",
		int(math.Round(report.Score.Total)), report.Score.Grade,
		report.Score.Defense, report.Score.Config, report.Score.DepHealth)
	fmt.Fprintf(w, "Schema: %s  Generated: %s  qsdev: %s\n",
		report.SchemaVersion, report.GeneratedAt.Format("2006-01-02 15:04:05 UTC"), report.QsdevVersion)
	if report.Tier.Current != "" {
		fmt.Fprintf(w, "Tier: %s (%d/%d)\n", report.Tier.Current, report.Tier.Position, report.Tier.Total)
		if report.Tier.NextTier != "" {
			desc := posture.TierDescription(report.Tier.NextTier)
			if desc != "" {
				fmt.Fprintf(w, "  Next tier: %s — %s\n", report.Tier.NextTier, desc)
			} else {
				fmt.Fprintf(w, "  Next tier: %s\n", report.Tier.NextTier)
			}
			fmt.Fprintf(w, "  Upgrade: qsdev init --tier %s --dry-run\n", report.Tier.NextTier)
		}
	}
	fmt.Fprintln(w)

	// Conformance (detailed)
	fmt.Fprintln(w, "Conformance:")
	renderConformanceLevel(w, "Baseline", report.Conformance.Baseline, pass, fail)
	renderConformanceLevel(w, "Enhanced", report.Conformance.Enhanced, pass, fail)
	if report.Conformance.Custom != nil {
		renderConformanceLevel(w, "Custom", *report.Conformance.Custom, pass, fail)
	}
	fmt.Fprintln(w)

	// Defense Coverage (detailed)
	fmt.Fprintf(w, "Defense Coverage: %d/%d layers (%.0f%%)\n",
		report.Defense.Enabled, report.Defense.Total, report.Defense.Score)
	for _, l := range report.Defense.Layers {
		ind := indicatorForLayer(l.Status, pass, partial, skip, fail)
		fmt.Fprintf(w, "  %s %-30s %-15s weight=%s", ind, l.Name, l.Status, l.Weight)
		if l.Status == posture.LayerPartial {
			fmt.Fprintf(w, " score=%d/10", l.Score)
		}
		fmt.Fprintln(w)
		if l.Reason != "" {
			fmt.Fprintf(w, "      Reason: %s\n", l.Reason)
		}
		if l.Details != "" {
			fmt.Fprintf(w, "      Details: %s\n", l.Details)
		}
	}
	fmt.Fprintln(w)

	// Config Health (detailed)
	fmt.Fprintf(w, "Config Health: %.0f%%\n", report.Config.Score)
	for _, f := range report.Config.Files {
		ind := configIndicator(f.State, pass, partial, skip, fail)
		fmt.Fprintf(w, "  %s %-40s %s (%s)\n", ind, f.Path, f.State, f.Category)
	}
	fmt.Fprintln(w)

	// Dependency Health (detailed)
	fmt.Fprintf(w, "Dependency Health: %.0f%%\n", report.Dependencies.Score)
	for _, eco := range report.Dependencies.Ecosystems {
		if !eco.Detected {
			continue
		}
		total := eco.VulnCounts.Total()
		fmt.Fprintf(w, "  %-20s lockfile=%-10s vulns=%d (C:%d H:%d M:%d L:%d)\n",
			eco.Name, eco.LockFile, total,
			eco.VulnCounts.Critical, eco.VulnCounts.High,
			eco.VulnCounts.Moderate, eco.VulnCounts.Low)
	}
	fmt.Fprintln(w)

	// Tools
	if len(report.Tools) > 0 {
		fmt.Fprintln(w, "Tools:")
		for _, t := range report.Tools {
			ind := pass
			if !t.Enabled {
				ind = skip
			} else if !t.Available {
				ind = fail
			}
			fmt.Fprintf(w, "  %s %-25s enabled=%v available=%v\n",
				ind, t.Name, t.Enabled, t.Available)
		}
		fmt.Fprintln(w)
	}

	// Drift (detailed)
	if report.Drift.TotalFindings > 0 {
		fmt.Fprintf(w, "Drift: %d finding(s)\n", report.Drift.TotalFindings)
		for _, cat := range report.Drift.Categories {
			if len(cat.Findings) == 0 {
				continue
			}
			fmt.Fprintf(w, "  %s:\n", cat.Name)
			for _, f := range cat.Findings {
				sevInd := driftIndicator(f.Severity, pass, partial, skip, fail)
				fmt.Fprintf(w, "    %s [%s] %s\n", sevInd, f.Severity, f.Subject)
				fmt.Fprintf(w, "        %s\n", f.Description)
				if f.Remediation != "" {
					fmt.Fprintf(w, "        Fix: %s\n", f.Remediation)
				}
			}
		}
		fmt.Fprintln(w)
	}

	return nil
}

// renderFix outputs only remediation commands, one per line, suitable for
// piping to a shell or displaying as actionable items.
func renderFix(report *posture.PostureReport, w io.Writer) error {
	seen := make(map[string]bool)
	for _, cat := range report.Drift.Categories {
		for _, f := range cat.Findings {
			if f.Remediation == "" {
				continue
			}
			// Deduplicate identical remediations.
			if seen[f.Remediation] {
				continue
			}
			seen[f.Remediation] = true
			fmt.Fprintln(w, f.Remediation)
		}
	}

	// If conformance baseline fails, suggest qsdev init.
	if !report.Conformance.Baseline.Pass {
		for _, c := range report.Conformance.Baseline.Checks {
			if c.Pass {
				continue
			}
			remediation := conformanceRemediation(c.Name)
			if remediation != "" && !seen[remediation] {
				seen[remediation] = true
				fmt.Fprintln(w, remediation)
			}
		}
	}

	return nil
}

// conformanceRemediation returns a suggested remediation for a failed
// conformance check.
func conformanceRemediation(checkName string) string {
	switch checkName {
	case "lock-files-present":
		return "Run your package manager's install/lock command to generate lock files"
	case "no-critical-vulns":
		return "Run qsdev check --scan to identify and remediate critical vulnerabilities"
	case "claude-md-present":
		return "Run qsdev init to generate CLAUDE.md"
	case "settings-json-present":
		return "Run qsdev init to generate .claude/settings.json"
	case "high-weight-layers-enabled":
		return "Run qsdev enable attach-guard to enable critical defense layers"
	case "pre-commit-hooks":
		return "Run qsdev init to configure pre-commit hooks"
	default:
		return ""
	}
}

// renderConformanceLevel outputs a conformance level's checks.
func renderConformanceLevel(w io.Writer, name string, level posture.ConformanceLevel, pass, fail string) {
	ind := pass
	verdict := "PASS"
	if !level.Pass {
		ind = fail
		verdict = "FAIL"
	}
	fmt.Fprintf(w, "  %s %s: %s\n", ind, name, verdict)
	for _, c := range level.Checks {
		checkInd := pass
		if !c.Pass {
			checkInd = fail
		}
		reason := ""
		if c.Reason != "" {
			reason = " -- " + c.Reason
		}
		fmt.Fprintf(w, "    %s %s%s\n", checkInd, c.Name, reason)
	}
}

// indicatorForLayer returns the appropriate indicator for a layer status.
func indicatorForLayer(status posture.LayerStatus, pass, partial, skip, fail string) string {
	switch status {
	case posture.LayerEnabled:
		return pass
	case posture.LayerPartial:
		return partial
	case posture.LayerDisabled:
		return fail
	case posture.LayerNotApplicable:
		return skip
	default:
		return skip
	}
}

// configIndicator returns the appropriate indicator for a config file state.
func configIndicator(state string, pass, _, skip, fail string) string {
	switch state {
	case "current":
		return pass
	case "modified":
		return pass // modified human-edited is OK
	case "outdated":
		return fail
	case "missing", "corrupt":
		return fail
	default:
		return skip
	}
}

// driftIndicator returns the appropriate indicator for a drift severity.
func driftIndicator(severity drift.Severity, _, _, _, fail string) string {
	switch severity {
	case drift.Critical, drift.Error:
		return fail
	case drift.Warning:
		return strings.Replace(fail, "!", "~", 1)
	default:
		return " "
	}
}
