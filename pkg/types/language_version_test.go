package types

import "testing"

func TestNewerVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		a, b string
		want bool
	}{
		{"1.26.7", "1.26.3", true},
		{"1.26.3", "1.26.7", false},
		{"1.26.7", "1.26.7", false},
		{"1.26.0", "1.26", false},
		{"1.26", "1.26.0", false},
		{"1.27", "1.26.9", true},
		{"20", "18.19.1", true},
		{"1.10", "1.9", true},
		{"1.26rc1", "1.25", false},
		{"1.26", "1.26rc1", false},
		{">=3.10", "3.9", false},
		{"", "1.0", false},
		{"1.0", "", false},
		{"1..2", "1.1", false},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			t.Parallel()
			if got := NewerVersion(tt.a, tt.b); got != tt.want {
				t.Errorf("NewerVersion(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestRaiseVersionsToDetected checks a configured version is raised to the
// one the project's manifest requires, and never lowered.
func TestRaiseVersionsToDetected(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		detected string // go.mod go directive
		version  string // answers version
		want     string
	}{
		{"older pin raised", "1.26.7", "1.26.3", "1.26.7"},
		{"newer pin kept", "1.24", "1.26.7", "1.26.7"},
		{"equal pin kept", "1.26.7", "1.26.7", "1.26.7"},
		{"unpinned left alone", "1.26.7", "", ""},
		{"unorderable pin kept", "1.26.7", "1.26rc1", "1.26rc1"},
		{"nothing detected", "", "1.26.3", "1.26.3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := WizardAnswers{
				Detected:  DetectedProject{HasGoMod: true, GoVersion: tt.detected},
				Languages: []LanguageChoice{{Name: "go", Version: tt.version}, {Name: "rust", Version: "stable"}},
			}
			a.RaiseVersionsToDetected()
			if got := a.Languages[0].Version; got != tt.want {
				t.Errorf("go version = %q, want %q", got, tt.want)
			}
			if got := a.Languages[1].Version; got != "stable" {
				t.Errorf("rust version = %q, want it untouched", got)
			}
		})
	}
}
