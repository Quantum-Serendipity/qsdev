//go:build race

package devinit

// raceScale widens time budgets under the race detector, which slows the
// selfprotect hook about four-fold.
const raceScale = 5
