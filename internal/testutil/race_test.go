package testutil

import (
	"runtime/debug"
	"testing"
)

// TestRaceScale checks RaceScale against how the test binary was built: the
// -race build setting, not the build tag the constant itself keys on.
func TestRaceScale(t *testing.T) {
	t.Parallel()
	info, ok := debug.ReadBuildInfo()
	if !ok {
		t.Fatal("the test binary carries no build information")
	}
	race := false
	for _, s := range info.Settings {
		if s.Key == "-race" {
			race = s.Value == "true"
		}
	}
	want := 1
	if race {
		want = 5
	}
	if RaceScale != want {
		t.Errorf("RaceScale = %d with -race=%t, want %d", RaceScale, race, want)
	}
}
