package teamreport

import (
	"strings"
	"testing"
	"time"
)

func TestIssueTitleWithDelta(t *testing.T) {
	p := projectSummaryHelper("myproject", 65, true, true, 0, 0, "v1.0.0", time.Now())

	yesterday := time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02")
	history := &HistoryStore{
		Entries: map[string][]TrendPoint{
			"myproject": {{Date: yesterday, Score: 80.0}},
		},
	}

	title := buildIssueTitle(p, history)

	if !strings.Contains(title, "[qsdev] Security posture degraded: myproject") {
		t.Errorf("unexpected title format: %s", title)
	}

	if !strings.Contains(title, "65/100") {
		t.Errorf("expected score in title: %s", title)
	}

	if !strings.Contains(title, "-15") {
		t.Errorf("expected delta in title: %s", title)
	}
}

func TestIssueTitleWithoutDelta(t *testing.T) {
	p := projectSummaryHelper("myproject", 65, true, true, 0, 0, "v1.0.0", time.Now())

	title := buildIssueTitle(p, nil)

	if !strings.Contains(title, "[qsdev] Security posture degraded: myproject (65/100)") {
		t.Errorf("unexpected title format: %s", title)
	}

	// Should not contain a delta like +0 or -0.
	if strings.Contains(title, "+") || strings.Contains(title, "-") {
		t.Errorf("should not contain delta when no history: %s", title)
	}
}

func TestIssueBodyContainsTable(t *testing.T) {
	p := projectSummaryHelper("myproject", 65, false, false, 3, 5, "v1.0.0", time.Now())

	alerts := []PostureAlert{
		{Severity: SeverityCritical, Message: "3 critical vulnerabilities", Action: "Fix vulns"},
		{Severity: SeverityHigh, Message: "Baseline failed", Action: "Fix baseline"},
	}

	body := buildIssueBody(p, alerts, nil)

	if !strings.Contains(body, "| Metric | Value |") {
		t.Error("expected metric table in issue body")
	}

	if !strings.Contains(body, "| Score | 65.0/100") {
		t.Error("expected score in metric table")
	}

	if !strings.Contains(body, "| Critical Vulnerabilities | 3 |") {
		t.Error("expected critical vulns in metric table")
	}

	if !strings.Contains(body, "| High Vulnerabilities | 5 |") {
		t.Error("expected high vulns in metric table")
	}

	if !strings.Contains(body, "FAIL") {
		t.Error("expected baseline FAIL in metric table")
	}

	if !strings.Contains(body, "## Issues Found") {
		t.Error("expected issues found section")
	}

	if !strings.Contains(body, "## Recommended Actions") {
		t.Error("expected recommended actions section")
	}
}

func TestIssueBodyWithHistory(t *testing.T) {
	p := projectSummaryHelper("myproject", 65, true, true, 0, 0, "v1.0.0", time.Now())

	yesterday := time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02")
	history := &HistoryStore{
		Entries: map[string][]TrendPoint{
			"myproject": {{Date: yesterday, Score: 80.0}},
		},
	}

	alerts := []PostureAlert{
		{Severity: SeverityHigh, Message: "Score dropped", Action: "Investigate"},
	}

	body := buildIssueBody(p, alerts, history)

	if !strings.Contains(body, "Previous Score") {
		t.Error("expected previous score in body when history available")
	}
}

func TestGenerateIssuesOnlyCriticalAndHigh(t *testing.T) {
	now := time.Now().UTC()
	report := &TeamReport{
		Projects: []ProjectSummary{
			projectSummaryHelper("critical-proj", 40, false, false, 5, 0, "v1.0.0", now),
			projectSummaryHelper("medium-only", 70, true, true, 0, 0, "v1.0.0", now),
		},
		Alerts: []PostureAlert{
			{Project: "critical-proj", Severity: SeverityCritical, Message: "Critical vulns", Action: "Fix"},
			{Project: "critical-proj", Severity: SeverityHigh, Message: "Baseline fail", Action: "Fix baseline"},
			{Project: "medium-only", Severity: SeverityMedium, Message: "Outdated qsdev", Action: "Update"},
		},
	}

	issues := GenerateIssues(report, nil)

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue (only critical+high), got %d", len(issues))
	}

	if issues[0].Title == "" {
		t.Error("expected non-empty title")
	}
}

func TestGenerateIssuesDedup(t *testing.T) {
	now := time.Now().UTC()
	report := &TeamReport{
		Projects: []ProjectSummary{
			projectSummaryHelper("proj-a", 40, false, false, 5, 3, "v1.0.0", now),
		},
		Alerts: []PostureAlert{
			{Project: "proj-a", Severity: SeverityCritical, Message: "Critical vulns", Action: "Fix vulns"},
			{Project: "proj-a", Severity: SeverityHigh, Message: "Baseline fail", Action: "Fix baseline"},
		},
	}

	issues := GenerateIssues(report, nil)

	// Should be deduped: one issue per project.
	if len(issues) != 1 {
		t.Errorf("expected 1 issue (deduped per project), got %d", len(issues))
	}
}

func TestGenerateIssuesNoAlerts(t *testing.T) {
	report := &TeamReport{
		Alerts: nil,
	}

	issues := GenerateIssues(report, nil)
	if len(issues) != 0 {
		t.Errorf("expected 0 issues when no alerts, got %d", len(issues))
	}
}

func TestIssueLabels(t *testing.T) {
	now := time.Now().UTC()
	report := &TeamReport{
		Projects: []ProjectSummary{
			projectSummaryHelper("proj", 40, false, false, 5, 0, "v1.0.0", now),
		},
		Alerts: []PostureAlert{
			{Project: "proj", Severity: SeverityCritical, Message: "Vulns", Action: "Fix"},
		},
	}

	issues := GenerateIssues(report, nil)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	if len(issues[0].Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(issues[0].Labels))
	}

	hasSecurityLabel := false
	hasPostureLabel := false
	for _, l := range issues[0].Labels {
		if l == "security" {
			hasSecurityLabel = true
		}
		if l == "qsdev-posture" {
			hasPostureLabel = true
		}
	}

	if !hasSecurityLabel {
		t.Error("expected 'security' label")
	}
	if !hasPostureLabel {
		t.Error("expected 'qsdev-posture' label")
	}
}
