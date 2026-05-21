package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
)

func TestRenderText_QuietModeSingleLine(t *testing.T) {
	report := &posture.PostureReport{
		Score: posture.AggregateScore{Total: 82.3, Grade: "B-"},
	}

	var buf bytes.Buffer
	opts := Options{Quiet: true}
	if err := RenderText(report, &buf, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	want := "82/100 B-\n"
	if got != want {
		t.Errorf("quiet mode output = %q, want %q", got, want)
	}
}

func TestRenderText_QuietModeRounding(t *testing.T) {
	report := &posture.PostureReport{
		Score: posture.AggregateScore{Total: 89.5, Grade: "A-"},
	}

	var buf bytes.Buffer
	opts := Options{Quiet: true}
	if err := RenderText(report, &buf, opts); err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	want := "90/100 A-\n"
	if got != want {
		t.Errorf("quiet mode output = %q, want %q", got, want)
	}
}

func TestRenderText_DefaultContainsSections(t *testing.T) {
	report := &posture.PostureReport{
		ProjectName: "test-project",
		ProjectPath: "/tmp/test",
		Score:       posture.AggregateScore{Total: 85, Grade: "B"},
		Conformance: posture.ConformanceResult{
			Baseline: posture.ConformanceLevel{Pass: true, Checks: []posture.ConformanceCheck{}},
			Enhanced: posture.ConformanceLevel{Pass: false, Checks: []posture.ConformanceCheck{}},
		},
		Defense: posture.DefenseCoverage{
			Enabled: 5,
			Total:   10,
			Score:   75,
			Layers: []posture.DefenseLayer{
				{Name: "sast", Status: posture.LayerEnabled, Weight: posture.WeightMedium},
				{Name: "nix-hardening", Status: posture.LayerDisabled, Weight: posture.WeightMedium},
			},
		},
		Config: posture.ConfigHealth{
			Score:   80,
			Current: 4,
			Total:   5,
			Files:   []posture.ConfigFileInfo{},
		},
		Dependencies: posture.DependencyHealth{
			Score:      90,
			Ecosystems: []posture.EcosystemStatus{},
			Totals:     posture.VulnSeverityCounts{High: 2},
		},
		Drift: drift.Report{
			Categories:    []drift.Category{},
			TotalFindings: 0,
			BySeverity:    make(map[drift.Severity]int),
		},
		Tools:      []posture.ToolStatus{},
		Ecosystems: []posture.EcosystemStatus{},
	}

	var buf bytes.Buffer
	opts := Options{UseColor: false}
	if err := RenderText(report, &buf, opts); err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	// Check that key sections are present.
	expectedPhrases := []string{
		"Security Posture: test-project",
		"Score: 85/100 (B)",
		"Conformance:",
		"Defense Coverage: 5/10 layers",
		"Config Health: 80%",
		"Dependency Health: 90%",
		"0 critical, 2 high",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(output, phrase) {
			t.Errorf("missing expected phrase %q in default output:\n%s", phrase, output)
		}
	}

	// Should contain footer hint.
	if !strings.Contains(output, "qsdev status --verbose") {
		t.Error("missing footer hint about --verbose")
	}
}

func TestRenderText_FixModeOnlyRemediation(t *testing.T) {
	report := &posture.PostureReport{
		Conformance: posture.ConformanceResult{
			Baseline: posture.ConformanceLevel{
				Pass: false,
				Checks: []posture.ConformanceCheck{
					{Name: "claude-md-present", Pass: false, Reason: "CLAUDE.md not found"},
				},
			},
			Enhanced: posture.ConformanceLevel{Checks: []posture.ConformanceCheck{}},
		},
		Drift: drift.Report{
			Categories: []drift.Category{
				{
					Name: "test-category",
					Findings: []drift.Finding{
						{
							Severity:    drift.Warning,
							Subject:     "test.yml",
							Description: "File was modified",
							Remediation: "Run qsdev update to regenerate this file",
						},
						{
							Severity:    drift.Info,
							Subject:     "info-only",
							Description: "Just information",
							// No remediation
						},
					},
				},
			},
			TotalFindings: 2,
			BySeverity:    map[drift.Severity]int{drift.Warning: 1, drift.Info: 1},
		},
	}

	var buf bytes.Buffer
	opts := Options{Fix: true}
	if err := RenderText(report, &buf, opts); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have exactly 2 remediation lines (one from drift, one from conformance).
	if len(lines) != 2 {
		t.Errorf("expected 2 remediation lines, got %d: %q", len(lines), output)
	}

	// Should contain the drift remediation.
	if !strings.Contains(output, "Run qsdev update to regenerate this file") {
		t.Error("missing drift remediation")
	}

	// Should contain the conformance remediation.
	if !strings.Contains(output, "Run qsdev init to generate CLAUDE.md") {
		t.Error("missing conformance remediation for claude-md-present")
	}

	// Should NOT contain descriptions or section headers.
	if strings.Contains(output, "Security Posture") {
		t.Error("fix mode should not contain section headers")
	}
	if strings.Contains(output, "File was modified") {
		t.Error("fix mode should not contain descriptions")
	}
}

func TestRenderText_DefaultTierLineShown(t *testing.T) {
	t.Parallel()

	report := &posture.PostureReport{
		ProjectName: "tier-test",
		ProjectPath: "/tmp/test",
		Score:       posture.AggregateScore{Total: 85, Grade: "B"},
		Tier: posture.ReportTierInfo{
			Current:  "standard",
			Position: 2,
			Total:    3,
			NextTier: "full",
		},
		Conformance: posture.ConformanceResult{
			Baseline: posture.ConformanceLevel{Pass: true, Checks: []posture.ConformanceCheck{}},
			Enhanced: posture.ConformanceLevel{Pass: true, Checks: []posture.ConformanceCheck{}},
		},
		Defense:      posture.DefenseCoverage{Layers: []posture.DefenseLayer{}},
		Config:       posture.ConfigHealth{Files: []posture.ConfigFileInfo{}},
		Dependencies: posture.DependencyHealth{},
		Drift:        drift.Report{Categories: []drift.Category{}, BySeverity: make(map[drift.Severity]int)},
	}

	var buf bytes.Buffer
	opts := Options{UseColor: false}
	if err := RenderText(report, &buf, opts); err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	if !strings.Contains(output, "Tier: standard (2/3)") {
		t.Errorf("missing tier line in default output:\n%s", output)
	}
	if !strings.Contains(output, "Next: qsdev init --tier full --dry-run") {
		t.Errorf("missing next-tier hint in default output:\n%s", output)
	}
	// Footer should also have the upgrade hint.
	if !strings.Contains(output, "Upgrade tier: qsdev init --tier full --dry-run") {
		t.Errorf("missing upgrade tier footer hint:\n%s", output)
	}
}

func TestRenderText_DefaultTierHiddenWhenEmpty(t *testing.T) {
	t.Parallel()

	report := &posture.PostureReport{
		ProjectName: "no-tier",
		ProjectPath: "/tmp/test",
		Score:       posture.AggregateScore{Total: 85, Grade: "B"},
		// Tier is zero value — Current is empty.
		Conformance: posture.ConformanceResult{
			Baseline: posture.ConformanceLevel{Pass: true, Checks: []posture.ConformanceCheck{}},
			Enhanced: posture.ConformanceLevel{Pass: true, Checks: []posture.ConformanceCheck{}},
		},
		Defense:      posture.DefenseCoverage{Layers: []posture.DefenseLayer{}},
		Config:       posture.ConfigHealth{Files: []posture.ConfigFileInfo{}},
		Dependencies: posture.DependencyHealth{},
		Drift:        drift.Report{Categories: []drift.Category{}, BySeverity: make(map[drift.Severity]int)},
	}

	var buf bytes.Buffer
	opts := Options{UseColor: false}
	if err := RenderText(report, &buf, opts); err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	if strings.Contains(output, "Tier:") {
		t.Errorf("tier line should not appear when TierInfo.Current is empty:\n%s", output)
	}
	if strings.Contains(output, "Upgrade tier:") {
		t.Errorf("upgrade tier footer should not appear when TierInfo.Current is empty:\n%s", output)
	}
}

func TestRenderText_DefaultTierFullNoNextTier(t *testing.T) {
	t.Parallel()

	report := &posture.PostureReport{
		ProjectName: "full-tier",
		ProjectPath: "/tmp/test",
		Score:       posture.AggregateScore{Total: 95, Grade: "A"},
		Tier: posture.ReportTierInfo{
			Current:  "full",
			Position: 3,
			Total:    3,
			// NextTier is empty — at max tier.
		},
		Conformance: posture.ConformanceResult{
			Baseline: posture.ConformanceLevel{Pass: true, Checks: []posture.ConformanceCheck{}},
			Enhanced: posture.ConformanceLevel{Pass: true, Checks: []posture.ConformanceCheck{}},
		},
		Defense:      posture.DefenseCoverage{Layers: []posture.DefenseLayer{}},
		Config:       posture.ConfigHealth{Files: []posture.ConfigFileInfo{}},
		Dependencies: posture.DependencyHealth{},
		Drift:        drift.Report{Categories: []drift.Category{}, BySeverity: make(map[drift.Severity]int)},
	}

	var buf bytes.Buffer
	opts := Options{UseColor: false}
	if err := RenderText(report, &buf, opts); err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	if !strings.Contains(output, "Tier: full (3/3)") {
		t.Errorf("missing tier line for full tier:\n%s", output)
	}
	if strings.Contains(output, "Next:") {
		t.Errorf("full tier should not show Next hint:\n%s", output)
	}
	if strings.Contains(output, "Upgrade tier:") {
		t.Errorf("full tier should not show upgrade footer:\n%s", output)
	}
}

func TestRenderText_VerboseTierExpanded(t *testing.T) {
	t.Parallel()

	report := &posture.PostureReport{
		ProjectName:   "verbose-tier",
		ProjectPath:   "/tmp/test",
		SchemaVersion: posture.SchemaVersion,
		QsdevVersion:  "1.0.0",
		Score:         posture.AggregateScore{Total: 85, Grade: "B"},
		Tier: posture.ReportTierInfo{
			Current:  "standard",
			Position: 2,
			Total:    3,
			NextTier: "full",
		},
		Conformance: posture.ConformanceResult{
			Baseline: posture.ConformanceLevel{Pass: true, Checks: []posture.ConformanceCheck{}},
			Enhanced: posture.ConformanceLevel{Pass: true, Checks: []posture.ConformanceCheck{}},
		},
		Defense:      posture.DefenseCoverage{Layers: []posture.DefenseLayer{}},
		Config:       posture.ConfigHealth{Files: []posture.ConfigFileInfo{}},
		Dependencies: posture.DependencyHealth{Ecosystems: []posture.EcosystemStatus{}},
		Drift:        drift.Report{Categories: []drift.Category{}, BySeverity: make(map[drift.Severity]int)},
	}

	var buf bytes.Buffer
	opts := Options{Verbose: true, UseColor: false}
	if err := RenderText(report, &buf, opts); err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	if !strings.Contains(output, "Tier: standard (2/3)") {
		t.Errorf("missing tier header in verbose output:\n%s", output)
	}
	if !strings.Contains(output, "Next tier: full") {
		t.Errorf("missing next tier line in verbose output:\n%s", output)
	}
	if !strings.Contains(output, "Full tooling: MCP servers, agent tools, consulting workflows, AlwaysOn tools") {
		t.Errorf("missing tier description in verbose output:\n%s", output)
	}
	if !strings.Contains(output, "Upgrade: qsdev init --tier full --dry-run") {
		t.Errorf("missing upgrade command in verbose output:\n%s", output)
	}
}

func TestRenderText_VerboseTierFullNoUpgrade(t *testing.T) {
	t.Parallel()

	report := &posture.PostureReport{
		ProjectName:   "verbose-full",
		ProjectPath:   "/tmp/test",
		SchemaVersion: posture.SchemaVersion,
		QsdevVersion:  "1.0.0",
		Score:         posture.AggregateScore{Total: 95, Grade: "A"},
		Tier: posture.ReportTierInfo{
			Current:  "full",
			Position: 3,
			Total:    3,
		},
		Conformance: posture.ConformanceResult{
			Baseline: posture.ConformanceLevel{Pass: true, Checks: []posture.ConformanceCheck{}},
			Enhanced: posture.ConformanceLevel{Pass: true, Checks: []posture.ConformanceCheck{}},
		},
		Defense:      posture.DefenseCoverage{Layers: []posture.DefenseLayer{}},
		Config:       posture.ConfigHealth{Files: []posture.ConfigFileInfo{}},
		Dependencies: posture.DependencyHealth{Ecosystems: []posture.EcosystemStatus{}},
		Drift:        drift.Report{Categories: []drift.Category{}, BySeverity: make(map[drift.Severity]int)},
	}

	var buf bytes.Buffer
	opts := Options{Verbose: true, UseColor: false}
	if err := RenderText(report, &buf, opts); err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	if !strings.Contains(output, "Tier: full (3/3)") {
		t.Errorf("missing tier line for full tier in verbose:\n%s", output)
	}
	if strings.Contains(output, "Next tier:") {
		t.Errorf("full tier should not show next tier in verbose:\n%s", output)
	}
	if strings.Contains(output, "Upgrade:") {
		t.Errorf("full tier should not show upgrade in verbose:\n%s", output)
	}
}

func TestRenderText_DefaultWithNoColor(t *testing.T) {
	report := &posture.PostureReport{
		ProjectName: "test",
		ProjectPath: "/tmp/test",
		Score:       posture.AggregateScore{Total: 85, Grade: "B"},
		Conformance: posture.ConformanceResult{
			Baseline: posture.ConformanceLevel{Pass: true, Checks: []posture.ConformanceCheck{}},
			Enhanced: posture.ConformanceLevel{Pass: true, Checks: []posture.ConformanceCheck{}},
		},
		Defense: posture.DefenseCoverage{
			Enabled: 5,
			Total:   10,
			Score:   75,
			Layers:  []posture.DefenseLayer{},
		},
		Config: posture.ConfigHealth{
			Score: 100,
			Files: []posture.ConfigFileInfo{},
		},
		Dependencies: posture.DependencyHealth{
			Score: 100,
		},
		Drift: drift.Report{
			Categories: []drift.Category{},
			BySeverity: make(map[drift.Severity]int),
		},
	}

	var buf bytes.Buffer
	opts := Options{UseColor: false}
	if err := RenderText(report, &buf, opts); err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	// No color should use [OK] style indicators, not ANSI sequences.
	if strings.Contains(output, "\033[") {
		t.Error("no-color output should not contain ANSI escape sequences")
	}
}
