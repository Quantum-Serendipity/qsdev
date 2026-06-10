package sarif

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/policy"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/risk"
)

func TestAllRuleIDsUnique(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool, len(AllRules))
	for _, r := range AllRules {
		if seen[r.ID] {
			t.Errorf("duplicate rule ID: %s", r.ID)
		}
		seen[r.ID] = true
	}
}

func TestRulesByNamespace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		namespace string
		wantCount int
	}{
		{"qsdev/policy", 9},
		{"qsdev/dep-risk", 5},
		{"qsdev/trust", 4},
		{"qsdev/nonexistent", 0},
	}

	for _, tt := range tests {
		t.Run(tt.namespace, func(t *testing.T) {
			t.Parallel()
			got := RulesByNamespace(tt.namespace)
			if len(got) != tt.wantCount {
				t.Errorf("RulesByNamespace(%q) returned %d rules, want %d", tt.namespace, len(got), tt.wantCount)
			}
		})
	}
}

func TestPolicySeverityToSARIF(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		category    string
		severity    policy.Severity
		bypassTier  policy.BypassTier
		monitorMode bool
		wantLevel   string
		wantSev     float64
	}{
		{
			name:        "monitor mode always note",
			category:    "self-protection",
			severity:    policy.Critical,
			bypassTier:  policy.EnforceAlways,
			monitorMode: true,
			wantLevel:   "note",
			wantSev:     1.0,
		},
		{
			name:       "self-protection critical enforce-always",
			category:   "self-protection",
			severity:   policy.Critical,
			bypassTier: policy.EnforceAlways,
			wantLevel:  "error",
			wantSev:    10.0,
		},
		{
			name:       "mcp-poisoning critical",
			category:   "mcp-poisoning",
			severity:   policy.Critical,
			bypassTier: policy.Session,
			wantLevel:  "error",
			wantSev:    10.0,
		},
		{
			name:       "config-guard high session",
			category:   "config-guard",
			severity:   policy.High,
			bypassTier: policy.Session,
			wantLevel:  "error",
			wantSev:    7.0,
		},
		{
			name:       "config-guard medium session",
			category:   "config-guard",
			severity:   policy.Medium,
			bypassTier: policy.Session,
			wantLevel:  "warning",
			wantSev:    5.0,
		},
		{
			name:       "integrity critical enforce-always",
			category:   "integrity",
			severity:   policy.Critical,
			bypassTier: policy.EnforceAlways,
			wantLevel:  "error",
			wantSev:    9.0,
		},
		{
			name:       "integrity high command",
			category:   "integrity",
			severity:   policy.High,
			bypassTier: policy.Command,
			wantLevel:  "warning",
			wantSev:    6.0,
		},
		{
			name:       "fallback critical",
			category:   "unknown",
			severity:   policy.Critical,
			bypassTier: policy.Command,
			wantLevel:  "error",
			wantSev:    9.0,
		},
		{
			name:       "fallback high",
			category:   "unknown",
			severity:   policy.High,
			bypassTier: policy.Command,
			wantLevel:  "error",
			wantSev:    7.0,
		},
		{
			name:       "fallback medium",
			category:   "unknown",
			severity:   policy.Medium,
			bypassTier: policy.Command,
			wantLevel:  "warning",
			wantSev:    5.0,
		},
		{
			name:       "fallback low",
			category:   "unknown",
			severity:   policy.Low,
			bypassTier: policy.Command,
			wantLevel:  "note",
			wantSev:    3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			level, sev := PolicySeverityToSARIF(tt.category, tt.severity, tt.bypassTier, tt.monitorMode)
			if level != tt.wantLevel {
				t.Errorf("level = %q, want %q", level, tt.wantLevel)
			}
			if sev != tt.wantSev {
				t.Errorf("securitySeverity = %v, want %v", sev, tt.wantSev)
			}
		})
	}
}

