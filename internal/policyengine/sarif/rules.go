package sarif

import "strings"

type RuleDefinition struct {
	ID               string
	Name             string
	ShortDescription string
	DefaultLevel     string
	SecuritySeverity float64
	Tags             []string
}

var AllRules = []RuleDefinition{
	// Policy rules (qsdev/policy/)
	{
		ID:               "qsdev/policy/SP-001",
		Name:             "self-protection-tamper",
		ShortDescription: "Self-protection: config tampering blocked",
		DefaultLevel:     "error",
		SecuritySeverity: 10.0,
		Tags:             []string{"security", "self-protection"},
	},
	{
		ID:               "qsdev/policy/SP-002",
		Name:             "self-protection-semantic",
		ShortDescription: "Self-protection: semantic check triggered",
		DefaultLevel:     "note",
		SecuritySeverity: 3.0,
		Tags:             []string{"security", "self-protection"},
	},
	{
		ID:               "qsdev/policy/CG-001",
		Name:             "config-guard-credential",
		ShortDescription: "Config-guard: credential file access blocked",
		DefaultLevel:     "error",
		SecuritySeverity: 8.0,
		Tags:             []string{"security", "config-guard"},
	},
	{
		ID:               "qsdev/policy/CG-002",
		Name:             "config-guard-package-config",
		ShortDescription: "Config-guard: package config modification detected",
		DefaultLevel:     "warning",
		SecuritySeverity: 5.0,
		Tags:             []string{"security", "config-guard"},
	},
	{
		ID:               "qsdev/policy/MCP-001",
		Name:             "mcp-poisoning-injection",
		ShortDescription: "MCP-poisoning: injection scan detection",
		DefaultLevel:     "warning",
		SecuritySeverity: 6.0,
		Tags:             []string{"security", "mcp-poisoning"},
	},
	{
		ID:               "qsdev/policy/MCP-002",
		Name:             "mcp-poisoning-deputy",
		ShortDescription: "MCP-poisoning: confused deputy access blocked",
		DefaultLevel:     "error",
		SecuritySeverity: 9.0,
		Tags:             []string{"security", "mcp-poisoning"},
	},
	{
		ID:               "qsdev/policy/INT-001",
		Name:             "integrity-dangerous-command",
		ShortDescription: "Integrity: dangerous command blocked",
		DefaultLevel:     "error",
		SecuritySeverity: 9.0,
		Tags:             []string{"security", "integrity"},
	},
	{
		ID:               "qsdev/policy/INT-002",
		Name:             "integrity-lockfile",
		ShortDescription: "Integrity: lock file check failed",
		DefaultLevel:     "note",
		SecuritySeverity: 3.0,
		Tags:             []string{"security", "integrity"},
	},
	{
		ID:               "qsdev/policy/MONITOR",
		Name:             "policy-monitor",
		ShortDescription: "Policy monitor-mode violation",
		DefaultLevel:     "note",
		SecuritySeverity: 1.0,
		Tags:             []string{"security", "monitor"},
	},

	// Risk rules (qsdev/dep-risk/)
	{
		ID:               "qsdev/dep-risk/critical-cve",
		Name:             "critical-cve",
		ShortDescription: "Dependency with critical CVE",
		DefaultLevel:     "error",
		SecuritySeverity: 9.0,
		Tags:             []string{"supply-chain", "vulnerability"},
	},
	{
		ID:               "qsdev/dep-risk/kev-listed",
		Name:             "kev-listed",
		ShortDescription: "Dependency with known exploited vulnerability",
		DefaultLevel:     "error",
		SecuritySeverity: 10.0,
		Tags:             []string{"supply-chain", "vulnerability"},
	},
	{
		ID:               "qsdev/dep-risk/malware",
		Name:             "malware",
		ShortDescription: "Malware detected in dependency",
		DefaultLevel:     "error",
		SecuritySeverity: 10.0,
		Tags:             []string{"supply-chain", "malware"},
	},
	{
		ID:               "qsdev/dep-risk/high-risk",
		Name:             "high-risk",
		ShortDescription: "High-risk dependency (grade D)",
		DefaultLevel:     "warning",
		SecuritySeverity: 6.0,
		Tags:             []string{"supply-chain", "risk"},
	},
	{
		ID:               "qsdev/dep-risk/failing",
		Name:             "failing",
		ShortDescription: "Failing dependency (grade F)",
		DefaultLevel:     "error",
		SecuritySeverity: 8.0,
		Tags:             []string{"supply-chain", "risk"},
	},

	// Trust rules (qsdev/trust/)
	{
		ID:               "qsdev/trust/confused-deputy-blocked",
		Name:             "confused-deputy-blocked",
		ShortDescription: "Confused deputy access blocked",
		DefaultLevel:     "error",
		SecuritySeverity: 9.0,
		Tags:             []string{"trust", "confused-deputy"},
	},
	{
		ID:               "qsdev/trust/tier3-injection",
		Name:             "tier3-injection",
		ShortDescription: "Tier 3 MCP injection detected",
		DefaultLevel:     "error",
		SecuritySeverity: 8.0,
		Tags:             []string{"trust", "injection"},
	},
	{
		ID:               "qsdev/trust/server-vulnerability",
		Name:             "server-vulnerability",
		ShortDescription: "MCP server vulnerability detected",
		DefaultLevel:     "warning",
		SecuritySeverity: 6.0,
		Tags:             []string{"trust", "vulnerability"},
	},
	{
		ID:               "qsdev/trust/tier3-post-fetch-suspicious",
		Name:             "tier3-post-fetch-suspicious",
		ShortDescription: "Suspicious post-fetch pattern detected",
		DefaultLevel:     "warning",
		SecuritySeverity: 5.0,
		Tags:             []string{"trust", "suspicious"},
	},
}

func RulesByNamespace(namespace string) []RuleDefinition {
	var matched []RuleDefinition
	prefix := namespace + "/"
	for _, r := range AllRules {
		if strings.HasPrefix(r.ID, prefix) {
			matched = append(matched, r)
		}
	}
	return matched
}
