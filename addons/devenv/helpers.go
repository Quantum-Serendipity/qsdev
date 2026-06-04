package devenv

import (
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/termutil"
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

// isAccessible delegates to termutil.IsAccessible.
func isAccessible() bool {
	return termutil.IsAccessible()
}
