package render

import (
	"os"

	"github.com/mattn/go-isatty"
)

// Indicator strings used when color output is enabled.
const (
	colorPass    = "\033[32m[✓]\033[0m" // green
	colorPartial = "\033[33m[~]\033[0m" // yellow
	colorSkip    = "\033[2m[ ]\033[0m"  // dim
	colorFail    = "\033[31m[✗]\033[0m" // red
)

// Indicator strings used when color output is disabled.
const (
	noColorPass    = "[OK]"
	noColorPartial = "[~~]"
	noColorSkip    = "[  ]"
	noColorFail    = "[!!]"
)

// ColorSupported returns true if the given file descriptor should receive
// colored output. It respects the NO_COLOR, FORCE_COLOR, and TERM
// environment variables, falling back to isatty detection.
func ColorSupported(fd uintptr) bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	if _, ok := os.LookupEnv("FORCE_COLOR"); ok {
		return true
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

// Indicators returns the four status indicator strings appropriate for the
// given color mode. The returned values are: pass, partial, skip, fail.
func Indicators(useColor bool) (pass, partial, skip, fail string) {
	if useColor {
		return colorPass, colorPartial, colorSkip, colorFail
	}
	return noColorPass, noColorPartial, noColorSkip, noColorFail
}

// PassIndicator returns the pass indicator for the given color mode.
func PassIndicator(useColor bool) string {
	if useColor {
		return colorPass
	}
	return noColorPass
}

// PartialIndicator returns the partial indicator for the given color mode.
func PartialIndicator(useColor bool) string {
	if useColor {
		return colorPartial
	}
	return noColorPartial
}

// SkipIndicator returns the skip indicator for the given color mode.
func SkipIndicator(useColor bool) string {
	if useColor {
		return colorSkip
	}
	return noColorSkip
}

// FailIndicator returns the fail indicator for the given color mode.
func FailIndicator(useColor bool) string {
	if useColor {
		return colorFail
	}
	return noColorFail
}
