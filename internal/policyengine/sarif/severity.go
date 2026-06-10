package sarif

import "github.com/Quantum-Serendipity/qsdev/internal/policyengine/policy"

func PolicySeverityToSARIF(category string, severity policy.Severity, bypassTier policy.BypassTier, monitorMode bool) (level string, securitySeverity float64) {
	if monitorMode {
		return "note", 1.0
	}

	switch {
	case category == "self-protection" && severity == policy.Critical && bypassTier == policy.EnforceAlways:
		return "error", 10.0
	case category == "mcp-poisoning" && severity == policy.Critical:
		return "error", 10.0
	case category == "config-guard" && severity == policy.High && bypassTier == policy.Session:
		return "error", 7.0
	case category == "config-guard" && severity == policy.Medium && bypassTier == policy.Session:
		return "warning", 5.0
	case category == "integrity" && severity == policy.Critical && bypassTier == policy.EnforceAlways:
		return "error", 9.0
	case category == "integrity" && severity == policy.High && bypassTier == policy.Command:
		return "warning", 6.0
	}

	switch severity {
	case policy.Critical:
		return "error", 9.0
	case policy.High:
		return "error", 7.0
	case policy.Medium:
		return "warning", 5.0
	default:
		return "note", 3.0
	}
}
