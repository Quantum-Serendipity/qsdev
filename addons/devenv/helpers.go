package devenv

import (
	"os"
	"strings"
)

// inputKeyFromURL derives an input name from a Nix flake URL by extracting
// the repository name.
func inputKeyFromURL(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return url
}

// isAccessible returns true when ACCESSIBLE or NO_COLOR env var is set.
func isAccessible() bool {
	if os.Getenv("ACCESSIBLE") != "" {
		return true
	}
	if os.Getenv("NO_COLOR") != "" {
		return true
	}
	return false
}
