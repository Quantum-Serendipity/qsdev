package posture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRenderBadge_ScoreVariant(t *testing.T) {
	report := &PostureReport{
		Score: AggregateScore{Total: 85, Grade: "B"},
		Conformance: ConformanceResult{
			Baseline: ConformanceLevel{Pass: true},
			Enhanced: ConformanceLevel{Pass: false},
		},
		Defense: DefenseCoverage{Enabled: 7, Total: 10, Score: 85},
	}

	data, err := RenderBadge(report, "score")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var badge BadgeJSON
	if err := json.Unmarshal(data, &badge); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if badge.SchemaVersion != 1 {
		t.Errorf("schemaVersion = %d, want 1", badge.SchemaVersion)
	}
	if badge.Label != "security posture" {
		t.Errorf("label = %q, want %q", badge.Label, "security posture")
	}
	if badge.Message != "85/100 B" {
		t.Errorf("message = %q, want %q", badge.Message, "85/100 B")
	}
	if badge.Color != "green" {
		t.Errorf("color = %q, want %q", badge.Color, "green")
	}
}

func TestRenderBadge_ConformanceVariant(t *testing.T) {
	tests := []struct {
		name     string
		baseline bool
		enhanced bool
		wantMsg  string
		wantClr  string
	}{
		{"enhanced pass", true, true, "enhanced PASS", "brightgreen"},
		{"baseline only", true, false, "baseline PASS", "green"},
		{"baseline fail", false, false, "baseline FAIL", "red"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &PostureReport{
				Conformance: ConformanceResult{
					Baseline: ConformanceLevel{Pass: tt.baseline},
					Enhanced: ConformanceLevel{Pass: tt.enhanced},
				},
			}
			data, err := RenderBadge(report, "conformance")
			if err != nil {
				t.Fatal(err)
			}

			var badge BadgeJSON
			if err := json.Unmarshal(data, &badge); err != nil {
				t.Fatal(err)
			}

			if badge.Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", badge.Message, tt.wantMsg)
			}
			if badge.Color != tt.wantClr {
				t.Errorf("color = %q, want %q", badge.Color, tt.wantClr)
			}
		})
	}
}

func TestRenderBadge_DefenseVariant(t *testing.T) {
	report := &PostureReport{
		Defense: DefenseCoverage{Enabled: 8, Total: 10, Score: 90},
	}

	data, err := RenderBadge(report, "defense")
	if err != nil {
		t.Fatal(err)
	}

	var badge BadgeJSON
	if err := json.Unmarshal(data, &badge); err != nil {
		t.Fatal(err)
	}

	if badge.Label != "defense coverage" {
		t.Errorf("label = %q, want %q", badge.Label, "defense coverage")
	}
	if badge.Message != "8/10 layers" {
		t.Errorf("message = %q, want %q", badge.Message, "8/10 layers")
	}
	if badge.Color != "brightgreen" {
		t.Errorf("color = %q, want %q (score 90 => brightgreen)", badge.Color, "brightgreen")
	}
}

func TestRenderBadge_ColorBoundaryValues(t *testing.T) {
	tests := []struct {
		score float64
		color string
	}{
		{100, "brightgreen"},
		{90, "brightgreen"},
		{89.5, "brightgreen"}, // rounds to 90
		{89.4, "green"},       // rounds to 89
		{80, "green"},
		{79.5, "green"},  // rounds to 80
		{79.4, "yellow"}, // rounds to 79
		{70, "yellow"},
		{69.4, "orange"}, // rounds to 69
		{60, "orange"},
		{59.4, "red"}, // rounds to 59
		{0, "red"},
	}

	for _, tt := range tests {
		got := scoreColor(tt.score)
		if got != tt.color {
			t.Errorf("scoreColor(%v) = %q, want %q", tt.score, got, tt.color)
		}
	}
}

func TestRenderBadge_EmptyReport(t *testing.T) {
	report := &PostureReport{}

	// Empty score variant should not error.
	data, err := RenderBadge(report, "score")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var badge BadgeJSON
	if err := json.Unmarshal(data, &badge); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Score should be "0/100 "
	if badge.SchemaVersion != 1 {
		t.Errorf("schemaVersion = %d, want 1", badge.SchemaVersion)
	}
	if badge.Color != "red" {
		t.Errorf("color for zero score = %q, want %q", badge.Color, "red")
	}
}

func TestRenderBadge_UnknownVariantError(t *testing.T) {
	report := &PostureReport{}

	_, err := RenderBadge(report, "unknown")
	if err == nil {
		t.Error("expected error for unknown variant")
	}
}

func TestRenderBadge_DefaultVariantIsScore(t *testing.T) {
	report := &PostureReport{
		Score: AggregateScore{Total: 75, Grade: "C"},
	}

	data, err := RenderBadge(report, "")
	if err != nil {
		t.Fatal(err)
	}

	var badge BadgeJSON
	if err := json.Unmarshal(data, &badge); err != nil {
		t.Fatal(err)
	}

	if badge.Label != "security posture" {
		t.Errorf("default variant label = %q, want %q", badge.Label, "security posture")
	}
}

func TestRenderAllBadges(t *testing.T) {
	report := &PostureReport{
		Score: AggregateScore{Total: 85, Grade: "B"},
		Conformance: ConformanceResult{
			Baseline: ConformanceLevel{Pass: true},
			Enhanced: ConformanceLevel{Pass: false},
		},
		Defense: DefenseCoverage{Enabled: 7, Total: 10, Score: 85},
	}

	dir := t.TempDir()
	if err := RenderAllBadges(report, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, name := range []string{"badge-score.json", "badge-conformance.json", "badge-defense.json"} {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("missing badge file %s: %v", name, err)
			continue
		}
		var badge BadgeJSON
		if err := json.Unmarshal(data, &badge); err != nil {
			t.Errorf("invalid JSON in %s: %v", name, err)
		}
		if badge.SchemaVersion != 1 {
			t.Errorf("%s: schemaVersion = %d, want 1", name, badge.SchemaVersion)
		}
	}
}

func TestRenderBadge_TrailingNewline(t *testing.T) {
	report := &PostureReport{Score: AggregateScore{Total: 50, Grade: "F"}}

	data, err := RenderBadge(report, "score")
	if err != nil {
		t.Fatal(err)
	}

	if len(data) == 0 || data[len(data)-1] != '\n' {
		t.Error("expected trailing newline in badge output")
	}
}
