package ecosystem

import (
	"testing"
	"time"
)

// TestReleaseAge_FloorAndDefault checks the D18 rule: a package manager
// enforces max(compliance window, its historical floor), and an unset window
// means the catalog baseline (DefaultMinReleaseAge).
func TestReleaseAge_FloorAndDefault(t *testing.T) {
	t.Parallel()
	const day = 24 * time.Hour
	tests := []struct {
		name  string
		age   time.Duration
		floor time.Duration
		want  time.Duration
	}{
		{"unset no floor", 0, 0, DefaultMinReleaseAge},
		{"unset week floor", 0, WeekMinReleaseAgeFloor, 7 * day},
		{"baseline no floor", 3 * day, 0, 3 * day},
		{"baseline week floor", 3 * day, WeekMinReleaseAgeFloor, 7 * day},
		{"enhanced week floor", 7 * day, WeekMinReleaseAgeFloor, 7 * day},
		{"strict no floor", 14 * day, 0, 14 * day},
		{"strict week floor", 14 * day, WeekMinReleaseAgeFloor, 14 * day},
		{"below default stays", time.Hour, 0, time.Hour},
		{"overlay below default floor", day, DefaultMinReleaseAge, 3 * day},
		{"strict default floor", 14 * day, DefaultMinReleaseAge, 14 * day},
		{"negative is unset", -time.Hour, 0, DefaultMinReleaseAge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := EffectiveReleaseAge(tt.age, tt.floor); got != tt.want {
				t.Errorf("EffectiveReleaseAge(%v, %v) = %v, want %v", tt.age, tt.floor, got, tt.want)
			}
			if got := (ModuleConfig{MinReleaseAge: tt.age}).ReleaseAge(tt.floor); got != tt.want {
				t.Errorf("ReleaseAge(%v) with MinReleaseAge %v = %v, want %v", tt.floor, tt.age, got, tt.want)
			}
		})
	}
}

// TestReleaseAgeDays_RoundsUp checks day conversion never loosens a gate:
// a partial day rounds up, and anything positive is at least one day.
func TestReleaseAgeDays_RoundsUp(t *testing.T) {
	t.Parallel()
	tests := []struct {
		d    time.Duration
		want int
	}{
		{0, 1},
		{-time.Hour, 1},
		{time.Minute, 1},
		{24 * time.Hour, 1},
		{25 * time.Hour, 2},
		{72 * time.Hour, 3},
		{168 * time.Hour, 7},
		{336 * time.Hour, 14},
		{336*time.Hour + time.Nanosecond, 15},
	}
	for _, tt := range tests {
		if got := ReleaseAgeDays(tt.d); got != tt.want {
			t.Errorf("ReleaseAgeDays(%v) = %d, want %d", tt.d, got, tt.want)
		}
	}
}
