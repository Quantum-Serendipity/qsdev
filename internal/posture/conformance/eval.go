package conformance

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
)

// ErrDependenciesInconclusive is returned (wrapped) for a dependencies.*
// expression when the report's vulnerability counts cannot back a verdict:
// no fresh scan completed, or the scan was not certifiable (an ecosystem's
// scan errored, or a vulnerability's severity could not be resolved). Zero
// totals then mean "unknown", not "clean", so the requirement must fail.
var ErrDependenciesInconclusive = errors.New("dependency vulnerability results are inconclusive")

// EvalCheckExpression evaluates a single policy check expression against
// a PostureReport. Returns (true, nil) if the check passes.
//
// Supported expression forms:
//
//	defense.<layer>.status == enabled|disabled|partial|not-applicable
//	dependencies.totals.<severity> == <int>
//	dependencies.totals.<severity> <= <int>
//	dependencies.totals.<severity> >= <int>
//	config.score >= <float>
//	score.total >= <float>
//	tools.<name>.enabled == true|false
//
// <severity> is one of critical, high, moderate, low, info or unknown.
//
// An error means the check cannot pass: either the expression is invalid, or
// (ErrDependenciesInconclusive) it reads dependency totals that no conclusive
// scan produced. Callers must treat any error as a failed check.
func EvalCheckExpression(expr string, report *posture.PostureReport) (bool, error) {
	pass, _, err := evalExpression(expr, report)
	return pass, err
}

// evalExpression evaluates expr and also returns the observed left-hand value,
// so a failing requirement can say what was actually found.
func evalExpression(expr string, report *posture.PostureReport) (pass bool, actual string, err error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return false, "", fmt.Errorf("empty expression")
	}

	// Parse: <lhs> <op> <rhs>
	lhs, op, rhs, err := parseExpression(expr)
	if err != nil {
		return false, "", err
	}

	parts := strings.Split(lhs, ".")

	switch parts[0] {
	case "defense":
		return evalDefense(parts, op, rhs, report)
	case "dependencies":
		return evalDependencies(parts, op, rhs, report)
	case "config":
		return evalConfig(parts, op, rhs, report)
	case "score":
		return evalScore(parts, op, rhs, report)
	case "tools":
		return evalTools(parts, op, rhs, report)
	default:
		return false, "", fmt.Errorf("unknown expression domain: %q", parts[0])
	}
}

func parseExpression(expr string) (lhs, op, rhs string, err error) {
	// Try operators in order of length (longest first).
	for _, candidate := range []string{"<=", ">=", "=="} {
		idx := strings.Index(expr, candidate)
		if idx >= 0 {
			lhs = strings.TrimSpace(expr[:idx])
			op = candidate
			rhs = strings.TrimSpace(expr[idx+len(candidate):])
			return lhs, op, rhs, nil
		}
	}
	return "", "", "", fmt.Errorf("no supported operator found in expression: %q", expr)
}

func evalDefense(parts []string, op, rhs string, report *posture.PostureReport) (bool, string, error) {
	// defense.<layer>.status == <status>
	if len(parts) != 3 || parts[2] != "status" {
		return false, "", fmt.Errorf("invalid defense expression: expected defense.<layer>.status")
	}
	if op != "==" {
		return false, "", fmt.Errorf("defense.*.status only supports == operator")
	}

	layerName := parts[1]
	layer := posture.FindLayerByName(report.Defense.Layers, layerName)
	if layer == nil {
		return false, "", fmt.Errorf("unknown defense layer: %q", layerName)
	}

	return string(layer.Status) == rhs, string(layer.Status), nil
}

