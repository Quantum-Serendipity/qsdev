package ecosystem

import "time"

// DefaultMinReleaseAge is the release-age window of the catalog's baseline
// compliance level (age_gating_threshold_hours: 72). It applies when
// ModuleConfig.MinReleaseAge is unset, so a module never generates a config
// without an age gate.
const DefaultMinReleaseAge = 72 * time.Hour

// WeekMinReleaseAgeFloor is the historical floor of the package managers
// that shipped a 7-day gate (yarn, bun, uv): the compliance window raises
// their gate but never lowers it below this.
const WeekMinReleaseAgeFloor = 7 * 24 * time.Hour

// EffectiveReleaseAge returns the release-age gate a package manager
// enforces: the compliance window d (DefaultMinReleaseAge when d is not
// positive), raised to the manager's floor.
func EffectiveReleaseAge(d, floor time.Duration) time.Duration {
	if d <= 0 {
		d = DefaultMinReleaseAge
	}
	return max(d, floor)
}

// ReleaseAge returns the module's release-age gate for a package manager
// with the given floor (see EffectiveReleaseAge).
func (c ModuleConfig) ReleaseAge(floor time.Duration) time.Duration {
	return EffectiveReleaseAge(c.MinReleaseAge, floor)
}

// ReleaseAgeDays converts a release-age gate to whole days, rounding up so
// the conversion never loosens it, and never below one day.
func ReleaseAgeDays(d time.Duration) int {
	const day = 24 * time.Hour
	return max(1, int((d+day-1)/day))
}
