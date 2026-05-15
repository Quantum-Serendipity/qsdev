package posture

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderText_QuietModeSingleLine(t *testing.T) {
	report := &PostureReport{
		Score: AggregateScore{Total: 82.3, Grade: "B-"},
	}

	var buf bytes.Buffer
	opts := RenderOptions{Quiet: true}
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
	report := &PostureReport{
		Score: AggregateScore{Total: 89.5, Grade: "A-"},
	}

	var buf bytes.Buffer
	opts := RenderOptions{Quiet: true}
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
	report := &PostureReport{
		ProjectName: "test-project",
		ProjectPath: "/tmp/test",
		Score:       AggregateScore{Total: 85, Grade: "B"},
		Conformance: ConformanceResult{
			Baseline: ConformanceLevel{Pass: true, Checks: []ConformanceCheck{}},
			Enhanced: ConformanceLevel{Pass: false, Checks: []ConformanceCheck{}},
		},
		Defense: DefenseCoverage{
			Enabled: 5,
			Total:   10,
			Score:   75,
			Layers: []DefenseLayer{
				{Name: "sast", Status: LayerEnabled, Weight: WeightMedium},
				{Name: "nix-hardening", Status: LayerDisabled, Weight: WeightMedium},
			},
		},
		Config: ConfigHealth{
			Score:   80,
			Current: 4,
			Total:   5,
			Files:   []ConfigFileInfo{},
		},
		Dependencies: DependencyHealth{
			Score:      90,
			Ecosystems: []EcosystemStatus{},
			Totals:     VulnSeverityCounts{High: 2},
		},
		Drift: DriftReport{
			Categories:    []DriftCategory{},
			TotalFindings: 0,
			BySeverity:    make(map[DriftSeverity]int),
		},
		Tools:      []ToolStatus{},
		Ecosystems: []EcosystemStatus{},
	}

	var buf bytes.Buffer
	opts := RenderOptions{UseColor: false}
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
	report := &PostureReport{
		Conformance: ConformanceResult{
			Baseline: ConformanceLevel{
				Pass: false,
				Checks: []ConformanceCheck{
					{Name: "claude-md-present", Pass: false, Reason: "CLAUDE.md not found"},
				},
			},
			Enhanced: ConformanceLevel{Checks: []ConformanceCheck{}},
		},
		Drift: DriftReport{
			Categories: []DriftCategory{
				{
					Name: "test-category",
					Findings: []DriftFinding{
						{
							Severity:    DriftWarning,
							Subject:     "test.yml",
							Description: "File was modified",
							Remediation: "Run qsdev update to regenerate this file",
						},
						{
							Severity:    DriftInfo,
							Subject:     "info-only",
							Description: "Just information",
							// No remediation
						},
					},
				},
			},
			TotalFindings: 2,
			BySeverity:    map[DriftSeverity]int{DriftWarning: 1, DriftInfo: 1},
		},
	}

	var buf bytes.Buffer
	opts := RenderOptions{Fix: true}
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

func TestRenderText_FixModeDeduplicatesRemediations(t *testing.T) {
	report := &PostureReport{
		Conformance: ConformanceResult{
			Baseline: ConformanceLevel{Pass: true, Checks: []ConformanceCheck{}},
			Enhanced: ConformanceLevel{Pass: true, Checks: []ConformanceCheck{}},
		},
		Drift: DriftReport{
			Categories: []DriftCategory{
				{
					Name: "cat",
					Findings: []DriftFinding{
						{Remediation: "Run qsdev update"},
						{Remediation: "Run qsdev update"}, // duplicate
						{Remediation: "Run qsdev init"},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	opts := RenderOptions{Fix: true}
	if err := RenderText(report, &buf, opts); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 deduplicated lines, got %d: %v", len(lines), lines)
	}
}

func TestRenderText_FixModeNoRemediations(t *testing.T) {
	report := &PostureReport{
		Conformance: ConformanceResult{
			Baseline: ConformanceLevel{Pass: true, Checks: []ConformanceCheck{}},
			Enhanced: ConformanceLevel{Pass: true, Checks: []ConformanceCheck{}},
		},
		Drift: DriftReport{
			Categories: []DriftCategory{},
		},
	}

	var buf bytes.Buffer
	opts := RenderOptions{Fix: true}
	if err := RenderText(report, &buf, opts); err != nil {
		t.Fatal(err)
	}

	if buf.Len() != 0 {
		t.Errorf("expected empty output when no remediations, got %q", buf.String())
	}
}

func TestRenderText_VerboseContainsDetails(t *testing.T) {
	report := &PostureReport{
		SchemaVersion: SchemaVersion,
		QsdevVersion:   "1.0.0",
		ProjectName:   "verbose-test",
		ProjectPath:   "/tmp/test",
		Score:         AggregateScore{Total: 85, Grade: "B", Defense: 90, Config: 80, DepHealth: 75},
		Conformance: ConformanceResult{
			Baseline: ConformanceLevel{
				Pass: true,
				Checks: []ConformanceCheck{
					{Name: "lock-files-present", Pass: true, Reason: "ok"},
				},
			},
			Enhanced: ConformanceLevel{
				Pass: false,
				Checks: []ConformanceCheck{
					{Name: "sast-enabled", Pass: false, Reason: "semgrep not enabled"},
				},
			},
		},
		Defense: DefenseCoverage{
			Enabled: 5,
			Total:   10,
			Score:   90,
			Layers: []DefenseLayer{
				{Name: "sast", Status: LayerPartial, Weight: WeightMedium, Score: 5, Reason: "config present but tool not enabled"},
			},
		},
		Config: ConfigHealth{
			Score: 80,
			Files: []ConfigFileInfo{
				{Path: "devenv.nix", State: "current", Category: "machine-owned"},
			},
		},
		Dependencies: DependencyHealth{
			Score:      75,
			Ecosystems: []EcosystemStatus{{Name: "go", Detected: true, LockFile: "valid", VulnCounts: VulnSeverityCounts{High: 1}}},
			Totals:     VulnSeverityCounts{High: 1},
		},
		Drift: DriftReport{
			Categories: []DriftCategory{
				{
					Name: "test",
					Findings: []DriftFinding{
						{Severity: DriftWarning, Subject: "test.yml", Description: "modified", Remediation: "Run qsdev update"},
					},
				},
			},
			TotalFindings: 1,
			BySeverity:    map[DriftSeverity]int{DriftWarning: 1},
		},
		Tools: []ToolStatus{
			{Name: "semgrep", Enabled: true, Available: true},
		},
	}

	var buf bytes.Buffer
	opts := RenderOptions{Verbose: true, UseColor: false}
	if err := RenderText(report, &buf, opts); err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	// Verbose should contain per-layer details.
	expectedPhrases := []string{
		"Schema: 1.0.0",
		"weight=medium",
		"score=5/10",
		"Reason:",
		"Conformance:",
		"Baseline: PASS",
		"Enhanced: FAIL",
		"devenv.nix",
		"current",
		"machine-owned",
		"Drift: 1 finding(s)",
		"Fix: Run qsdev update",
		"semgrep",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(output, phrase) {
			t.Errorf("verbose output missing phrase %q:\n%s", phrase, output)
		}
	}
}

func TestRenderText_DefaultWithNoColor(t *testing.T) {
	report := &PostureReport{
		ProjectName: "test",
		ProjectPath: "/tmp/test",
		Score:       AggregateScore{Total: 85, Grade: "B"},
		Conformance: ConformanceResult{
			Baseline: ConformanceLevel{Pass: true, Checks: []ConformanceCheck{}},
			Enhanced: ConformanceLevel{Pass: true, Checks: []ConformanceCheck{}},
		},
		Defense: DefenseCoverage{
			Enabled: 5,
			Total:   10,
			Score:   75,
			Layers:  []DefenseLayer{},
		},
		Config: ConfigHealth{
			Score: 100,
			Files: []ConfigFileInfo{},
		},
		Dependencies: DependencyHealth{
			Score: 100,
		},
		Drift: DriftReport{
			Categories: []DriftCategory{},
			BySeverity: make(map[DriftSeverity]int),
		},
	}

	var buf bytes.Buffer
	opts := RenderOptions{UseColor: false}
	if err := RenderText(report, &buf, opts); err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	// No color should use [OK] style indicators, not ANSI sequences.
	if strings.Contains(output, "\033[") {
		t.Error("no-color output should not contain ANSI escape sequences")
	}
}
