//go:build race

package rules

// raceScale widens wall-clock test budgets under the race detector, which
// slows this code about four-fold.
const raceScale = 5
