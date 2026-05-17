package sliceutil

// Contains checks whether a string slice includes the given value.
func Contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

// Dedup returns a new slice with duplicates removed, preserving order.
func Dedup(items []string) []string {
	seen := make(map[string]bool, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// Remove returns a new slice with all occurrences of val removed.
func Remove(slice []string, val string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != val {
			result = append(result, s)
		}
	}
	return result
}
