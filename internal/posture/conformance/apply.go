package conformance

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
)

// PolicyFileCheck names the single failing check recorded when the policy
// file exists but cannot be loaded.
const PolicyFileCheck posture.CheckName = "policy-file"

// Evaluate evaluates every requirement of a custom policy against report and
// returns the resulting conformance level. The level passes only when every
// requirement passes. A requirement whose expression is invalid, or that reads
// dependency totals no conclusive scan produced, fails with the reason, so a
// typo or a skipped scan can never pass.
func Evaluate(custom *Custom, report *posture.PostureReport) *posture.ConformanceLevel {
	level := &posture.ConformanceLevel{
		Pass:   true,
		Checks: make([]posture.ConformanceCheck, 0, len(custom.Requirements)),
	}
	for _, req := range custom.Requirements {
		check := evaluateRequirement(req, report)
		if !check.Pass {
			level.Pass = false
		}
		level.Checks = append(level.Checks, check)
	}
	return level
}

func evaluateRequirement(req Requirement, report *posture.PostureReport) posture.ConformanceCheck {
	expr := strings.TrimSpace(req.Check)
	check := posture.ConformanceCheck{Name: posture.CheckName(strings.TrimSpace(req.Name))}
	pass, actual, err := evalExpression(expr, report)
	switch {
	case err != nil:
		check.Reason = fmt.Sprintf("%s: %v", expr, err)
	case pass:
		check.Pass = true
		check.Reason = expr
	default:
		check.Reason = fmt.Sprintf("%s: actual %s", expr, actual)
	}
	return check
}

// LoadPolicy loads the custom section of the project's policy file
// (PolicyFileName at projectRoot). It returns (nil, nil) when there is no
// policy file or the file has no custom section, and an error when the file
// exists but cannot be read, parsed or validated.
func LoadPolicy(projectRoot string) (*Custom, error) {
	pf, err := LoadFile(filepath.Join(projectRoot, PolicyFileName()))
	if err != nil || pf == nil {
		return nil, err
	}
	return pf.Conformance.Custom, nil
}

// PolicyError returns the failing custom level that stands in for a policy
// that could not be evaluated at all: its single PolicyFileCheck carries the
// cause, so a broken policy fails the gate instead of being skipped.
func PolicyError(err error) *posture.ConformanceLevel {
	return &posture.ConformanceLevel{
		Checks: []posture.ConformanceCheck{{
			Name:   PolicyFileCheck,
			Reason: err.Error(),
		}},
	}
}

// Apply loads the project's policy file (see LoadPolicy) and, when it declares
// custom requirements, records their evaluation against report in
// report.Conformance.Custom. With no policy file, or one without a custom
// section, the report is left unchanged; a policy file that cannot be loaded
// yields a failing level (PolicyError).
func Apply(projectRoot string, report *posture.PostureReport) {
	custom, err := LoadPolicy(projectRoot)
	switch {
	case err != nil:
		report.Conformance.Custom = PolicyError(err)
	case custom != nil:
		report.Conformance.Custom = Evaluate(custom, report)
	}
}
