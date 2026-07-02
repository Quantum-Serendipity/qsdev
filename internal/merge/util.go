package merge

// DeepMergeJSON merges src into dst and returns a new map; the inputs are not
// modified. For conflicting keys src wins; when both values are objects it
// recurses, so unknown nested keys present only in dst survive. For everything
// else (including arrays) src overwrites dst.
func DeepMergeJSON(dst, src map[string]any) map[string]any {
	out := make(map[string]any, len(dst))
	for k, v := range dst {
		out[k] = v
	}
	for k, v := range src {
		if srcMap, ok := v.(map[string]any); ok {
			if dstMap, ok := out[k].(map[string]any); ok {
				out[k] = DeepMergeJSON(dstMap, srcMap)
				continue
			}
		}
		out[k] = v
	}
	return out
}

// unionStrings returns the union of two string slices, preserving order.
// Elements from a appear first, followed by elements from b not already in a.
func unionStrings(a, b []string) []string {
	seen := make(map[string]bool)
	var result []string
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
