package sarif

import (
	"cmp"
	"fmt"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/policy"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/risk"
)

type SarifResult struct {
	RuleID              string
	Level               string
	Message             string
	ArtifactURI         string
	SecuritySeverity    float64
	PartialFingerprints map[string]string
}

const (
	// policyRulePrefix namespaces policy rule IDs in SARIF output.
	policyRulePrefix = "qsdev/policy/"
	// MonitorRuleID is the SARIF rule every monitor-mode policy hit reports
	// under, so its level matches the "note" descriptor rather than the level
	// of the enforcing rule it shadows.
	MonitorRuleID = policyRulePrefix + "MONITOR"
)

// FindingFromDecision converts a policy decision into a SARIF result for the
// rule that produced it. A monitor-mode hit reports under MonitorRuleID; any
// other hit reports under qsdev/policy/<rule ID>, so each policy rule (built-in
// or custom) keeps its own identity. artifactPath is the file the tool call
// touched; when it is empty the result carries no location rather than a
// guessed one. The fingerprint combines the policy rule with the path (or,
// without one, the decision message), so distinct violations of a rule stay
// distinct alerts.
func FindingFromDecision(decision policy.PolicyDecision, rule policy.PolicyRule, artifactPath string) SarifResult {
	level, secSev := PolicySeverityToSARIF(rule.Category, rule.Severity, rule.BypassTier, rule.MonitorMode)

	policyRuleID := policyRulePrefix + cmp.Or(rule.ID, decision.RuleID, "unknown")
	ruleID := policyRuleID
	if rule.MonitorMode {
		ruleID = MonitorRuleID
	}

	if artifactPath != "" {
		artifactPath = filepath.ToSlash(filepath.Clean(artifactPath))
	}

	return SarifResult{
		RuleID:           ruleID,
		Level:            level,
		Message:          decision.Message,
		ArtifactURI:      artifactPath,
		SecuritySeverity: secSev,
		PartialFingerprints: map[string]string{
			"ruleId":     ruleID,
			"policyRule": policyRuleID,
			"target":     cmp.Or(artifactPath, decision.Message),
		},
	}
}

// ceilingToRiskRule maps the hard ceilings risk.ApplyCeilings reports to their
// dedicated SARIF rules. A ceiling without a dedicated rule reports under the
// grade-based rule.
var ceilingToRiskRule = map[string]string{
	"malware":                    "qsdev/dep-risk/malware",
	"kev":                        "qsdev/dep-risk/kev-listed",
	"critical-cve-fix-available": "qsdev/dep-risk/critical-cve",
	"critical-cve-no-fix":        "qsdev/dep-risk/critical-cve",
}

var gradeToRiskRule = map[risk.RiskGrade]string{
	risk.GradeC: "qsdev/dep-risk/moderate-risk",
	risk.GradeD: "qsdev/dep-risk/high-risk",
	risk.GradeF: "qsdev/dep-risk/failing",
}

// FindingFromRiskScore converts a package score into a SARIF result, or nil
// for a passing (A/B) package with no hard ceiling. A malware, KEV or
// critical-CVE ceiling reports under its dedicated rule; otherwise the grade
// selects the rule. Level and security severity come from the rule's
// descriptor so result and catalog never disagree.
func FindingFromRiskScore(score risk.PackageScore) *SarifResult {
	ruleID, ok := ceilingToRiskRule[score.CeilingApplied]
	if !ok {
		ruleID, ok = gradeToRiskRule[score.Grade]
	}
	if !ok {
		return nil
	}
	def, _ := ruleDefinition(ruleID)

	msg := fmt.Sprintf("%s@%s: risk score %d (grade %s)", score.PackageName, score.PackageVersion, score.Score, score.Grade)
	if score.CeilingApplied != "" {
		msg += fmt.Sprintf(", capped by %s", score.CeilingApplied)
	}

	return &SarifResult{
		RuleID:           ruleID,
		Level:            def.DefaultLevel,
		Message:          msg,
		ArtifactURI:      "",
		SecuritySeverity: def.SecuritySeverity,
		PartialFingerprints: map[string]string{
			"ruleId":  ruleID,
			"package": fmt.Sprintf("%s/%s@%s", score.Ecosystem, score.PackageName, score.PackageVersion),
		},
	}
}

var eventTypeToTrustRuleID = map[string]string{
	"confused-deputy-blocked":     "qsdev/trust/confused-deputy-blocked",
	"tier3-injection":             "qsdev/trust/tier3-injection",
	"server-vulnerability":        "qsdev/trust/server-vulnerability",
	"tier3-post-fetch-suspicious": "qsdev/trust/tier3-post-fetch-suspicious",
}

// FindingFromTrustEvent converts an MCP trust event into a SARIF result
// located at .mcp.json, where MCP servers are configured. The fingerprint
// includes the event message so distinct events on one server stay distinct.
func FindingFromTrustEvent(serverName, eventType, message string) SarifResult {
	ruleID, ok := eventTypeToTrustRuleID[eventType]
	if !ok {
		ruleID = "qsdev/trust/" + eventType
	}

	level, secSev := "warning", 5.0
	if def, ok := ruleDefinition(ruleID); ok {
		level, secSev = def.DefaultLevel, def.SecuritySeverity
	}

	return SarifResult{
		RuleID:           ruleID,
		Level:            level,
		Message:          fmt.Sprintf("[%s] %s", serverName, message),
		ArtifactURI:      ".mcp.json",
		SecuritySeverity: secSev,
		PartialFingerprints: map[string]string{
			"ruleId": ruleID,
			"server": serverName,
			"target": message,
		},
	}
}

// ruleDefinition looks up a rule in the catalog.
func ruleDefinition(id string) (RuleDefinition, bool) {
	for _, r := range AllRules {
		if r.ID == id {
			return r, true
		}
	}
	return RuleDefinition{}, false
}
