package render

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
)

func TestRenderJSON_SchemaVersionPresent(t *testing.T) {
	report := &posture.PostureReport{
		SchemaVersion: "old-version", // should be overwritten
		GeneratedAt:   time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
		QsdevVersion:  "1.0.0",
		ProjectName:   "test-project",
		ProjectPath:   "/tmp/test",
		Score:         posture.AggregateScore{Total: 85, Grade: "B"},
		Conformance: posture.ConformanceResult{
			Baseline: posture.ConformanceLevel{Checks: []posture.ConformanceCheck{}},
			Enhanced: posture.ConformanceLevel{Checks: []posture.ConformanceCheck{}},
		},
		Defense:      posture.DefenseCoverage{Layers: []posture.DefenseLayer{}},
		Config:       posture.ConfigHealth{Files: []posture.ConfigFileInfo{}},
		Dependencies: posture.DependencyHealth{Ecosystems: []posture.EcosystemStatus{}},
		Drift:        drift.Report{Categories: []drift.Category{}, BySeverity: make(map[drift.Severity]int)},
		Tools:        []posture.ToolStatus{},
		Ecosystems:   []posture.EcosystemStatus{},
	}

	data, err := RenderJSON(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain the current SchemaVersion, not the old one.
	if !strings.Contains(string(data), `"schemaVersion": "1.0.0"`) {
		t.Errorf("expected schemaVersion %q in output, got:\n%s", posture.SchemaVersion, string(data))
	}
	if strings.Contains(string(data), `"old-version"`) {
		t.Error("old schema version should have been overwritten")
	}
}

func TestRenderJSON_EmptyReport(t *testing.T) {
	report := &posture.PostureReport{}

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
	report := &posture.PostureReport{
		SchemaVersion: posture.SchemaVersion,
		GeneratedAt:   time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
		QsdevVersion:  "1.0.0",
		ProjectName:   "determinism",
		ProjectPath:   "/tmp/test",
		Score:         posture.AggregateScore{Total: 85, Grade: "B"},
		Conformance: posture.ConformanceResult{
			Baseline: posture.ConformanceLevel{Checks: []posture.ConformanceCheck{}},
			Enhanced: posture.ConformanceLevel{Checks: []posture.ConformanceCheck{}},
		},
		Defense:      posture.DefenseCoverage{Layers: []posture.DefenseLayer{}},
		Config:       posture.ConfigHealth{Files: []posture.ConfigFileInfo{}},
		Dependencies: posture.DependencyHealth{Ecosystems: []posture.EcosystemStatus{}},
		Drift:        drift.Report{Categories: []drift.Category{}, BySeverity: make(map[drift.Severity]int)},
		Tools:        []posture.ToolStatus{},
		Ecosystems:   []posture.EcosystemStatus{},
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
	report := &posture.PostureReport{
		SchemaVersion: posture.SchemaVersion,
		Conformance: posture.ConformanceResult{
			Baseline: posture.ConformanceLevel{Checks: []posture.ConformanceCheck{}},
			Enhanced: posture.ConformanceLevel{Checks: []posture.ConformanceCheck{}},
		},
		Defense:      posture.DefenseCoverage{Layers: []posture.DefenseLayer{}},
		Config:       posture.ConfigHealth{Files: []posture.ConfigFileInfo{}},
		Dependencies: posture.DependencyHealth{Ecosystems: []posture.EcosystemStatus{}},
		Drift:        drift.Report{Categories: []drift.Category{}, BySeverity: make(map[drift.Severity]int)},
		Tools:        []posture.ToolStatus{},
		Ecosystems:   []posture.EcosystemStatus{},
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

func TestRenderJSON_TierFieldPresent(t *testing.T) {
	report := &posture.PostureReport{
		SchemaVersion: posture.SchemaVersion,
		ProjectName:   "tier-json-test",
		Tier: posture.ReportTierInfo{
			Current:  "standard",
			Position: 2,
			Total:    3,
			NextTier: "full",
		},
		Conformance: posture.ConformanceResult{
			Baseline: posture.ConformanceLevel{Checks: []posture.ConformanceCheck{}},
			Enhanced: posture.ConformanceLevel{Checks: []posture.ConformanceCheck{}},
		},
		Defense:      posture.DefenseCoverage{Layers: []posture.DefenseLayer{}},
		Config:       posture.ConfigHealth{Files: []posture.ConfigFileInfo{}},
		Dependencies: posture.DependencyHealth{Ecosystems: []posture.EcosystemStatus{}},
		Drift:        drift.Report{Categories: []drift.Category{}, BySeverity: make(map[drift.Severity]int)},
		Tools:        []posture.ToolStatus{},
		Ecosystems:   []posture.EcosystemStatus{},
	}

	data, err := RenderJSON(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify tier fields are present.
	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"current": "standard"`) {
		t.Errorf("missing tier.current in JSON:\n%s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"position": 2`) {
		t.Errorf("missing tier.position in JSON:\n%s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"nextTier": "full"`) {
		t.Errorf("missing tier.nextTier in JSON:\n%s", jsonStr)
	}

	// Round-trip to verify deserialization.
	var decoded posture.PostureReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.Tier.Current != "standard" {
		t.Errorf("Tier.Current = %q, want %q", decoded.Tier.Current, "standard")
	}
	if decoded.Tier.Position != 2 {
		t.Errorf("Tier.Position = %d, want 2", decoded.Tier.Position)
	}
	if decoded.Tier.NextTier != "full" {
		t.Errorf("Tier.NextTier = %q, want %q", decoded.Tier.NextTier, "full")
	}
}

func TestRenderJSON_TierOmitsNextTierWhenEmpty(t *testing.T) {
	report := &posture.PostureReport{
		SchemaVersion: posture.SchemaVersion,
		Tier: posture.ReportTierInfo{
			Current:  "full",
			Position: 3,
			Total:    3,
			// NextTier is empty.
		},
		Conformance: posture.ConformanceResult{
			Baseline: posture.ConformanceLevel{Checks: []posture.ConformanceCheck{}},
			Enhanced: posture.ConformanceLevel{Checks: []posture.ConformanceCheck{}},
		},
		Defense:      posture.DefenseCoverage{Layers: []posture.DefenseLayer{}},
		Config:       posture.ConfigHealth{Files: []posture.ConfigFileInfo{}},
		Dependencies: posture.DependencyHealth{Ecosystems: []posture.EcosystemStatus{}},
		Drift:        drift.Report{Categories: []drift.Category{}, BySeverity: make(map[drift.Severity]int)},
		Tools:        []posture.ToolStatus{},
		Ecosystems:   []posture.EcosystemStatus{},
	}

	data, err := RenderJSON(report)
	if err != nil {
		t.Fatal(err)
	}

	// nextTier should be omitted when empty (omitempty).
	if strings.Contains(string(data), `"nextTier"`) {
		t.Errorf("nextTier should be omitted when empty:\n%s", string(data))
	}
}

func TestRenderJSON_RoundTrip(t *testing.T) {
	report := &posture.PostureReport{
		SchemaVersion: posture.SchemaVersion,
		GeneratedAt:   time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
		QsdevVersion:  "1.0.0",
		ProjectName:   "round-trip-test",
		ProjectPath:   "/tmp/test",
		Score:         posture.AggregateScore{Total: 82.5, Grade: "B-", Defense: 90, Config: 80, DepHealth: 75},
		Conformance: posture.ConformanceResult{
			Baseline: posture.ConformanceLevel{
				Pass:   true,
				Checks: []posture.ConformanceCheck{{Name: "test-check", Pass: true, Reason: "ok"}},
			},
			Enhanced: posture.ConformanceLevel{
				Pass:   false,
				Checks: []posture.ConformanceCheck{{Name: "enhanced-check", Pass: false, Reason: "missing"}},
			},
		},
		Defense: posture.DefenseCoverage{
			Score:   90,
			Enabled: 5,
			Total:   8,
			Layers: []posture.DefenseLayer{
				{Name: "sast", Status: posture.LayerEnabled, Weight: posture.WeightMedium},
			},
		},
		Config: posture.ConfigHealth{
			Score: 80,
			Total: 5,
			Files: []posture.ConfigFileInfo{
				{Path: "test.yml", State: "current", Category: "machine-owned"},
			},
		},
		Dependencies: posture.DependencyHealth{
			Score:      75,
			Ecosystems: []posture.EcosystemStatus{{Name: "go", Detected: true, LockFile: "valid"}},
			Totals:     posture.VulnSeverityCounts{High: 2},
		},
		Drift: drift.Report{
			Categories:    []drift.Category{},
			TotalFindings: 0,
			BySeverity:    make(map[drift.Severity]int),
		},
		Tools:      []posture.ToolStatus{{Name: "semgrep", Enabled: true, Available: true}},
		Ecosystems: []posture.EcosystemStatus{{Name: "go", Detected: true}},
	}

	data, err := RenderJSON(report)
	if err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}

	var decoded posture.PostureReport
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
