package sarif

import (
	"fmt"

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

var categoryToRuleID = map[string]string{
	"self-protection": "qsdev/policy/SP-001",
	"config-guard":    "qsdev/policy/CG-001",
	"mcp-poisoning":   "qsdev/policy/MCP-001",
	"integrity":       "qsdev/policy/INT-001",
}

var categoryToArtifact = map[string]string{
	"self-protection": ".qsdev/state.yaml",
	"config-guard":    ".env",
	"mcp-poisoning":   ".mcp.json",
	"integrity":       "bin/",
}

func FindingFromDecision(decision policy.PolicyDecision, category string, severity policy.Severity, bypassTier policy.BypassTier, monitorMode bool) SarifResult {
	level, secSev := PolicySeverityToSARIF(category, severity, bypassTier, monitorMode)

	ruleID, ok := categoryToRuleID[category]
	if !ok {
		ruleID = "qsdev/policy/MONITOR"
	}

	artifactURI, ok := categoryToArtifact[category]
	if !ok {
		artifactURI = ""
	}

	return SarifResult{
		RuleID:           ruleID,
		Level:            level,
		Message:          decision.Message,
		ArtifactURI:      artifactURI,
		SecuritySeverity: secSev,
		PartialFingerprints: map[string]string{
			"ruleId":   ruleID,
			"category": category,
		},
	}
}

func FindingFromRiskScore(score risk.PackageScore) *SarifResult {
	switch score.Grade {
	case risk.GradeA, risk.GradeB:
		return nil
	}

	var ruleID, level string
	var secSev float64

	switch score.Grade {
	case risk.GradeF:
		ruleID = "qsdev/dep-risk/failing"
		level = "error"
		secSev = 8.0
	case risk.GradeD:
		ruleID = "qsdev/dep-risk/high-risk"
		level = "warning"
		secSev = 6.0
	case risk.GradeC:
		ruleID = "qsdev/dep-risk/high-risk"
		level = "note"
		secSev = 4.0
	}

	msg := fmt.Sprintf("%s@%s: risk score %d (grade %s)", score.PackageName, score.PackageVersion, score.Score, score.Grade)

	return &SarifResult{
		RuleID:           ruleID,
		Level:            level,
		Message:          msg,
		ArtifactURI:      "",
		SecuritySeverity: secSev,
		PartialFingerprints: map[string]string{
			"ruleId":  ruleID,
			"package": score.PackageName + "@" + score.PackageVersion,
		},
	}
}

var eventTypeToTrustRuleID = map[string]string{
	"confused-deputy-blocked":     "qsdev/trust/confused-deputy-blocked",
	"tier3-injection":             "qsdev/trust/tier3-injection",
	"server-vulnerability":        "qsdev/trust/server-vulnerability",
	"tier3-post-fetch-suspicious": "qsdev/trust/tier3-post-fetch-suspicious",
}

func FindingFromTrustEvent(serverName, eventType, message string) SarifResult {
	ruleID, ok := eventTypeToTrustRuleID[eventType]
	if !ok {
		ruleID = "qsdev/trust/" + eventType
	}

	var level string
	var secSev float64
	for _, r := range AllRules {
		if r.ID == ruleID {
			level = r.DefaultLevel
			secSev = r.SecuritySeverity
			break
		}
	}
	if level == "" {
		level = "warning"
		secSev = 5.0
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
		},
	}
}
