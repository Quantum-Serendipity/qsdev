package termutil

import "os"

// IsAccessible returns true when the environment indicates that the user
// prefers an accessible (non-visual) interface: ACCESSIBLE is set, NO_COLOR
// is set, or TERM is "dumb".
func IsAccessible() bool {
	if os.Getenv("ACCESSIBLE") != "" {
		return true
	}
	if os.Getenv("NO_COLOR") != "" {
		return true
	}
	if os.Getenv("TERM") == "dumb" {
		return true
	}
	return false
}
