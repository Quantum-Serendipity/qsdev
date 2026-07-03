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
			c, err := gocvss30.ParseVector(vector)
			if err != nil {
				return 0, false
			}
			return c.BaseScore(), true
		}
		c, err := gocvss31.ParseVector(vector)
		if err != nil {
			return 0, false
		}
		return c.BaseScore(), true
	case "CVSS_V2":
		c, err := gocvss20.ParseVector(vector)
		if err != nil {
			return 0, false
		}
		return c.BaseScore(), true
	default:
		return 0, false
	}
}

// bucketCVSSScore maps a CVSS base score onto a lowercase severity label that
// NormalizeSeverity recognizes, mirroring the go-cvss qualitative thresholds
// (>= 9.0 critical, >= 7.0 high, >= 4.0 moderate, >= 0.1 low). A 0.0/NONE score
// is a determined "no impact" result, so it maps to "info" — visible but not
// overstated — rather than a real "low" finding.
func bucketCVSSScore(score float64) string {
	switch {
	case score >= 9.0:
		return "critical"
	case score >= 7.0:
		return "high"
	case score >= 4.0:
		return "moderate"
	case score >= 0.1:
		return "low"
	default:
		return "info"
	}
}
