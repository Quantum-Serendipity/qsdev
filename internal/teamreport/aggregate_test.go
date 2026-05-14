package teamreport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/posture"
)

func makeReport(name string, score float64, baselinePass, enhancedPass bool, critVulns, highVulns int, gdevVersion string, generatedAt time.Time) *posture.PostureReport {
	return &posture.PostureReport{
		SchemaVersion: posture.SchemaVersion,
		GeneratedAt:   generatedAt,
		GdevVersion:   gdevVersion,
		ProjectName:   name,
		Score: posture.AggregateScore{
			Total:     score,
			Grade:     posture.ScoreToGrade(score),
			Defense:   score,
			Config:    score,
			DepHealth: score,
		},
		Conformance: posture.ConformanceResult{
			Baseline: posture.ConformanceLevel{
				Pass:   baselinePass,
				Checks: []posture.ConformanceCheck{},
			},
			Enhanced: posture.ConformanceLevel{
				Pass:   enhancedPass,
				Checks: []posture.ConformanceCheck{},
			},
		},
		Dependencies: posture.DependencyHealth{
			Totals: posture.VulnSeverityCounts{
				Critical: critVulns,
				High:     highVulns,
			},
			Ecosystems: []posture.EcosystemStatus{},
		},
		Defense: posture.DefenseCoverage{
			Layers: []posture.DefenseLayer{},
		},
		Config: posture.ConfigHealth{
			Files: []posture.ConfigFileInfo{},
		},
		Drift: posture.DriftReport{
			Categories: []posture.DriftCategory{},
			BySeverity: make(map[posture.DriftSeverity]int),
		},
		Tools:      []posture.ToolStatus{},
		Ecosystems: []posture.EcosystemStatus{},
	}
}

