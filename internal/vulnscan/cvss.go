package vulnscan

import (
	"strings"

	gocvss20 "github.com/pandatix/go-cvss/20"
	gocvss30 "github.com/pandatix/go-cvss/30"
	gocvss31 "github.com/pandatix/go-cvss/31"
	gocvss40 "github.com/pandatix/go-cvss/40"
)

// severityFromCVSS derives a coarse severity label from an OSV record's
// top-level severity[] array of CVSS vectors. Go advisories (GO-YYYY-NNNN), and
// many others, carry severity ONLY there — as CVSS vector strings — and never in
// database_specific.severity, so without this fallback their SeverityLabel is
// empty, NormalizeSeverity resolves them to "unknown" (fail-closed), and the
// audit exit gate is pinned non-zero for any project carrying a single advisory.
//
// It selects the highest-confidence vector by version preference
// (CVSS_V4 > CVSS_V3 > CVSS_V2), parses it with the matching go-cvss subpackage
// (dispatching CVSS v3.0 vs v3.1 on the vector's "CVSS:3.0"/"CVSS:3.1" prefix),
// computes the base score, and buckets it into a lowercase label NormalizeSeverity
// recognizes (see bucketCVSSScore). Within a preference tier the first entry
// that parses wins; if the best tier's vector is unparseable it falls through to
// the next tier. It returns "" when no entry parses, so an unparseable or absent
// vector stays "unknown" (fail-closed) rather than being demoted to a benign
// bucket. OSV severity "type" values are CVSS_V2/CVSS_V3/CVSS_V4.
func severityFromCVSS(entries ...osvSeverity) string {
	for _, want := range []string{"CVSS_V4", "CVSS_V3", "CVSS_V2"} {
		for _, e := range entries {
			if e.Type != want {
				continue
			}
			if score, ok := cvssBaseScore(e.Type, e.Score); ok {
				return bucketCVSSScore(score)
			}
		}
	}
	return ""
}

// cvssBaseScore parses a single CVSS vector with the subpackage matching its OSV
// type, dispatching CVSS v3.0 vs v3.1 on the vector prefix (defaulting to v3.1
// for any other 3.x), and returns the base score. v4 exposes its score via
// Score(); v2/v3 via BaseScore(). ok is false when the vector is empty or fails
// to parse, so callers fall through to the next-best entry (fail-closed).
func cvssBaseScore(typ, vector string) (float64, bool) {
	if strings.TrimSpace(vector) == "" {
		return 0, false
	}
	switch typ {
	case "CVSS_V4":
		c, err := gocvss40.ParseVector(vector)
		if err != nil {
			return 0, false
		}
		return c.Score(), true
	case "CVSS_V3":
		if strings.HasPrefix(vector, "CVSS:3.0") {
			return baseScore(gocvss30.ParseVector(vector))
		}
		return baseScore(gocvss31.ParseVector(vector))
	case "CVSS_V2":
		return baseScore(gocvss20.ParseVector(vector))
	default:
		return 0, false
	}
}

// baseScore adapts a go-cvss ParseVector result (v2/v3.x expose BaseScore) to
// the (score, ok) shape cvssBaseScore returns.
func baseScore(c interface{ BaseScore() float64 }, err error) (float64, bool) {
	if err != nil {
		return 0, false
	}
	return c.BaseScore(), true
}

// bucketCVSSScore maps a CVSS base score onto a lowercase severity label that
// NormalizeSeverity recognizes. The thresholds come from the library's own
// qualitative rating (>= 9.0 CRITICAL, >= 7.0 HIGH, >= 4.0 MEDIUM, >= 0.1 LOW;
// identical for v3.1 and v4.0, and applied to v2 scores for want of an official
// v2 scale) rather than a hand-mirrored table. Two labels are deliberately
// remapped: MEDIUM folds into this codebase's "moderate" vocabulary, and a
// 0.0/NONE score — a determined "no impact" result — maps to "info", visible
// but not overstated, rather than a real "low" finding.
func bucketCVSSScore(score float64) string {
	rating, err := gocvss31.Rating(score)
	if err != nil {
		// Unreachable for a parsed vector (scores are always 0.0-10.0); stay
		// fail-closed by resolving to "" -> "unknown" rather than guessing.
		return ""
	}
	switch rating {
	case "NONE":
		return "info"
	case "MEDIUM":
		return "moderate"
	default:
		return strings.ToLower(rating)
	}
}
