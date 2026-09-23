package sarif

import (
	"maps"
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
		{"qsdev/dep-risk", 6},
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

// TestFindingFromDecision guards F192: the result is keyed on the rule that
// fired (not a per-category guess), carries the real path or no location at
// all, and fingerprints distinct violations distinctly.
func TestFindingFromDecision(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		rule            policy.PolicyRule
		decision        policy.PolicyDecision
		path            string
		wantRuleID      string
		wantLevel       string
		wantURI         string
		wantPolicyRule  string
		wantFingerprint string
	}{
		{
			name:            "credential rule keeps its own ID and real path",
			rule:            policy.PolicyRule{ID: "CG-003", Category: "config-guard", Severity: policy.High, BypassTier: policy.Session},
			decision:        policy.PolicyDecision{Action: policy.Block, RuleID: "CG-003", Message: "Cannot read ~/.ssh/id_rsa"},
			path:            "/home/u/.ssh/./id_rsa",
			wantRuleID:      "qsdev/policy/CG-003",
			wantLevel:       "error",
			wantURI:         "/home/u/.ssh/id_rsa",
			wantPolicyRule:  "qsdev/policy/CG-003",
			wantFingerprint: "/home/u/.ssh/id_rsa",
		},
		{
			name:            "custom rule in unknown category is not reported as monitor",
			rule:            policy.PolicyRule{ID: "ORG-042", Category: "org-custom", Severity: policy.Critical, BypassTier: policy.EnforceAlways},
			decision:        policy.PolicyDecision{Action: policy.Block, RuleID: "ORG-042", Message: "blocked by org"},
			wantRuleID:      "qsdev/policy/ORG-042",
			wantLevel:       "error",
			wantPolicyRule:  "qsdev/policy/ORG-042",
			wantFingerprint: "blocked by org",
		},
		{
			name:            "monitor hit reports under MONITOR but fingerprints the rule",
			rule:            policy.PolicyRule{ID: "SP-001", Category: "self-protection", Severity: policy.Critical, BypassTier: policy.EnforceAlways, MonitorMode: true},
			decision:        policy.PolicyDecision{RuleID: "SP-001", Message: "would block"},
			path:            ".claude/settings.json",
			wantRuleID:      MonitorRuleID,
			wantLevel:       "note",
			wantURI:         ".claude/settings.json",
			wantPolicyRule:  "qsdev/policy/SP-001",
			wantFingerprint: ".claude/settings.json",
		},
		{
			name:            "rule ID falls back to the decision",
			rule:            policy.PolicyRule{Category: "integrity", Severity: policy.Medium, BypassTier: policy.Command},
			decision:        policy.PolicyDecision{Action: policy.Block, RuleID: "INT-003", Message: "m"},
			wantRuleID:      "qsdev/policy/INT-003",
			wantLevel:       "warning",
			wantPolicyRule:  "qsdev/policy/INT-003",
			wantFingerprint: "m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := FindingFromDecision(tt.decision, tt.rule, tt.path)
			if result.RuleID != tt.wantRuleID {
				t.Errorf("RuleID = %q, want %q", result.RuleID, tt.wantRuleID)
			}
			if result.Level != tt.wantLevel {
				t.Errorf("Level = %q, want %q", result.Level, tt.wantLevel)
			}
			if result.ArtifactURI != tt.wantURI {
				t.Errorf("ArtifactURI = %q, want %q", result.ArtifactURI, tt.wantURI)
			}
			if result.Message != tt.decision.Message {
				t.Errorf("Message = %q, want %q", result.Message, tt.decision.Message)
			}
			if got := result.PartialFingerprints["policyRule"]; got != tt.wantPolicyRule {
				t.Errorf("fingerprint policyRule = %q, want %q", got, tt.wantPolicyRule)
			}
			if got := result.PartialFingerprints["target"]; got != tt.wantFingerprint {
				t.Errorf("fingerprint target = %q, want %q", got, tt.wantFingerprint)
			}
		})
	}
}