func evalDependencies(parts []string, op, rhs string, report *posture.PostureReport) (bool, string, error) {
	// dependencies.totals.<severity> <op> <int>
	if len(parts) != 3 || parts[1] != "totals" {
		return false, "", fmt.Errorf("invalid dependencies expression: expected dependencies.totals.<severity>")
	}

	totals := report.Dependencies.Totals
	var actual int
	switch severity := parts[2]; severity {
	case "critical":
		actual = totals.Critical
	case "high":
		actual = totals.High
	case "moderate":
		actual = totals.Moderate
	case "low":
		actual = totals.Low
	case "info":
		actual = totals.Info
	case "unknown":
		actual = totals.Unknown
	default:
		return false, "", fmt.Errorf("unknown severity: %q", severity)
	}

	expected, err := strconv.Atoi(rhs)
	if err != nil {
		return false, "", fmt.Errorf("invalid integer in expression: %q", rhs)
	}

	// The expression is well formed; only now decide whether the totals it
	// reads mean anything. Unscanned or inconclusive totals are zero, which
	// would otherwise satisfy "== 0" without a single dependency checked.
	if err := dependenciesConclusive(report.Dependencies); err != nil {
		return false, "", err
	}

	pass, err := compareInt(actual, op, expected)
	return pass, strconv.Itoa(actual), err
}

// dependenciesConclusive reports, as a wrapped ErrDependenciesInconclusive,
// why a report's dependency totals cannot be relied on; nil when a fresh scan
// completed and is certifiable.
func dependenciesConclusive(deps posture.DependencyHealth) error {
	switch {
	case deps.ScanFailed:
		return fmt.Errorf("%w: the dependency vulnerability scan failed", ErrDependenciesInconclusive)
	case !deps.Scanned:
		return fmt.Errorf("%w: dependencies were not scanned for vulnerabilities (run with --scan)",
			ErrDependenciesInconclusive)
	case !deps.Certifiable():
		return fmt.Errorf("%w: %d vulnerabilities have an unresolved severity",
			ErrDependenciesInconclusive, deps.Totals.Unknown)
	default:
		return nil
	}
}

func evalConfig(parts []string, op, rhs string, report *posture.PostureReport) (bool, string, error) {
	// config.score >= <float>
	if len(parts) != 2 || parts[1] != "score" {
		return false, "", fmt.Errorf("invalid config expression: expected config.score")
	}

	expected, err := strconv.ParseFloat(rhs, 64)
	if err != nil {
		return false, "", fmt.Errorf("invalid float in expression: %q", rhs)
	}

	pass, err := compareFloat(report.Config.Score, op, expected)
	return pass, formatFloat(report.Config.Score), err
}

func evalScore(parts []string, op, rhs string, report *posture.PostureReport) (bool, string, error) {
	// score.total >= <float>
	if len(parts) != 2 || parts[1] != "total" {
		return false, "", fmt.Errorf("invalid score expression: expected score.total")
	}

	expected, err := strconv.ParseFloat(rhs, 64)
	if err != nil {
		return false, "", fmt.Errorf("invalid float in expression: %q", rhs)
	}

	pass, err := compareFloat(report.Score.Total, op, expected)
	return pass, formatFloat(report.Score.Total), err
}

func evalTools(parts []string, op, rhs string, report *posture.PostureReport) (bool, string, error) {
	// tools.<name>.enabled == true|false
	if len(parts) != 3 || parts[2] != "enabled" {
		return false, "", fmt.Errorf("invalid tools expression: expected tools.<name>.enabled")
	}
	if op != "==" {
		return false, "", fmt.Errorf("tools.*.enabled only supports == operator")
	}

	toolName := parts[1]
	enabled := false
	for _, t := range report.Tools {
		if t.Name == toolName {
			enabled = t.Enabled
			break
		}
	}
	actual := strconv.FormatBool(enabled)

	switch rhs {
	case "true":
		return enabled, actual, nil
	case "false":
		return !enabled, actual, nil
	default:
		return false, "", fmt.Errorf("tools.*.enabled value must be true or false, got: %q", rhs)
	}
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func compareInt(actual int, op string, expected int) (bool, error) {
	switch op {
	case "==":
		return actual == expected, nil
	case "<=":
		return actual <= expected, nil
	case ">=":
		return actual >= expected, nil
	default:
		return false, fmt.Errorf("unsupported operator for integer comparison: %q", op)
	}
}

func compareFloat(actual float64, op string, expected float64) (bool, error) {
	switch op {
	case "==":
		return actual == expected, nil
	case "<=":
		return actual <= expected, nil
	case ">=":
		return actual >= expected, nil
	default:
		return false, fmt.Errorf("unsupported operator for float comparison: %q", op)
	}
}
