package render

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
)

func TestRenderSARIF_ValidStructure(t *testing.T) {
	report := &posture.PostureReport{
		QsdevVersion: "1.0.0",
		Defense: posture.DefenseCoverage{
			Layers: []posture.DefenseLayer{
				{Name: "sast", Status: posture.LayerEnabled, Weight: posture.WeightMedium},
				{Name: "nix-hardening", Status: posture.LayerDisabled, Weight: posture.WeightMedium, Reason: "not configured"},
			},
		},
		Config:       posture.ConfigHealth{Files: []posture.ConfigFileInfo{}},
		Dependencies: posture.DependencyHealth{},
		Drift: drift.Report{
			Categories: []drift.Category{},
			BySeverity: make(map[drift.Severity]int),
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

	// Trailing newline.
	if data[len(data)-1] != '\n' {
		t.Error("expected trailing newline")
	}
}

func TestRenderSARIF_DriftFindings(t *testing.T) {
	report := &posture.PostureReport{
		QsdevVersion: "1.0.0",
		Defense:      posture.DefenseCoverage{Layers: []posture.DefenseLayer{}},
		Config:       posture.ConfigHealth{Files: []posture.ConfigFileInfo{}},
		Dependencies: posture.DependencyHealth{},
		Drift: drift.Report{
			Categories: []drift.Category{
				{
					Name: categoryLockfileDrift,
					Findings: []drift.Finding{
						{Severity: drift.Error, Subject: "Cargo.lock", Description: "missing lockfile"},
						{Severity: drift.Warning, Subject: "go.sum", Description: "stale lockfile"},
					},
				},
				{
					Name: categoryToolAvailability,
					Findings: []drift.Finding{
						{Severity: drift.Warning, Subject: "semgrep", Description: "semgrep not found"},
					},
				},
			},
			TotalFindings: 3,
			BySeverity:    map[drift.Severity]int{drift.Error: 1, drift.Warning: 2},
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

func TestRenderSARIF_EmptyReport(t *testing.T) {
	report := &posture.PostureReport{
		QsdevVersion: "1.0.0",
		Defense:      posture.DefenseCoverage{Layers: []posture.DefenseLayer{}},
		Config:       posture.ConfigHealth{Files: []posture.ConfigFileInfo{}},
		Dependencies: posture.DependencyHealth{},
		Drift: drift.Report{
			Categories: []drift.Category{},
			BySeverity: make(map[drift.Severity]int),
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
