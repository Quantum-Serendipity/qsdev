//go:build race

package testutil

// RaceScale widens time budgets in tests under the race detector, which
// slows instrumented code about four-fold. It is the one copy every package
// scales its time budgets by.
const RaceScale = 5
