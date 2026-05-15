package posture

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderSARIF_ValidStructure(t *testing.T) {
	report := &PostureReport{
		QsdevVersion: "1.0.0",
		Defense: DefenseCoverage{
			Layers: []DefenseLayer{
				{Name: "sast", Status: LayerEnabled, Weight: WeightMedium},
				{Name: "nix-hardening", Status: LayerDisabled, Weight: WeightMedium, Reason: "not configured"},
			},
		},
		Config:       ConfigHealth{Files: []ConfigFileInfo{}},
		Dependencies: DependencyHealth{},
		Drift: DriftReport{
			Categories: []DriftCategory{},
			BySeverity: make(map[DriftSeverity]int),
		},
	}

	data, err := RenderSARIF(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's valid JSON.
	var log sarifLog
	if err := json.Unmarshal(data, &log); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Verify SARIF version.
	if log.Version != "2.1.0" {
		t.Errorf("SARIF version = %q, want %q", log.Version, "2.1.0")
	}

	// Verify schema reference.
	if !strings.Contains(log.Schema, "sarif-schema-2.1.0") {
		t.Errorf("schema URI missing sarif-schema-2.1.0: %q", log.Schema)
	}

	// Verify exactly one run.
	if len(log.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(log.Runs))
	}

	// Verify tool name.
	if log.Runs[0].Tool.Driver.Name != "qsdev" {
		t.Errorf("tool name = %q, want %q", log.Runs[0].Tool.Driver.Name, "qsdev")
	}

	// Verify tool version.
	if log.Runs[0].Tool.Driver.Version != "1.0.0" {
		t.Errorf("tool version = %q, want %q", log.Runs[0].Tool.Driver.Version, "1.0.0")
	}

	// Trailing newline.
	if data[len(data)-1] != '\n' {
		t.Error("expected trailing newline")
	}
}

func TestRenderSARIF_AllRulesPresent(t *testing.T) {
	report := &PostureReport{
		QsdevVersion: "1.0.0",
		Defense:     DefenseCoverage{Layers: []DefenseLayer{}},
		Config:      ConfigHealth{Files: []ConfigFileInfo{}},
		Dependencies: DependencyHealth{
			Ecosystems: []EcosystemStatus{},
		},
		Drift: DriftReport{
			Categories: []DriftCategory{},
			BySeverity: make(map[DriftSeverity]int),
		},
	}

	data, err := RenderSARIF(report)
	if err != nil {
		t.Fatal(err)
	}

	var log sarifLog
	if err := json.Unmarshal(data, &log); err != nil {
		t.Fatal(err)
	}

	rules := log.Runs[0].Tool.Driver.Rules
	if len(rules) != 12 {
		t.Errorf("expected 12 rules, got %d", len(rules))
	}

	expectedRules := []string{
		"qsdev/defense-disabled", "qsdev/defense-partial",
		"qsdev/config-missing", "qsdev/config-outdated", "qsdev/config-modified",
		"qsdev/vuln-critical", "qsdev/vuln-high",
		"qsdev/lockfile-missing", "qsdev/lockfile-stale",
		"qsdev/hooks-not-installed", "qsdev/markers-broken", "qsdev/tool-unavailable",
	}

	ruleIDs := make(map[string]bool)
	for _, r := range rules {
		ruleIDs[r.ID] = true
	}

	for _, expected := range expectedRules {
		if !ruleIDs[expected] {
			t.Errorf("missing rule: %s", expected)
		}
	}
}

func TestRenderSARIF_DisabledDefenseEmitsResult(t *testing.T) {
	report := &PostureReport{
		QsdevVersion: "1.0.0",
		Defense: DefenseCoverage{
			Layers: []DefenseLayer{
				{Name: "sast", Status: LayerDisabled, Weight: WeightMedium, Reason: "semgrep not enabled"},
			},
		},
		Config:       ConfigHealth{Files: []ConfigFileInfo{}},
		Dependencies: DependencyHealth{},
		Drift: DriftReport{
			Categories: []DriftCategory{},
			BySeverity: make(map[DriftSeverity]int),
		},
	}

	data, err := RenderSARIF(report)
	if err != nil {
		t.Fatal(err)
	}

	var log sarifLog
	if err := json.Unmarshal(data, &log); err != nil {
		t.Fatal(err)
	}

	found := false
	for _, r := range log.Runs[0].Results {
		if r.RuleID == "qsdev/defense-disabled" {
			found = true
			if r.Level != "warning" {
				t.Errorf("expected level 'warning', got %q", r.Level)
			}
			if !strings.Contains(r.Message.Text, "sast") {
				t.Errorf("expected message to mention 'sast', got %q", r.Message.Text)
			}
		}
	}
	if !found {
		t.Error("expected a qsdev/defense-disabled result for disabled sast layer")
	}
}

func TestRenderSARIF_NotApplicableNoResult(t *testing.T) {
	report := &PostureReport{
		QsdevVersion: "1.0.0",
		Defense: DefenseCoverage{
			Layers: []DefenseLayer{
				{Name: "container-security", Status: LayerNotApplicable, Weight: WeightMedium},
			},
		},
		Config:       ConfigHealth{Files: []ConfigFileInfo{}},
		Dependencies: DependencyHealth{},
		Drift: DriftReport{
			Categories: []DriftCategory{},
			BySeverity: make(map[DriftSeverity]int),
		},
	}

	data, err := RenderSARIF(report)
	if err != nil {
		t.Fatal(err)
	}

	var log sarifLog
	if err := json.Unmarshal(data, &log); err != nil {
		t.Fatal(err)
	}

	for _, r := range log.Runs[0].Results {
		if strings.Contains(r.Message.Text, "container-security") {
			t.Error("not-applicable layer should not emit a result")
		}
	}
}

