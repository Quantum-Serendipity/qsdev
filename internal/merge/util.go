package merge

// unionStrings returns the union of two string slices, preserving order.
// Elements from a appear first, followed by elements from b not already in a.
func unionStrings(a, b []string) []string {
	size := len(a) + len(b)
	if size < 0 {
		return nil
	}
	seen := make(map[string]bool, size)
	result := make([]string, 0, size)
	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// diffStrings returns elements in a that are not in b.
func diffStrings(a, b []string) []string {
	bSet := make(map[string]bool, len(b))
	for _, s := range b {
		bSet[s] = true
	}
	var result []string
	for _, s := range a {
		if !bSet[s] {
			result = append(result, s)
		}
	}
	return result
}