func TestAggregateThreeReports(t *testing.T) {
	now := time.Now().UTC()
	reports := []*posture.PostureReport{
		makeReport("alpha", 90.0, true, true, 0, 0, "v1.0.0", now),
		makeReport("beta", 70.0, true, false, 0, 1, "v1.0.0", now),
		makeReport("gamma", 50.0, false, false, 2, 3, "v1.0.0", now),
	}

	result, err := Aggregate(reports, AggregateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Summary.ProjectCount != 3 {
		t.Errorf("expected 3 projects, got %d", result.Summary.ProjectCount)
	}

	// Average: (90 + 70 + 50) / 3 = 70.0
	if result.Summary.AverageScore != 70.0 {
		t.Errorf("expected average 70.0, got %.1f", result.Summary.AverageScore)
	}

	// Median of sorted [50, 70, 90] = 70.0
	if result.Summary.MedianScore != 70.0 {
		t.Errorf("expected median 70.0, got %.1f", result.Summary.MedianScore)
	}

	if len(result.Projects) != 3 {
		t.Errorf("expected 3 project summaries, got %d", len(result.Projects))
	}
}

func TestAggregateSingleReport(t *testing.T) {
	now := time.Now().UTC()
	reports := []*posture.PostureReport{
		makeReport("solo", 85.0, true, true, 0, 0, "v1.0.0", now),
	}

	result, err := Aggregate(reports, AggregateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Summary.ProjectCount != 1 {
		t.Errorf("expected 1 project, got %d", result.Summary.ProjectCount)
	}

	if result.Summary.AverageScore != 85.0 {
		t.Errorf("expected average 85.0, got %.1f", result.Summary.AverageScore)
	}

	if result.Summary.MedianScore != 85.0 {
		t.Errorf("expected median 85.0, got %.1f", result.Summary.MedianScore)
	}
}

func TestAggregateEmptyInput(t *testing.T) {
	_, err := Aggregate(nil, AggregateOptions{})
	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
}

func TestAggregatePassRates(t *testing.T) {
	now := time.Now().UTC()
	reports := []*posture.PostureReport{
		makeReport("a", 80, true, true, 0, 0, "v1.0.0", now),
		makeReport("b", 60, true, false, 0, 0, "v1.0.0", now),
		makeReport("c", 40, false, false, 0, 0, "v1.0.0", now),
		makeReport("d", 70, true, true, 0, 0, "v1.0.0", now),
	}

	result, err := Aggregate(reports, AggregateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 3/4 baseline pass = 75.0%
	if result.Summary.BaselinePassRate != 75.0 {
		t.Errorf("expected baseline pass rate 75.0%%, got %.1f%%", result.Summary.BaselinePassRate)
	}

	// 2/4 enhanced pass = 50.0%
	if result.Summary.EnhancedPassRate != 50.0 {
		t.Errorf("expected enhanced pass rate 50.0%%, got %.1f%%", result.Summary.EnhancedPassRate)
	}
}

func TestAggregateVulnTotals(t *testing.T) {
	now := time.Now().UTC()
	reports := []*posture.PostureReport{
		makeReport("a", 80, true, true, 1, 2, "v1.0.0", now),
		makeReport("b", 60, true, false, 3, 4, "v1.0.0", now),
	}

	result, err := Aggregate(reports, AggregateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Summary.TotalCriticalVulns != 4 {
		t.Errorf("expected 4 critical vulns, got %d", result.Summary.TotalCriticalVulns)
	}

	if result.Summary.TotalHighVulns != 6 {
		t.Errorf("expected 6 high vulns, got %d", result.Summary.TotalHighVulns)
	}
}

func TestAggregateStaleDetection(t *testing.T) {
	staleTime := time.Now().UTC().Add(-8 * 24 * time.Hour) // 8 days ago
	freshTime := time.Now().UTC().Add(-1 * time.Hour)       // 1 hour ago

	reports := []*posture.PostureReport{
		makeReport("stale-project", 80, true, true, 0, 0, "v1.0.0", staleTime),
		makeReport("fresh-project", 90, true, true, 0, 0, "v1.0.0", freshTime),
	}

	result, err := Aggregate(reports, AggregateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var staleCount int
	for _, p := range result.Projects {
		if p.Stale {
			staleCount++
		}
	}

	if staleCount != 1 {
		t.Errorf("expected 1 stale project, got %d", staleCount)
	}
}

func TestAggregateOutdatedGdev(t *testing.T) {
	now := time.Now().UTC()
	reports := []*posture.PostureReport{
		makeReport("old-project", 80, true, true, 0, 0, "v1.0.0", now),
		makeReport("current-project", 90, true, true, 0, 0, "v1.5.0", now),
	}

	result, err := Aggregate(reports, AggregateOptions{
		GdevVersion: "v1.5.0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Summary.ProjectsNeedUpdate != 1 {
		t.Errorf("expected 1 project needing update, got %d", result.Summary.ProjectsNeedUpdate)
	}
}

func TestLoadPostureReports(t *testing.T) {
	dir := t.TempDir()

	// Write a valid report.
	r := makeReport("test", 85, true, true, 0, 0, "v1.0.0", time.Now().UTC())
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "test.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	// Write an invalid JSON file.
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{invalid}"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a non-JSON file (should be skipped).
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	reports, warnings, err := LoadPostureReports(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(reports) != 1 {
		t.Errorf("expected 1 report, got %d", len(reports))
	}

	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
}

func TestLoadPostureReportsInvalidSchema(t *testing.T) {
	dir := t.TempDir()

	r := makeReport("test", 85, true, true, 0, 0, "v1.0.0", time.Now().UTC())
	r.SchemaVersion = "99.99.99" // unsupported
	data, _ := json.MarshalIndent(r, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "test.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	reports, warnings, err := LoadPostureReports(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(reports) != 0 {
		t.Errorf("expected 0 reports for bad schema, got %d", len(reports))
	}

	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(warnings))
	}
}

func TestLoadPostureReportsNotDirectory(t *testing.T) {
	f, err := os.CreateTemp("", "test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	_, _, err = LoadPostureReports(f.Name())
	if err == nil {
		t.Fatal("expected error for non-directory, got nil")
	}
}

func TestIsOutdatedGdev(t *testing.T) {
	tests := []struct {
		project, current string
		want             bool
	}{
		{"v1.0.0", "v1.5.0", true},
		{"v1.3.0", "v1.5.0", false},
		{"v1.4.0", "v1.5.0", false},
		{"v1.5.0", "v1.5.0", false},
		{"v0.9.0", "v1.0.0", true},
		{"", "v1.0.0", false},
		{"v1.0.0", "", false},
	}

	for _, tt := range tests {
		got := isOutdatedGdev(tt.project, tt.current)
		if got != tt.want {
			t.Errorf("isOutdatedGdev(%q, %q) = %v, want %v", tt.project, tt.current, got, tt.want)
		}
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input      string
		wantMajor  int
		wantMinor  int
	}{
		{"v1.5.0", 1, 5},
		{"1.5.0", 1, 5},
		{"v0.9.3", 0, 9},
		{"bad", 0, 0},
		{"", 0, 0},
	}

	for _, tt := range tests {
		maj, min := parseVersion(tt.input)
		if maj != tt.wantMajor || min != tt.wantMinor {
			t.Errorf("parseVersion(%q) = (%d, %d), want (%d, %d)",
				tt.input, maj, min, tt.wantMajor, tt.wantMinor)
		}
	}
}