func TestRenderSARIF_EmptyReport(t *testing.T) {
	report := &PostureReport{
		QsdevVersion:  "1.0.0",
		Defense:      DefenseCoverage{Layers: []DefenseLayer{}},
		Config:       ConfigHealth{Files: []ConfigFileInfo{}},
		Dependencies: DependencyHealth{},
		Drift: DriftReport{
			Categories: []DriftCategory{},
			BySeverity: make(map[DriftSeverity]int),
		},
	}

	data, err := RenderSARIF(report)
	if err != nil {
		t.Fatal(err)
	}

	var log sarifLog
	if err := json.Unmarshal(data, &log); err != nil {
		t.Fatal(err)
	}

	if len(log.Runs[0].Results) != 0 {
		t.Errorf("expected 0 results for empty report, got %d", len(log.Runs[0].Results))
	}
}

func TestRenderSARIF_VulnResults(t *testing.T) {
	report := &PostureReport{
		QsdevVersion: "1.0.0",
		Defense:     DefenseCoverage{Layers: []DefenseLayer{}},
		Config:      ConfigHealth{Files: []ConfigFileInfo{}},
		Dependencies: DependencyHealth{
			Totals: VulnSeverityCounts{Critical: 2, High: 5},
		},
		Drift: DriftReport{
			Categories: []DriftCategory{},
			BySeverity: make(map[DriftSeverity]int),
		},
	}

	data, err := RenderSARIF(report)
	if err != nil {
		t.Fatal(err)
	}

	var log sarifLog
	if err := json.Unmarshal(data, &log); err != nil {
		t.Fatal(err)
	}

	critFound := false
	highFound := false
	for _, r := range log.Runs[0].Results {
		if r.RuleID == "qsdev/vuln-critical" {
			critFound = true
			if r.Level != "error" {
				t.Errorf("vuln-critical level = %q, want %q", r.Level, "error")
			}
		}
		if r.RuleID == "qsdev/vuln-high" {
			highFound = true
			if r.Level != "warning" {
				t.Errorf("vuln-high level = %q, want %q", r.Level, "warning")
			}
		}
	}
	if !critFound {
		t.Error("expected qsdev/vuln-critical result")
	}
	if !highFound {
		t.Error("expected qsdev/vuln-high result")
	}
}

func TestRenderSARIF_PartialDefenseEmitsResult(t *testing.T) {
	report := &PostureReport{
		QsdevVersion: "1.0.0",
		Defense: DefenseCoverage{
			Layers: []DefenseLayer{
				{Name: "secrets-scanning", Status: LayerPartial, Weight: WeightMedium, Score: 5, Reason: "only gitleaks"},
			},
		},
		Config:       ConfigHealth{Files: []ConfigFileInfo{}},
		Dependencies: DependencyHealth{},
		Drift: DriftReport{
			Categories: []DriftCategory{},
			BySeverity: make(map[DriftSeverity]int),
		},
	}

	data, err := RenderSARIF(report)
	if err != nil {
		t.Fatal(err)
	}

	var log sarifLog
	if err := json.Unmarshal(data, &log); err != nil {
		t.Fatal(err)
	}

	found := false
	for _, r := range log.Runs[0].Results {
		if r.RuleID == "qsdev/defense-partial" {
			found = true
		}
	}
	if !found {
		t.Error("expected qsdev/defense-partial result for partial layer")
	}
}

func TestRenderSARIF_DriftFindings(t *testing.T) {
	report := &PostureReport{
		QsdevVersion: "1.0.0",
		Defense:     DefenseCoverage{Layers: []DefenseLayer{}},
		Config:      ConfigHealth{Files: []ConfigFileInfo{}},
		Dependencies: DependencyHealth{},
		Drift: DriftReport{
			Categories: []DriftCategory{
				{
					Name: categoryLockfileDrift,
					Findings: []DriftFinding{
						{Severity: DriftError, Subject: "Cargo.lock", Description: "missing lockfile"},
						{Severity: DriftWarning, Subject: "go.sum", Description: "stale lockfile"},
					},
				},
				{
					Name: categoryToolAvailability,
					Findings: []DriftFinding{
						{Severity: DriftWarning, Subject: "semgrep", Description: "semgrep not found"},
					},
				},
			},
			TotalFindings: 3,
			BySeverity:    map[DriftSeverity]int{DriftError: 1, DriftWarning: 2},
		},
	}

	data, err := RenderSARIF(report)
	if err != nil {
		t.Fatal(err)
	}

	var log sarifLog
	if err := json.Unmarshal(data, &log); err != nil {
		t.Fatal(err)
	}

	ruleIDCounts := make(map[string]int)
	for _, r := range log.Runs[0].Results {
		ruleIDCounts[r.RuleID]++
	}

	if ruleIDCounts["qsdev/lockfile-missing"] != 1 {
		t.Errorf("expected 1 lockfile-missing result, got %d", ruleIDCounts["qsdev/lockfile-missing"])
	}
	if ruleIDCounts["qsdev/lockfile-stale"] != 1 {
		t.Errorf("expected 1 lockfile-stale result, got %d", ruleIDCounts["qsdev/lockfile-stale"])
	}
	if ruleIDCounts["qsdev/tool-unavailable"] != 1 {
		t.Errorf("expected 1 tool-unavailable result, got %d", ruleIDCounts["qsdev/tool-unavailable"])
	}
}
