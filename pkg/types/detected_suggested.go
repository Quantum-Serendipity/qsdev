package types

import (
	"slices"
	"strings"
)

// WithSuggested returns lc completed from the detected module's suggested
// configuration (Suggested[lc.Name]): an empty Version or PackageManager is
// filled in, and each suggested extra whose key lc does not already set is
// appended. Values lc already carries (from flags, a profile or the wizard)
// always win. lc's Extras slice is never modified in place.
func (d DetectedProject) WithSuggested(lc LanguageChoice) LanguageChoice {
	s, ok := d.Suggested[lc.Name]
	if !ok {
		return lc
	}
	if lc.Version == "" {
		lc.Version = s.Version
	}
	if lc.PackageManager == "" {
		lc.PackageManager = s.PackageManager
	}
	extras := slices.Clone(lc.Extras)
	for _, e := range s.Extras {
		key := extraKey(e)
		if !slices.ContainsFunc(extras, func(x string) bool { return extraKey(x) == key }) {
			extras = append(extras, e)
		}
	}
	lc.Extras = extras
	return lc
}

// extraKey returns the key of a LanguageChoice extra ("key=value" or a bare
// "key" flag).
func extraKey(extra string) string {
	key, _, _ := strings.Cut(extra, "=")
	return key
}
