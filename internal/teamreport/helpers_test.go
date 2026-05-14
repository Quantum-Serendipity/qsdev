package teamreport

import (
	"testing"
	"time"
)

func TestMedianFloat64(t *testing.T) {
	tests := []struct {
		name   string
		sorted []float64
		want   float64
	}{
		{"empty", nil, 0},
		{"single", []float64{42.0}, 42.0},
		{"odd", []float64{10, 20, 30}, 20.0},
		{"even", []float64{10, 20, 30, 40}, 25.0},
		{"two", []float64{60, 80}, 70.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := medianFloat64(tt.sorted)
			if got != tt.want {
				t.Errorf("medianFloat64(%v) = %v, want %v", tt.sorted, got, tt.want)
			}
		})
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{"just now", now.Add(-10 * time.Second), "just now"},
		{"minutes", now.Add(-5 * time.Minute), "5 min ago"},
		{"1 min", now.Add(-1 * time.Minute), "1 min ago"},
		{"hours", now.Add(-3 * time.Hour), "3h ago"},
		{"1 hour", now.Add(-1 * time.Hour), "1h ago"},
		{"days", now.Add(-5 * 24 * time.Hour), "5d ago"},
		{"1 day", now.Add(-1 * 24 * time.Hour), "1d ago"},
		{"months", now.Add(-45 * 24 * time.Hour), "1mo ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relativeTime(tt.t)
			if got != tt.want {
				t.Errorf("relativeTime() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRelativeTimeFuture(t *testing.T) {
	future := time.Now().UTC().Add(1 * time.Hour)
	got := relativeTime(future)
	if got != "in the future" {
		t.Errorf("expected 'in the future', got %q", got)
	}
}

func TestScoreToGrade(t *testing.T) {
	tests := []struct {
		score float64
		want  string
	}{
		{97, "A+"},
		{93, "A"},
		{90, "A-"},
		{87, "B+"},
		{83, "B"},
		{80, "B-"},
		{70, "C-"},
		{50, "F"},
	}

	for _, tt := range tests {
		got := scoreToGrade(tt.score)
		if got != tt.want {
			t.Errorf("scoreToGrade(%.1f) = %q, want %q", tt.score, got, tt.want)
		}
	}
}

func TestRoundTo1(t *testing.T) {
	tests := []struct {
		input float64
		want  float64
	}{
		{1.0, 1.0},
		{1.15, 1.2},
		{1.14, 1.1},
		{75.555, 75.6},
	}

	for _, tt := range tests {
		got := roundTo1(tt.input)
		if got != tt.want {
			t.Errorf("roundTo1(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestSortProjectsByScoreDesc(t *testing.T) {
	projects := []ProjectSummary{
		projectSummaryHelper("low", 50, true, true, 0, 0, "v1.0.0", time.Now()),
		projectSummaryHelper("high", 90, true, true, 0, 0, "v1.0.0", time.Now()),
		projectSummaryHelper("mid", 70, true, true, 0, 0, "v1.0.0", time.Now()),
	}

	sortProjectsByScoreDesc(projects)

	if projects[0].Name != "high" {
		t.Errorf("expected first project to be 'high', got %q", projects[0].Name)
	}
	if projects[1].Name != "mid" {
		t.Errorf("expected second project to be 'mid', got %q", projects[1].Name)
	}
	if projects[2].Name != "low" {
		t.Errorf("expected third project to be 'low', got %q", projects[2].Name)
	}
}
