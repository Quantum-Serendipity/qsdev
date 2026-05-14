package teamreport

import (
	"strings"
	"testing"
	"time"
)

func makeTeamReport() *TeamReport {
	now := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	return &TeamReport{
		SchemaVersion: "1.0.0",
		GeneratedAt:   now,
		Summary: TeamSummary{
			ProjectCount:       3,
			AverageScore:       75.0,
			MedianScore:        80.0,
			BaselinePassRate:   66.7,
			EnhancedPassRate:   33.3,
			TotalCriticalVulns: 2,
			TotalHighVulns:     5,
			ProjectsNeedUpdate: 1,
		},
		Projects: []ProjectSummary{
			projectSummaryHelper("alpha", 90.0, true, true, 0, 0, "v1.5.0", now),
			projectSummaryHelper("beta", 80.0, true, false, 0, 2, "v1.4.0", now),
			projectSummaryHelper("gamma", 55.0, false, false, 2, 3, "v1.0.0", now),
		},
		Alerts: []PostureAlert{
			{Project: "gamma", Severity: SeverityCritical, Message: "2 critical vulnerabilities found", Action: "Fix immediately"},
			{Project: "gamma", Severity: SeverityHigh, Message: "Baseline conformance check failed", Action: "Fix baseline"},
			{Project: "beta", Severity: SeverityMedium, Message: "gdev version outdated", Action: "Update gdev"},
		},
	}
}

func TestRenderMarkdownHeader(t *testing.T) {
	report := makeTeamReport()
	md := RenderMarkdown(report)

	if !strings.Contains(md, "# Team Security Posture Dashboard") {
		t.Error("expected header in markdown output")
	}

	if !strings.Contains(md, "**Date:** 2025-06-15") {
		t.Error("expected date in header")
	}
}

func TestRenderMarkdownOverviewTable(t *testing.T) {
	report := makeTeamReport()
	md := RenderMarkdown(report)

	if !strings.Contains(md, "## Overview") {
		t.Error("expected overview section")
	}

	if !strings.Contains(md, "| Projects | 3 |") {
		t.Error("expected project count in overview")
	}

	if !strings.Contains(md, "| Average Score | 75.0 |") {
		t.Error("expected average score in overview")
	}

	if !strings.Contains(md, "| Median Score | 80.0 |") {
		t.Error("expected median score in overview")
	}

	if !strings.Contains(md, "| Baseline Pass Rate | 66.7% |") {
		t.Error("expected baseline pass rate in overview")
	}

	if !strings.Contains(md, "| Enhanced Pass Rate | 33.3% |") {
		t.Error("expected enhanced pass rate in overview")
	}

	if !strings.Contains(md, "| Critical Vulnerabilities | 2 |") {
		t.Error("expected critical vulns in overview")
	}

	if !strings.Contains(md, "| High Vulnerabilities | 5 |") {
		t.Error("expected high vulns in overview")
	}

	if !strings.Contains(md, "| Projects Needing Update | 1 |") {
		t.Error("expected projects needing update in overview")
	}
}

func TestRenderMarkdownProjectTable(t *testing.T) {
	report := makeTeamReport()
	md := RenderMarkdown(report)

	if !strings.Contains(md, "## Project Scores") {
		t.Error("expected project scores section")
	}

	if !strings.Contains(md, "| Project | Score | Grade | Baseline | Enhanced | Vulns (C/H) | gdev Version | Last Scan |") {
		t.Error("expected project table header")
	}

	// Alpha should appear (sorted by score desc, alpha is highest).
	if !strings.Contains(md, "| alpha |") {
		t.Error("expected alpha in project table")
	}

	if !strings.Contains(md, "| beta |") {
		t.Error("expected beta in project table")
	}

	if !strings.Contains(md, "| gamma |") {
		t.Error("expected gamma in project table")
	}
}

func TestRenderMarkdownAttentionSection(t *testing.T) {
	report := makeTeamReport()
	md := RenderMarkdown(report)

	if !strings.Contains(md, "## Attention Required") {
		t.Error("expected attention required section")
	}

	if !strings.Contains(md, "### Critical") {
		t.Error("expected critical subsection")
	}

	if !strings.Contains(md, "### High") {
		t.Error("expected high subsection")
	}

	if !strings.Contains(md, "### Medium") {
		t.Error("expected medium subsection")
	}

	if !strings.Contains(md, "**gamma**") {
		t.Error("expected gamma in attention section")
	}
}

func TestRenderMarkdownEmptyAlerts(t *testing.T) {
	report := makeTeamReport()
	report.Alerts = nil
	md := RenderMarkdown(report)

	if !strings.Contains(md, "## Attention Required") {
		t.Error("expected attention required section even with no alerts")
	}

	if !strings.Contains(md, "None") {
		t.Error("expected 'None' when no alerts present")
	}
}

func TestRenderMarkdownFooter(t *testing.T) {
	report := makeTeamReport()
	md := RenderMarkdown(report)

	if !strings.Contains(md, "Generated at 2025-06-15T10:00:00Z by gdev team-report") {
		t.Error("expected footer with timestamp")
	}
}

func TestRenderMarkdownScoreChanges(t *testing.T) {
	report := makeTeamReport()
	report.Trends = []ProjectTrend{
		{
			Project: "alpha",
			DataPoints: []TrendPoint{
				{Date: "2025-06-14", Score: 85.0},
				{Date: "2025-06-15", Score: 90.0},
			},
		},
		{
			Project: "gamma",
			DataPoints: []TrendPoint{
				{Date: "2025-06-14", Score: 70.0},
				{Date: "2025-06-15", Score: 55.0},
			},
		},
	}

	md := RenderMarkdown(report)

	if !strings.Contains(md, "## Score Changes") {
		t.Error("expected score changes section when trends present")
	}

	if !strings.Contains(md, "alpha") {
		t.Error("expected alpha in score changes")
	}

	if !strings.Contains(md, "gamma") {
		t.Error("expected gamma in score changes")
	}
}

func TestRenderMarkdownNoScoreChanges(t *testing.T) {
	report := makeTeamReport()
	report.Trends = nil
	md := RenderMarkdown(report)

	if strings.Contains(md, "## Score Changes") {
		t.Error("should not show score changes when no trends")
	}
}

func TestRenderMarkdownStaleScans(t *testing.T) {
	report := makeTeamReport()
	report.Projects[2].Stale = true // gamma is stale

	md := RenderMarkdown(report)

	if !strings.Contains(md, "## Stale Scans") {
		t.Error("expected stale scans section when stale projects exist")
	}

	if !strings.Contains(md, "**gamma**") {
		t.Error("expected gamma in stale scans section")
	}
}

func TestRenderMarkdownNoStaleScans(t *testing.T) {
	report := makeTeamReport()
	// No stale projects.
	md := RenderMarkdown(report)

	if strings.Contains(md, "## Stale Scans") {
		t.Error("should not show stale scans when none are stale")
	}
}
