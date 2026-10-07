package toolcheck

import (
	"strconv"
	"strings"
)

// CompareVersions compares two dot-separated version strings.
// It returns -1 if a < b, 0 if equal, 1 if a > b.
// Missing segments are treated as 0 (e.g. "3.11" == "3.11.0").
// Each segment compares by its leading digits, so a pre-release or build
// suffix is ignored ("2.4pre20211001" == "2.4", "2.1.2+rev" == "2.1.2"); a
// segment with no leading digit compares as 0.
func CompareVersions(a, b string) int {
	aParts := splitVersion(a)
	bParts := splitVersion(b)

	for i := range max(len(aParts), len(bParts)) {
		av, bv := 0, 0
		if i < len(aParts) {
			av = aParts[i]
		}
		if i < len(bParts) {
			bv = bParts[i]
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

// MeetsMinimum reports whether version >= minimum. An empty minimum is no
// floor, so any version meets it; an unknown (empty) version never meets a
// floor.
func MeetsMinimum(version, minimum string) bool {
	if minimum == "" {
		return true
	}
	if version == "" {
		return false
	}
	return CompareVersions(version, minimum) >= 0
}

// splitVersion splits a version string on "." and parses the leading digits
// of each segment as an integer.
func splitVersion(v string) []int {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ".")
	result := make([]int, len(parts))
	for i, p := range parts {
		end := 0
		for end < len(p) && p[end] >= '0' && p[end] <= '9' {
			end++
		}
		// Atoi fails only on an empty run (or overflow): either compares as 0.
		n, _ := strconv.Atoi(p[:end])
		result[i] = n
	}
	return result
}
