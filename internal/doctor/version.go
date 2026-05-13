package doctor

import (
	"strconv"
	"strings"
)

// CompareVersions compares two dot-separated version strings.
// It returns -1 if a < b, 0 if equal, 1 if a > b.
// Missing segments are treated as 0 (e.g. "3.11" == "3.11.0").
// Non-numeric segments compare as 0 for that segment.
func CompareVersions(a, b string) int {
	aParts := splitVersion(a)
	bParts := splitVersion(b)

	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}

	for i := 0; i < maxLen; i++ {
		av := 0
		bv := 0
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

// MeetsMinimum returns true if version >= minimum.
func MeetsMinimum(version, minimum string) bool {
	return CompareVersions(version, minimum) >= 0
}

// splitVersion splits a version string on "." and parses each segment
// as an integer. Non-numeric segments become 0.
func splitVersion(v string) []int {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ".")
	result := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			n = 0
		}
		result[i] = n
	}
	return result
}
