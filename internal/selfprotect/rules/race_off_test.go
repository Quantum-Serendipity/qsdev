//go:build !race

package rules

// raceScale widens wall-clock test budgets under the race detector.
const raceScale = 1
