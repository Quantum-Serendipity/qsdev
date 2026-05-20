package render

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
)

func TestRenderBadge_ScoreVariant(t *testing.T) {
	report := &posture.PostureReport{
		Score: posture.AggregateScore{Total: 85, Grade: "B"},
		Conformance: posture.ConformanceResult{
			Baseline: posture.ConformanceLevel{Pass: true},
			Enhanced: posture.ConformanceLevel{Pass: false},
		},
		Defense: posture.DefenseCoverage{Enabled: 7, Total: 10, Score: 85},
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

func TestRenderBadge_TierVariant(t *testing.T) {
	tests := []struct {
		name      string
		tier      posture.ReportTierInfo
		wantMsg   string
		wantColor string
	}{
		{
			name:      "supply-chain-only",
			tier:      posture.ReportTierInfo{Current: "supply-chain-only", Position: 1, Total: 3, NextTier: "standard"},
			wantMsg:   "supply-chain-only",
			wantColor: "yellow",
		},
		{
			name:      "standard",
			tier:      posture.ReportTierInfo{Current: "standard", Position: 2, Total: 3, NextTier: "full"},
			wantMsg:   "standard",
			wantColor: "blue",
		},
		{
			name:      "full",
			tier:      posture.ReportTierInfo{Current: "full", Position: 3, Total: 3},
			wantMsg:   "full",
			wantColor: "brightgreen",
		},
		{
			name:      "empty tier",
			tier:      posture.ReportTierInfo{},
			wantMsg:   "unknown",
			wantColor: "lightgrey",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &posture.PostureReport{Tier: tt.tier}
			data, err := RenderBadge(report, "tier")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var badge BadgeJSON
			if err := json.Unmarshal(data, &badge); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}

			if badge.Label != "tier" {
				t.Errorf("label = %q, want %q", badge.Label, "tier")
			}
			if badge.Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", badge.Message, tt.wantMsg)
			}
			if badge.Color != tt.wantColor {
				t.Errorf("color = %q, want %q", badge.Color, tt.wantColor)
			}
		})
	}
}

func TestRenderBadge_UnknownVariantError(t *testing.T) {
	report := &posture.PostureReport{}

	_, err := RenderBadge(report, "unknown")
	if err == nil {
		t.Error("expected error for unknown variant")
	}
}

func TestRenderBadge_TrailingNewline(t *testing.T) {
	report := &posture.PostureReport{Score: posture.AggregateScore{Total: 50, Grade: "F"}}

	data, err := RenderBadge(report, "score")
	if err != nil {
		t.Fatal(err)
	}

	if len(data) == 0 || data[len(data)-1] != '\n' {
		t.Error("expected trailing newline in badge output")
	}
}

func TestRenderAllBadges(t *testing.T) {
	report := &posture.PostureReport{
		Score: posture.AggregateScore{Total: 85, Grade: "B"},
		Conformance: posture.ConformanceResult{
			Baseline: posture.ConformanceLevel{Pass: true},
			Enhanced: posture.ConformanceLevel{Pass: false},
		},
		Defense: posture.DefenseCoverage{Enabled: 7, Total: 10, Score: 85},
	}

	dir := t.TempDir()
	if err := RenderAllBadges(report, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, name := range []string{"badge-score.json", "badge-conformance.json", "badge-defense.json", "badge-tier.json"} {
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
