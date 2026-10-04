//go:build !race

package devinit

// raceScale widens time budgets under the race detector; see race_on_test.go.
const raceScale = 1
