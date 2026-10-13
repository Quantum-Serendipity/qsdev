package types

import (
	"strconv"
	"strings"
)

// NewerVersion reports whether a is a strictly newer version than b. Both
// must be plain dotted numeric versions ("1.26", "1.26.7", "20"); missing
// components count as zero, so "1.26" and "1.26.0" are equal. Any other form
// (a range, a channel, a pre-release such as "1.26rc1") is not comparable and
// yields false, so a caller never replaces a version it cannot order.
func NewerVersion(a, b string) bool {
	as, okA := numericVersion(a)
	bs, okB := numericVersion(b)
	if !okA || !okB {
		return false
	}
	for i := range max(len(as), len(bs)) {
		x, y := 0, 0
		if i < len(as) {
			x = as[i]
		}
		if i < len(bs) {
			y = bs[i]
		}
		if x != y {
			return x > y
		}
	}
	return false
}

// numericVersion splits a plain dotted numeric version into its components.
func numericVersion(v string) ([]int, bool) {
	if v == "" {
		return nil, false
	}
	parts := strings.Split(v, ".")
	nums := make([]int, len(parts))
	for i, p := range parts {
		if p == "" || strings.TrimLeft(p, "0123456789") != "" {
			return nil, false
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, false
		}
		nums[i] = n
	}
	return nums, true
}

// RaiseVersionsToDetected raises each configured language version to the
// version detection reports the project requires (go.mod's go directive, a
// Node.js or Python version file) when that is newer, so a regeneration never
// pins a toolchain older than the project's own manifest asks for. Versions
// are only ever raised, and only when both are plain numeric versions.
func (a *WizardAnswers) RaiseVersionsToDetected() {
	required := make(map[string]string)
	for _, lc := range a.Detected.LanguageChoices() {
		if lc.Version != "" {
			required[lc.Name] = lc.Version
		}
	}
	for i := range a.Languages {
		lang := &a.Languages[i]
		if want, ok := required[lang.Name]; ok && NewerVersion(want, lang.Version) {
			lang.Version = want
		}
	}
}
