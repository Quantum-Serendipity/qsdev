package javascript

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// supportedNodeMajors lists the Node.js majors that have a working
// nodejs_<major> attribute in nixpkgs-unstable, which generated devenv.yaml
// files track, in ascending order. EOL majors are removed (nodejs_18) or
// turned into a throw (nodejs_20, odd majors such as nodejs_25), so
// resolveNodeMajor never emits an attribute outside this list.
var supportedNodeMajors = []int{22, 24, 26}

// defaultNodeMajor is the newest LTS line in supportedNodeMajors. It is used
// when the project states no Node.js version or asks for "lts/*".
const defaultNodeMajor = 24

// npmMinReleaseAgeVersion is the first npm release that knows the
// min-release-age setting the generated .npmrc relies on. Older npm (npm 10,
// bundled with Node.js 22) reads the key as an unknown string and ignores it.
const npmMinReleaseAgeVersion = "11.10.0"

// npmMinReleaseAgeNodeMajor is the oldest supported Node.js major whose npm
// is at least npmMinReleaseAgeVersion. npm projects on an older Node.js get
// the npm of this major instead of their own (see npmNixPackage).
const npmMinReleaseAgeNodeMajor = 24

// nodeLTSCodenames maps .nvmrc "lts/<codename>" aliases to their major.
var nodeLTSCodenames = map[string]int{
	"argon": 4, "boron": 6, "carbon": 8, "dubnium": 10, "erbium": 12,
	"fermium": 14, "gallium": 16, "hydrogen": 18, "iron": 20, "jod": 22,
	"krypton": 24,
}

// versionRe matches a (possibly partial or x-range) version such as "18",
// "18.x", "20.11.1" or "v22.3.0-rc.1".
var versionRe = regexp.MustCompile(`^v?(\d+)(?:\.(\d+|[xX*]))?(?:\.(\d+|[xX*]))?(?:[-+].*)?$`)

// comparatorOpRe splits a range comparator into its operator and version.
var comparatorOpRe = regexp.MustCompile(`^(>=|<=|>|<|=|\^|~)?\s*(.*)$`)

// wildcard is the parseVersion component value for an absent or x component.
const wildcard = -1

// parseVersion parses a version such as "20", "20.x" or "v20.11.1", ignoring
// any leading range operator. Absent or x-range minor/patch components are
// returned as wildcard.
func parseVersion(s string) (major, minor, patch int, ok bool) {
	m := comparatorOpRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, 0, 0, false
	}
	v := versionRe.FindStringSubmatch(m[2])
	if v == nil {
		return 0, 0, 0, false
	}
	component := func(s string) int {
		n, err := strconv.Atoi(s)
		if err != nil {
			return wildcard
		}
		return n
	}
	return component(v[1]), component(v[2]), component(v[3]), true
}

// majorRange is an inclusive range of acceptable Node.js majors.
type majorRange struct{ lo, hi int }

func (r majorRange) contains(major int) bool { return major >= r.lo && major <= r.hi }

// parseNodeRange converts an engines.node / .nvmrc constraint into the major
// ranges it accepts: one per "||" alternative, each the intersection of its
// comparators. Only majors are tracked; a comparator such as "<20.5" keeps
// major 20 acceptable. ok is false when any part is unparseable.
func parseNodeRange(spec string) (ranges []majorRange, ok bool) {
	for _, alt := range strings.Split(spec, "||") {
		r, ok := parseRangeAlternative(strings.TrimSpace(alt))
		if !ok {
			return nil, false
		}
		ranges = append(ranges, r)
	}
	return ranges, true
}

// operatorGapRe joins an operator to its version (">= 18" -> ">=18").
var operatorGapRe = regexp.MustCompile(`(>=|<=|>|<|=|\^|~)\s+`)

