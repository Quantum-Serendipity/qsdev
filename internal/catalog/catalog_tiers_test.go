package catalog

import (
	"testing"
	"time"
)

// TestEffectiveAgeGate pins the release-age window each compliance level
// enforces. An empty or unknown level must never fall below the lowest-order
// level (baseline): a typo in security.level cannot disable the gate.
func TestEffectiveAgeGate(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	tests := []struct {
		level string
		want  time.Duration
	}{
		{"baseline", 72 * time.Hour},
		{"enhanced", 168 * time.Hour},
		{"strict", 336 * time.Hour},
		{"", 72 * time.Hour},
		{"no-such-level", 72 * time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			t.Parallel()
			if got := cat.AgeGate(tt.level); got != tt.want {
				t.Errorf("AgeGate(%q) = %v, want %v", tt.level, got, tt.want)
			}
			if got := EffectiveAgeGate(tt.level); got != tt.want {
				t.Errorf("EffectiveAgeGate(%q) = %v, want %v", tt.level, got, tt.want)
			}
		})
	}
}

// TestAgeGate_EmptyCatalog checks a catalog without compliance levels yields
// zero, which consumers clamp to their own floor.
func TestAgeGate_EmptyCatalog(t *testing.T) {
	t.Parallel()
	if got := (&Catalog{}).AgeGate("strict"); got != 0 {
		t.Errorf("AgeGate on empty catalog = %v, want 0", got)
	}
}