func TestFindingFromDecisionCategories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		category   string
		wantRuleID string
	}{
		{"self-protection", "qsdev/policy/SP-001"},
		{"config-guard", "qsdev/policy/CG-001"},
		{"mcp-poisoning", "qsdev/policy/MCP-001"},
		{"integrity", "qsdev/policy/INT-001"},
		{"unknown-category", "qsdev/policy/MONITOR"},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			t.Parallel()
			decision := policy.PolicyDecision{
				Action:  policy.Block,
				Message: "test message",
			}
			result := FindingFromDecision(decision, tt.category, policy.Critical, policy.EnforceAlways, false)
			if result.RuleID != tt.wantRuleID {
				t.Errorf("ruleID = %q, want %q", result.RuleID, tt.wantRuleID)
			}
			if result.Message != "test message" {
				t.Errorf("message = %q, want %q", result.Message, "test message")
			}
		})
	}
}

func TestFindingFromRiskScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		grade      risk.RiskGrade
		wantNil    bool
		wantRuleID string
		wantLevel  string
	}{
		{"grade A returns nil", risk.GradeA, true, "", ""},
		{"grade B returns nil", risk.GradeB, true, "", ""},
		{"grade C returns high-risk note", risk.GradeC, false, "qsdev/dep-risk/high-risk", "note"},
		{"grade D returns high-risk warning", risk.GradeD, false, "qsdev/dep-risk/high-risk", "warning"},
		{"grade F returns failing error", risk.GradeF, false, "qsdev/dep-risk/failing", "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			score := risk.PackageScore{
				PackageName:    "test-pkg",
				PackageVersion: "1.0.0",
				Score:          42,
				Grade:          tt.grade,
			}
			result := FindingFromRiskScore(score)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("expected non-nil result")
				return
			}
			if result.RuleID != tt.wantRuleID {
				t.Errorf("ruleID = %q, want %q", result.RuleID, tt.wantRuleID)
			}
			if result.Level != tt.wantLevel {
				t.Errorf("level = %q, want %q", result.Level, tt.wantLevel)
			}
		})
	}
}

func TestFindingFromDecisionMonitorMode(t *testing.T) {
	t.Parallel()

	categories := []string{"self-protection", "config-guard", "mcp-poisoning", "integrity"}
	for _, cat := range categories {
		t.Run(cat, func(t *testing.T) {
			t.Parallel()
			decision := policy.PolicyDecision{Message: "monitor test"}
			result := FindingFromDecision(decision, cat, policy.Critical, policy.EnforceAlways, true)
			if result.Level != "note" {
				t.Errorf("monitor mode level = %q, want %q", result.Level, "note")
			}
			if result.SecuritySeverity != 1.0 {
				t.Errorf("monitor mode severity = %v, want 1.0", result.SecuritySeverity)
			}
		})
	}
}

func TestFindingFromTrustEvent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		eventType  string
		wantRuleID string
		wantLevel  string
	}{
		{"confused-deputy-blocked", "qsdev/trust/confused-deputy-blocked", "error"},
		{"tier3-injection", "qsdev/trust/tier3-injection", "error"},
		{"server-vulnerability", "qsdev/trust/server-vulnerability", "warning"},
		{"tier3-post-fetch-suspicious", "qsdev/trust/tier3-post-fetch-suspicious", "warning"},
		{"unknown-event", "qsdev/trust/unknown-event", "warning"},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			t.Parallel()
			result := FindingFromTrustEvent("test-server", tt.eventType, "test message")
			if result.RuleID != tt.wantRuleID {
				t.Errorf("ruleID = %q, want %q", result.RuleID, tt.wantRuleID)
			}
			if result.Level != tt.wantLevel {
				t.Errorf("level = %q, want %q", result.Level, tt.wantLevel)
			}
			if result.ArtifactURI != ".mcp.json" {
				t.Errorf("artifactURI = %q, want %q", result.ArtifactURI, ".mcp.json")
			}
		})
	}
}