func TestFindingFromDecision_DistinctViolationsDistinctFingerprints(t *testing.T) {
	t.Parallel()

	rule := policy.PolicyRule{ID: "CG-001", Category: "config-guard", Severity: policy.High, BypassTier: policy.Session}
	a := FindingFromDecision(policy.PolicyDecision{Message: "x"}, rule, ".npmrc")
	b := FindingFromDecision(policy.PolicyDecision{Message: "x"}, rule, ".pypirc")
	if maps.Equal(a.PartialFingerprints, b.PartialFingerprints) {
		t.Errorf("two different files share fingerprint %v", a.PartialFingerprints)
	}
}

func TestFindingFromRiskScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		grade      risk.RiskGrade
		ceiling    string
		wantNil    bool
		wantRuleID string
		wantLevel  string
		wantSev    float64
	}{
		{name: "grade A returns nil", grade: risk.GradeA, wantNil: true},
		{name: "grade B returns nil", grade: risk.GradeB, wantNil: true},
		{name: "grade C returns moderate-risk note", grade: risk.GradeC, wantRuleID: "qsdev/dep-risk/moderate-risk", wantLevel: "note", wantSev: 4.0},
		{name: "grade D returns high-risk warning", grade: risk.GradeD, wantRuleID: "qsdev/dep-risk/high-risk", wantLevel: "warning", wantSev: 6.0},
		{name: "grade F returns failing error", grade: risk.GradeF, wantRuleID: "qsdev/dep-risk/failing", wantLevel: "error", wantSev: 8.0},
		{name: "malware ceiling", grade: risk.GradeF, ceiling: "malware", wantRuleID: "qsdev/dep-risk/malware", wantLevel: "error", wantSev: 10.0},
		{name: "kev ceiling", grade: risk.GradeF, ceiling: "kev", wantRuleID: "qsdev/dep-risk/kev-listed", wantLevel: "error", wantSev: 10.0},
		{name: "critical cve ceiling", grade: risk.GradeF, ceiling: "critical-cve-no-fix", wantRuleID: "qsdev/dep-risk/critical-cve", wantLevel: "error", wantSev: 9.0},
		{name: "ceiling without dedicated rule uses grade", grade: risk.GradeF, ceiling: "unblocked-install-scripts", wantRuleID: "qsdev/dep-risk/failing", wantLevel: "error", wantSev: 8.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			score := risk.PackageScore{
				PackageName:    "test-pkg",
				PackageVersion: "1.0.0",
				Ecosystem:      risk.EcosystemNpm,
				Score:          42,
				Grade:          tt.grade,
				CeilingApplied: tt.ceiling,
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
			if result.SecuritySeverity != tt.wantSev {
				t.Errorf("security severity = %v, want %v", result.SecuritySeverity, tt.wantSev)
			}
		})
	}
}

// TestFindingFromRiskScore_CeilingNamesMatchScorer pins the ceiling names the
// SARIF mapping relies on to the names risk.ApplyCeilings actually reports.
func TestFindingFromRiskScore_CeilingNamesMatchScorer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		info       risk.PackageInfo
		wantRuleID string
	}{
		{risk.PackageInfo{MalwareDetected: true}, "qsdev/dep-risk/malware"},
		{risk.PackageInfo{KEVListed: true}, "qsdev/dep-risk/kev-listed"},
		{risk.PackageInfo{CVECritical: 1, FixAvailable: true}, "qsdev/dep-risk/critical-cve"},
		{risk.PackageInfo{CVECritical: 1}, "qsdev/dep-risk/critical-cve"},
	}

	for _, tt := range tests {
		t.Run(tt.wantRuleID, func(t *testing.T) {
			t.Parallel()
			_, ceiling := risk.ApplyCeilings(100, &tt.info)
			result := FindingFromRiskScore(risk.PackageScore{Grade: risk.GradeF, CeilingApplied: ceiling})
			if result == nil || result.RuleID != tt.wantRuleID {
				t.Errorf("ceiling %q maps to %+v, want rule %q", ceiling, result, tt.wantRuleID)
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
			rule := policy.PolicyRule{ID: "R-1", Category: cat, Severity: policy.Critical, BypassTier: policy.EnforceAlways, MonitorMode: true}
			result := FindingFromDecision(decision, rule, "")
			if result.RuleID != MonitorRuleID {
				t.Errorf("monitor mode ruleID = %q, want %q", result.RuleID, MonitorRuleID)
			}
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
