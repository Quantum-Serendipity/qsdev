package termutil

import "os"

// IsAccessible returns true when the environment indicates that the user
// prefers an accessible (non-visual) interface: ACCESSIBLE is set, or TERM is
// "dumb".
//
// NO_COLOR is deliberately not a signal here. Per https://no-color.org/ it
// only asks programs not to emit ANSI color; lipgloss/termenv already honor
// it, so huh forms keep the full interactive TUI and render without color.
func IsAccessible() bool {
	if os.Getenv("ACCESSIBLE") != "" {
		return true
	}
	if os.Getenv("TERM") == "dumb" {
		return true
	}
	return false
}
