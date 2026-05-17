package policy

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
)

// EvalCheckExpression evaluates a single policy check expression against
// a PostureReport. Returns (true, nil) if the check passes.
//
// Supported expression forms:
//
//	defense.<layer>.status == enabled|disabled|partial|not-applicable
//	dependencies.totals.<severity> == <int>
//	dependencies.totals.<severity> <= <int>
//	config.score >= <float>
//	score.total >= <float>
//	tools.<name>.enabled == true|false
func EvalCheckExpression(expr string, report *posture.PostureReport) (bool, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return false, fmt.Errorf("empty expression")
	}

	// Parse: <lhs> <op> <rhs>
	lhs, op, rhs, err := parseExpression(expr)
	if err != nil {
		return false, err
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
		return false, fmt.Errorf("unknown expression domain: %q", parts[0])
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

func evalDefense(parts []string, op, rhs string, report *posture.PostureReport) (bool, error) {
	// defense.<layer>.status == <status>
	if len(parts) != 3 || parts[2] != "status" {
		return false, fmt.Errorf("invalid defense expression: expected defense.<layer>.status")
	}
	if op != "==" {
		return false, fmt.Errorf("defense.*.status only supports == operator")
	}

	layerName := parts[1]
	layer := findLayerByName(report.Defense.Layers, layerName)
	if layer == nil {
		return false, fmt.Errorf("unknown defense layer: %q", layerName)
	}

	return string(layer.Status) == rhs, nil
}

func evalDependencies(parts []string, op, rhs string, report *posture.PostureReport) (bool, error) {
	// dependencies.totals.<severity> <op> <int>
	if len(parts) != 3 || parts[1] != "totals" {
		return false, fmt.Errorf("invalid dependencies expression: expected dependencies.totals.<severity>")
	}

	severity := parts[2]
	var actual int
	switch severity {
	case "critical":
		actual = report.Dependencies.Totals.Critical
	case "high":
		actual = report.Dependencies.Totals.High
	case "moderate":
		actual = report.Dependencies.Totals.Moderate
	case "low":
		actual = report.Dependencies.Totals.Low
	default:
		return false, fmt.Errorf("unknown severity: %q", severity)
	}

	expected, err := strconv.Atoi(rhs)
	if err != nil {
		return false, fmt.Errorf("invalid integer in expression: %q", rhs)
	}

	return compareInt(actual, op, expected)
}

func evalConfig(parts []string, op, rhs string, report *posture.PostureReport) (bool, error) {
	// config.score >= <float>
	if len(parts) != 2 || parts[1] != "score" {
		return false, fmt.Errorf("invalid config expression: expected config.score")
	}

	expected, err := strconv.ParseFloat(rhs, 64)
	if err != nil {
		return false, fmt.Errorf("invalid float in expression: %q", rhs)
	}

	return compareFloat(report.Config.Score, op, expected)
}

func evalScore(parts []string, op, rhs string, report *posture.PostureReport) (bool, error) {
	// score.total >= <float>
	if len(parts) != 2 || parts[1] != "total" {
		return false, fmt.Errorf("invalid score expression: expected score.total")
	}

	expected, err := strconv.ParseFloat(rhs, 64)
	if err != nil {
		return false, fmt.Errorf("invalid float in expression: %q", rhs)
	}

	return compareFloat(report.Score.Total, op, expected)
}

func evalTools(parts []string, op, rhs string, report *posture.PostureReport) (bool, error) {
	// tools.<name>.enabled == true|false
	if len(parts) != 3 || parts[2] != "enabled" {
		return false, fmt.Errorf("invalid tools expression: expected tools.<name>.enabled")
	}
	if op != "==" {
		return false, fmt.Errorf("tools.*.enabled only supports == operator")
	}

	toolName := parts[1]
	enabled := false
	for _, t := range report.Tools {
		if t.Name == toolName {
			enabled = t.Enabled
			break
		}
	}

	switch rhs {
	case "true":
		return enabled, nil
	case "false":
		return !enabled, nil
	default:
		return false, fmt.Errorf("tools.*.enabled value must be true or false, got: %q", rhs)
	}
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

// findLayerByName returns a pointer to the layer with the given name,
// or nil if not found.
func findLayerByName(layers []posture.DefenseLayer, name string) *posture.DefenseLayer {
	for i, l := range layers {
		if l.Name == name {
			return &layers[i]
		}
	}
	return nil
}
