package posture

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestRenderJSON_SchemaVersionPresent(t *testing.T) {
	report := &PostureReport{
		SchemaVersion: "old-version", // should be overwritten
		GeneratedAt:   time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
		QsdevVersion:   "1.0.0",
		ProjectName:   "test-project",
		ProjectPath:   "/tmp/test",
		Score:         AggregateScore{Total: 85, Grade: "B"},
		Conformance: ConformanceResult{
			Baseline: ConformanceLevel{Checks: []ConformanceCheck{}},
			Enhanced: ConformanceLevel{Checks: []ConformanceCheck{}},
		},
		Defense:      DefenseCoverage{Layers: []DefenseLayer{}},
		Config:       ConfigHealth{Files: []ConfigFileInfo{}},
		Dependencies: DependencyHealth{Ecosystems: []EcosystemStatus{}},
		Drift:        DriftReport{Categories: []DriftCategory{}, BySeverity: make(map[DriftSeverity]int)},
		Tools:        []ToolStatus{},
		Ecosystems:   []EcosystemStatus{},
	}

	data, err := RenderJSON(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain the current SchemaVersion, not the old one.
	if !strings.Contains(string(data), `"schemaVersion": "1.0.0"`) {
		t.Errorf("expected schemaVersion %q in output, got:\n%s", SchemaVersion, string(data))
	}
	if strings.Contains(string(data), `"old-version"`) {
		t.Error("old schema version should have been overwritten")
	}
}

func TestRenderJSON_RoundTrip(t *testing.T) {
	report := &PostureReport{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
		QsdevVersion:   "1.0.0",
		ProjectName:   "round-trip-test",
		ProjectPath:   "/tmp/test",
		Score:         AggregateScore{Total: 82.5, Grade: "B-", Defense: 90, Config: 80, DepHealth: 75},
		Conformance: ConformanceResult{
			Baseline: ConformanceLevel{
				Pass:   true,
				Checks: []ConformanceCheck{{Name: "test-check", Pass: true, Reason: "ok"}},
			},
			Enhanced: ConformanceLevel{
				Pass:   false,
				Checks: []ConformanceCheck{{Name: "enhanced-check", Pass: false, Reason: "missing"}},
			},
		},
		Defense: DefenseCoverage{
			Score:   90,
			Enabled: 5,
			Total:   8,
			Layers: []DefenseLayer{
				{Name: "sast", Status: LayerEnabled, Weight: WeightMedium},
			},
		},
		Config: ConfigHealth{
			Score: 80,
			Total: 5,
			Files: []ConfigFileInfo{
				{Path: "test.yml", State: "current", Category: "machine-owned"},
			},
		},
		Dependencies: DependencyHealth{
			Score:      75,
			Ecosystems: []EcosystemStatus{{Name: "go", Detected: true, LockFile: "valid"}},
			Totals:     VulnSeverityCounts{High: 2},
		},
		Drift: DriftReport{
			Categories:    []DriftCategory{},
			TotalFindings: 0,
			BySeverity:    make(map[DriftSeverity]int),
		},
		Tools:      []ToolStatus{{Name: "semgrep", Enabled: true, Available: true}},
		Ecosystems: []EcosystemStatus{{Name: "go", Detected: true}},
	}

	data, err := RenderJSON(report)
	if err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}

	var decoded PostureReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("round-trip unmarshal error: %v", err)
	}

	if decoded.ProjectName != "round-trip-test" {
		t.Errorf("ProjectName = %q, want %q", decoded.ProjectName, "round-trip-test")
	}
	if decoded.Score.Total != 82.5 {
		t.Errorf("Score.Total = %f, want 82.5", decoded.Score.Total)
	}
	if decoded.Dependencies.Totals.High != 2 {
		t.Errorf("Dependencies.Totals.High = %d, want 2", decoded.Dependencies.Totals.High)
	}
	if len(decoded.Defense.Layers) != 1 {
		t.Errorf("Defense.Layers length = %d, want 1", len(decoded.Defense.Layers))
	}
}

func TestRenderJSON_EmptyReport(t *testing.T) {
	report := &PostureReport{}

	data, err := RenderJSON(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify nil slices are serialized as [] not null.
	if strings.Contains(string(data), ": null") {
		t.Errorf("found null in JSON output; all slices should be empty arrays:\n%s", string(data))
	}

	// Verify SchemaVersion was set.
	if !strings.Contains(string(data), `"schemaVersion": "1.0.0"`) {
		t.Error("SchemaVersion not set in empty report output")
	}
}

func TestRenderJSON_Deterministic(t *testing.T) {
	report := &PostureReport{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
		QsdevVersion:   "1.0.0",
		ProjectName:   "determinism",
		ProjectPath:   "/tmp/test",
		Score:         AggregateScore{Total: 85, Grade: "B"},
		Conformance: ConformanceResult{
			Baseline: ConformanceLevel{Checks: []ConformanceCheck{}},
			Enhanced: ConformanceLevel{Checks: []ConformanceCheck{}},
		},
		Defense:      DefenseCoverage{Layers: []DefenseLayer{}},
		Config:       ConfigHealth{Files: []ConfigFileInfo{}},
		Dependencies: DependencyHealth{Ecosystems: []EcosystemStatus{}},
		Drift:        DriftReport{Categories: []DriftCategory{}, BySeverity: make(map[DriftSeverity]int)},
		Tools:        []ToolStatus{},
		Ecosystems:   []EcosystemStatus{},
	}

	first, err := RenderJSON(report)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 50; i++ {
		got, err := RenderJSON(report)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(first, got) {
			t.Fatalf("iteration %d: non-deterministic output", i)
		}
	}
}

func TestRenderJSON_TrailingNewline(t *testing.T) {
	report := &PostureReport{
		SchemaVersion: SchemaVersion,
		Conformance: ConformanceResult{
			Baseline: ConformanceLevel{Checks: []ConformanceCheck{}},
			Enhanced: ConformanceLevel{Checks: []ConformanceCheck{}},
		},
		Defense:      DefenseCoverage{Layers: []DefenseLayer{}},
		Config:       ConfigHealth{Files: []ConfigFileInfo{}},
		Dependencies: DependencyHealth{Ecosystems: []EcosystemStatus{}},
		Drift:        DriftReport{Categories: []DriftCategory{}, BySeverity: make(map[DriftSeverity]int)},
		Tools:        []ToolStatus{},
		Ecosystems:   []EcosystemStatus{},
	}

	data, err := RenderJSON(report)
	if err != nil {
		t.Fatal(err)
	}

	if len(data) == 0 {
		t.Fatal("empty output")
	}
	if data[len(data)-1] != '\n' {
		t.Errorf("expected trailing newline, last byte is %q", data[len(data)-1])
	}
}
