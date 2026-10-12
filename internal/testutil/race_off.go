//go:build !race

package testutil

// RaceScale widens time budgets in tests under the race detector; see
// race_on.go.
const RaceScale = 1
