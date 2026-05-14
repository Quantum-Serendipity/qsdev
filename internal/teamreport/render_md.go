package teamreport

import (
	"fmt"
	"sort"
	"strings"
)

// RenderMarkdown produces a Markdown dashboard from a TeamReport.
// The output is designed for GitHub-flavored Markdown rendering.
func RenderMarkdown(report *TeamReport) string {
	var b strings.Builder

	renderHeader(&b, report)
	renderOverviewTable(&b, report)
	renderProjectTable(&b, report)
	renderAttentionSection(&b, report)
	renderScoreChanges(&b, report)
	renderStaleScans(&b, report)
	renderFooter(&b, report)

	return b.String()
}

func renderHeader(b *strings.Builder, report *TeamReport) {
	b.WriteString("# Team Security Posture Dashboard\n\n")
	fmt.Fprintf(b, "**Date:** %s\n\n", report.GeneratedAt.Format("2006-01-02"))
}

func renderOverviewTable(b *strings.Builder, report *TeamReport) {
	b.WriteString("## Overview\n\n")
	b.WriteString("| Metric | Value |\n")
	b.WriteString("|--------|-------|\n")
	fmt.Fprintf(b, "| Projects | %d |\n", report.Summary.ProjectCount)
	fmt.Fprintf(b, "| Average Score | %.1f |\n", report.Summary.AverageScore)
	fmt.Fprintf(b, "| Median Score | %.1f |\n", report.Summary.MedianScore)
	fmt.Fprintf(b, "| Baseline Pass Rate | %.1f%% |\n", report.Summary.BaselinePassRate)
	fmt.Fprintf(b, "| Enhanced Pass Rate | %.1f%% |\n", report.Summary.EnhancedPassRate)
	fmt.Fprintf(b, "| Critical Vulnerabilities | %d |\n", report.Summary.TotalCriticalVulns)
	fmt.Fprintf(b, "| High Vulnerabilities | %d |\n", report.Summary.TotalHighVulns)
	fmt.Fprintf(b, "| Projects Needing Update | %d |\n", report.Summary.ProjectsNeedUpdate)
	b.WriteString("\n")
}

func renderProjectTable(b *strings.Builder, report *TeamReport) {
	b.WriteString("## Project Scores\n\n")
	b.WriteString("| Project | Score | Grade | Baseline | Enhanced | Vulns (C/H) | gdev Version | Last Scan |\n")
	b.WriteString("|---------|-------|-------|----------|----------|-------------|--------------|----------|\n")

	// Sort by score descending.
	sorted := make([]ProjectSummary, len(report.Projects))
	copy(sorted, report.Projects)
	sortProjectsByScoreDesc(sorted)

	for _, p := range sorted {
		baselineStatus := "PASS"
		if !p.Conformance.Baseline.Pass {
			baselineStatus = "FAIL"
		}
		enhancedStatus := "PASS"
		if !p.Conformance.Enhanced.Pass {
			enhancedStatus = "FAIL"
		}

		fmt.Fprintf(b, "| %s | %.1f | %s | %s | %s | %d/%d | %s | %s |\n",
			p.Name,
			p.Score.Total,
			p.Score.Grade,
			baselineStatus,
			enhancedStatus,
			p.VulnTotals.Critical,
			p.VulnTotals.High,
			p.GdevVersion,
			relativeTime(p.LastScan),
		)
	}
	b.WriteString("\n")
}

func renderAttentionSection(b *strings.Builder, report *TeamReport) {
	b.WriteString("## Attention Required\n\n")

	if len(report.Alerts) == 0 {
		b.WriteString("None\n\n")
		return
	}

	// Group alerts by severity, maintaining order: critical, high, medium.
	grouped := map[string][]PostureAlert{
		SeverityCritical: {},
		SeverityHigh:     {},
		SeverityMedium:   {},
	}
	for _, a := range report.Alerts {
		grouped[a.Severity] = append(grouped[a.Severity], a)
	}

	severityOrder := []string{SeverityCritical, SeverityHigh, SeverityMedium}
	severityLabels := map[string]string{
		SeverityCritical: "Critical",
		SeverityHigh:     "High",
		SeverityMedium:   "Medium",
	}

	for _, sev := range severityOrder {
		alerts := grouped[sev]
		if len(alerts) == 0 {
			continue
		}

		// Sort within severity by score ascending (we use project name
		// as a proxy since alerts don't carry scores directly; the alerts
		// are already sorted from generateAlerts).
		fmt.Fprintf(b, "### %s\n\n", severityLabels[sev])
		for _, a := range alerts {
			fmt.Fprintf(b, "- **%s**: %s\n  - Action: %s\n", a.Project, a.Message, a.Action)
		}
		b.WriteString("\n")
	}
}

func renderScoreChanges(b *strings.Builder, report *TeamReport) {
	if len(report.Trends) == 0 {
		return
	}

	// Build delta list: project -> (current - previous).
	type scoreDelta struct {
		Project  string
		Current  float64
		Previous float64
		Delta    float64
	}

	var deltas []scoreDelta

	// Build a quick lookup for current scores.
	currentScores := make(map[string]float64)
	for _, p := range report.Projects {
		currentScores[p.Name] = p.Score.Total
	}

	for _, t := range report.Trends {
		if len(t.DataPoints) < 2 {
			continue
		}

		current := t.DataPoints[len(t.DataPoints)-1]
		previous := t.DataPoints[len(t.DataPoints)-2]
		delta := current.Score - previous.Score

		if delta == 0 {
			continue
		}

		deltas = append(deltas, scoreDelta{
			Project:  t.Project,
			Current:  current.Score,
			Previous: previous.Score,
			Delta:    delta,
		})
	}

	if len(deltas) == 0 {
		return
	}

	// Sort by delta ascending (biggest drops first).
	sort.Slice(deltas, func(i, j int) bool {
		return deltas[i].Delta < deltas[j].Delta
	})

	b.WriteString("## Score Changes\n\n")
	b.WriteString("| Project | Previous | Current | Delta |\n")
	b.WriteString("|---------|----------|---------|-------|\n")
	for _, d := range deltas {
		sign := "+"
		if d.Delta < 0 {
			sign = ""
		}
		fmt.Fprintf(b, "| %s | %.1f | %.1f | %s%.1f |\n",
			d.Project, d.Previous, d.Current, sign, d.Delta)
	}
	b.WriteString("\n")
}

func renderStaleScans(b *strings.Builder, report *TeamReport) {
	var stale []ProjectSummary
	for _, p := range report.Projects {
		if p.Stale {
			stale = append(stale, p)
		}
	}

	if len(stale) == 0 {
		return
	}

	b.WriteString("## Stale Scans\n\n")
	b.WriteString("The following projects have not been scanned in over 7 days:\n\n")
	for _, p := range stale {
		fmt.Fprintf(b, "- **%s**: last scanned %s\n", p.Name, relativeTime(p.LastScan))
	}
	b.WriteString("\n")
}

func renderFooter(b *strings.Builder, report *TeamReport) {
	fmt.Fprintf(b, "---\n\n*Generated at %s by gdev team-report*\n",
		report.GeneratedAt.Format("2006-01-02T15:04:05Z"))
}