func parseRangeAlternative(alt string) (majorRange, bool) {
	r := majorRange{lo: 0, hi: math.MaxInt}
	if lo, hi, found := strings.Cut(alt, " - "); found {
		loMajor, _, _, okLo := parseVersion(lo)
		hiMajor, _, _, okHi := parseVersion(hi)
		return majorRange{lo: loMajor, hi: hiMajor}, okLo && okHi
	}
	for _, c := range strings.Fields(operatorGapRe.ReplaceAllString(alt, "$1")) {
		if c == "*" || strings.EqualFold(c, "x") {
			continue
		}
		op := comparatorOpRe.FindStringSubmatch(c)[1]
		major, minor, patch, ok := parseVersion(c)
		if !ok {
			return majorRange{}, false
		}
		// Whether the version is exactly <major>.0.0 (or a bare major).
		atMajorStart := minor <= 0 && patch <= 0
		switch op {
		case ">=":
			r.lo = max(r.lo, major)
		case ">":
			if minor == wildcard {
				// ">18" means ">=19.0.0".
				major++
			}
			r.lo = max(r.lo, major)
		case "<":
			if atMajorStart {
				major--
			}
			r.hi = min(r.hi, major)
		case "<=":
			r.hi = min(r.hi, major)
		default: // "", "=", "^", "~": all stay within one major.
			r.lo = max(r.lo, major)
			r.hi = min(r.hi, major)
		}
	}
	return r, r.lo <= r.hi
}

// resolveNodeMajor maps a Node.js version constraint (engines.node or an
// .nvmrc value) to a major in supportedNodeMajors: defaultNodeMajor (the
// newest LTS) when the constraint accepts it, so an open range such as ">=18"
// does not land on a non-LTS "Current" release, otherwise the newest
// supported major the constraint accepts. When none is accepted it falls back to the oldest
// supported major at or above the requested one (or the newest supported
// major), and note explains the substitution so it is never silent.
func resolveNodeMajor(spec string) (major int, note string) {
	spec = strings.TrimSpace(spec)
	newest := supportedNodeMajors[len(supportedNodeMajors)-1]

	switch lower := strings.ToLower(spec); lower {
	case "":
		return defaultNodeMajor, ""
	case "node", "latest", "current", "stable":
		return newest, ""
	case "lts", "lts/*":
		return defaultNodeMajor, ""
	default:
		if codename, ok := strings.CutPrefix(lower, "lts/"); ok {
			m, known := nodeLTSCodenames[codename]
			if !known {
				return defaultNodeMajor, fmt.Sprintf("unrecognized Node.js alias %q; using nodejs_%d", spec, defaultNodeMajor)
			}
			spec = strconv.Itoa(m)
		}
	}

	ranges, ok := parseNodeRange(spec)
	if !ok {
		return defaultNodeMajor, fmt.Sprintf("unrecognized Node.js version %q; using nodejs_%d", spec, defaultNodeMajor)
	}
	accepts := func(major int) bool {
		return slices.ContainsFunc(ranges, func(r majorRange) bool { return r.contains(major) })
	}
	if accepts(defaultNodeMajor) {
		return defaultNodeMajor, ""
	}
	for _, v := range slices.Backward(supportedNodeMajors) {
		if accepts(v) {
			return v, ""
		}
	}

	lowest := math.MaxInt
	for _, r := range ranges {
		lowest = min(lowest, r.lo)
	}
	for _, v := range supportedNodeMajors {
		if v >= lowest {
			return v, fmt.Sprintf("Node.js %s is not packaged in nixpkgs (end of life); using nodejs_%d", spec, v)
		}
	}
	return newest, fmt.Sprintf("Node.js %s is newer than any packaged release; using nodejs_%d", spec, newest)
}

// nodeNixPackage returns the nixpkgs attribute for a supported Node.js major,
// including its bundled npm.
func nodeNixPackage(major int) string {
	return fmt.Sprintf("pkgs.nodejs_%d", major)
}

// nodeSlimNixPackage returns the nixpkgs attribute for a supported Node.js
// major without npm, for projects that take npm from npmNixPackage: the full
// nodejs_<major> would put its own bundled npm on PATH as well.
func nodeSlimNixPackage(major int) string {
	return fmt.Sprintf("pkgs.nodejs-slim_%d", major)
}

// npmNixPackage returns the npm for an npm project on Node.js major: the npm
// output of that major's nodejs-slim when it is new enough to enforce
// min-release-age, otherwise that of npmMinReleaseAgeNodeMajor. The npm output
// runs on its own Node.js, while the project's scripts use the project's.
func npmNixPackage(major int) string {
	return fmt.Sprintf("pkgs.nodejs-slim_%d.npm", max(major, npmMinReleaseAgeNodeMajor))
}
