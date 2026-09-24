package python

import (
	"bufio"
	"cmp"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// latestKnownMinor is the newest CPython 3.x minor release that
// nixpkgs-python is known to provide. Specifiers that name a newer minor
// extend the candidate range, so this only bounds open-ended upper ranges.
const latestKnownMinor = 14

// oldestCandidateMinor is the oldest CPython 3.x minor considered when a
// specifier does not name a minor itself (e.g. ">=3" or "<3.11").
const oldestCandidateMinor = 8

// specOperatorSpaceRe matches whitespace between a specifier operator and its
// version (PEP 440 allows ">= 3.10"), which is removed before tokenising.
var specOperatorSpaceRe = regexp.MustCompile(`(===|==|!=|~=|<=|>=|<|>|\^|~|=)\s+`)

// specClauseRe parses one specifier clause: an optional PEP 440 or Poetry
// operator, a numeric release, an optional ".*" wildcard, and any
// pre/post/dev suffix (ignored).
var specClauseRe = regexp.MustCompile(`^(===|==|!=|~=|<=|>=|<|>|\^|~|=)?v?(\d+(?:\.\d+){0,2})(\.\*)?`)

// minorVersion is a CPython major.minor pair.
type minorVersion struct{ major, minor int }

func (v minorVersion) less(o minorVersion) bool {
	return v.major < o.major || (v.major == o.major && v.minor < o.minor)
}

func (v minorVersion) String() string {
	return strconv.Itoa(v.major) + "." + strconv.Itoa(v.minor)
}

// specClause is one parsed version clause, e.g. ">=3.10" or "!=3.9.*".
type specClause struct {
	op       string
	parts    []int // release components as written (1 to 3)
	wildcard bool
	release  string // the release as written, e.g. "3.11.4"
}

func (c specClause) minor() minorVersion {
	v := minorVersion{major: c.parts[0]}
	if len(c.parts) > 1 {
		v.minor = c.parts[1]
	}
	return v
}

// isLowerBound reports whether the clause bounds versions from below at a
// named minor. A major-only clause (">=3", "^3", "==3.*") names no minor, so
// it does not make the oldest candidate minor the project's lower bound.
func (c specClause) isLowerBound() bool {
	return len(c.parts) > 1 && c.op != "<" && c.op != "<=" && c.op != "!="
}

// isExact reports whether the clause pins one full release, e.g. "==3.11.4".
func (c specClause) isExact() bool {
	switch c.op {
	case "", "=", "==", "===":
		return len(c.parts) == 3 && !c.wildcard
	}
	return false
}

// allows reports whether some release of CPython minor m satisfies c. Upper
// bounds that name a patch release exclude the whole minor, because
// nixpkgs-python resolves a MAJOR.MINOR version to its newest patch.
func (c specClause) allows(m minorVersion) bool {
	v := c.minor()
	sameMajor := m.major == v.major
	sameMinor := m == v || (len(c.parts) == 1 && sameMajor)
	switch c.op {
	case ">=", ">":
		return !m.less(v)
	case "<=":
		if len(c.parts) == 3 {
			return m.less(v)
		}
		return !v.less(m)
	case "<":
		return m.less(v)
	case "!=":
		return !c.wildcard || !sameMinor
	case "~=":
		if len(c.parts) >= 3 {
			return m == v
		}
		return sameMajor && !m.less(v)
	case "^":
		return sameMajor && !m.less(v)
	default: // "", "=", "==", "===", "~" (Poetry tilde pins the minor)
		return sameMinor
	}
}

// parseSpecifier parses a PEP 440 specifier set or a Poetry constraint into
// its alternatives ("||"-separated), each a list of clauses that must all
// hold. It reports false when any clause is unparseable. A bare "*" (any
// version) yields an alternative with no clauses.
func parseSpecifier(spec string) ([][]specClause, bool) {
	var alternatives [][]specClause
	for _, alt := range strings.Split(spec, "||") {
		alt = specOperatorSpaceRe.ReplaceAllString(alt, "$1")
		var clauses []specClause
		for _, tok := range strings.FieldsFunc(alt, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' }) {
			if tok == "*" {
				continue
			}
			m := specClauseRe.FindStringSubmatch(tok)
			if m == nil {
				return nil, false
			}
			var parts []int
			for _, p := range strings.Split(m[2], ".") {
				n, err := strconv.Atoi(p)
				if err != nil {
					return nil, false
				}
				parts = append(parts, n)
			}
			clauses = append(clauses, specClause{op: m[1], parts: parts, wildcard: m[3] != "", release: m[2]})
		}
		alternatives = append(alternatives, clauses)
	}
	return alternatives, len(alternatives) > 0
}

// versionForSpecifier chooses the Python version to pin for a
// requires-python specifier or Poetry python constraint. Among the CPython
// minors the specifier allows, it picks the lowest when every alternative has
// a lower bound, and otherwise the default version (or failing that the
// highest allowed minor below it). An exact pin such as "==3.11.4" is
// returned as written. It returns "" when the specifier is empty,
// unparseable or unsatisfiable.
func versionForSpecifier(spec string) string {
	alternatives, ok := parseSpecifier(strings.TrimSpace(spec))
	if !ok {
		return ""
	}
	hasLower := true
	var allowed []minorVersion
	for _, clauses := range alternatives {
		if len(clauses) == 0 {
			return "" // "*": no constraint to derive a version from
		}
		hasLower = hasLower && slices.ContainsFunc(clauses, specClause.isLowerBound)
		for _, m := range candidateMinors(clauses) {
			if allowsAll(clauses, m) && !slices.Contains(allowed, m) {
				allowed = append(allowed, m)
			}
		}
	}
	if len(allowed) == 0 {
		return ""
	}
	// A project that still admits Python 2 (">=2.7,!=3.0.*", "^2.7 || ^3.6")
	// is developed on Python 3 whenever the specifier allows it.
	def := parseMinor(defaultPythonVersion)
	if slices.ContainsFunc(allowed, func(m minorVersion) bool { return m.major == def.major }) {
		allowed = slices.DeleteFunc(allowed, func(m minorVersion) bool { return m.major != def.major })
	}
	slices.SortFunc(allowed, compareMinor)

	chosen := allowed[0]
	if !hasLower {
		for _, m := range allowed {
			if !def.less(m) {
				chosen = m // highest allowed minor at or below the default
			}
		}
	}
	for _, clauses := range alternatives {
		for _, c := range clauses {
			if c.isExact() && c.minor() == chosen {
				return c.release
			}
		}
	}
	return chosen.String()
}

// candidateMinors returns the CPython minors worth testing against clauses:
// every 3.x minor from oldestCandidateMinor to the newest known or named
// minor, plus any older minor a clause names explicitly.
func candidateMinors(clauses []specClause) []minorVersion {
	highest := latestKnownMinor
	var candidates []minorVersion
	for _, c := range clauses {
		m := c.minor()
		if m.major == 3 && m.minor > highest {
			highest = m.minor
		}
		if len(c.parts) > 1 && m.less(minorVersion{major: 3, minor: oldestCandidateMinor}) {
			candidates = append(candidates, m)
		}
	}
	for minor := oldestCandidateMinor; minor <= highest; minor++ {
		candidates = append(candidates, minorVersion{major: 3, minor: minor})
	}
	return candidates
}

// allowsAll reports whether every clause allows some release of m.
func allowsAll(clauses []specClause, m minorVersion) bool {
	for _, c := range clauses {
		if !c.allows(m) {
			return false
		}
	}
	return true
}

func compareMinor(a, b minorVersion) int {
	return cmp.Or(cmp.Compare(a.major, b.major), cmp.Compare(a.minor, b.minor))
}

// parseMinor parses the MAJOR.MINOR prefix of a version; it is only used on
// trusted constants.
func parseMinor(version string) minorVersion {
	major, rest, _ := strings.Cut(version, ".")
	minor, _, _ := strings.Cut(rest, ".")
	ma, _ := strconv.Atoi(major)
	mi, _ := strconv.Atoi(minor)
	return minorVersion{major: ma, minor: mi}
}

// readPythonVersionFile returns the version requested by a .python-version
// file: the first line that is neither blank nor a "#" comment (uv and pyenv
// ignore those), with a uv "cpython-" / "cpython@" prefix and any platform
// suffix (cpython-3.12.4-linux-x86_64-gnu) removed. The raw request is
// returned alongside so callers can report values they reject.
func readPythonVersionFile(path string) (version, raw string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close() //nolint:errcheck // best-effort read

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		raw = strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		v := raw
		for _, prefix := range []string{"cpython-", "cpython@"} {
			if rest, ok := strings.CutPrefix(v, prefix); ok {
				v, _, _ = strings.Cut(rest, "-")
			}
		}
		return v, raw
	}
	return "", ""
}
