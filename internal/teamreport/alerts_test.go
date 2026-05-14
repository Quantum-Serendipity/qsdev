package teamreport

import (
	"testing"
	"time"
)

func TestAlertCriticalVulns(t *testing.T) {
	now := time.Now().UTC()
	projects := []ProjectSummary{
		projectSummaryHelper("vuln-project", 80, true, true, 3, 0, "v1.0.0", now),
	}

	alerts := generateAlerts(projects, AggregateOptions{}, nil)

	var found bool
	for _, a := range alerts {
		if a.Project == "vuln-project" && a.Severity == SeverityCritical {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected critical alert for project with critical vulns")
	}
}

func TestAlertBaselineFail(t *testing.T) {
	now := time.Now().UTC()
	projects := []ProjectSummary{
		projectSummaryHelper("fail-project", 80, false, false, 0, 0, "v1.0.0", now),
	}

	alerts := generateAlerts(projects, AggregateOptions{}, nil)

	var found bool
	for _, a := range alerts {
		if a.Project == "fail-project" && a.Severity == SeverityHigh {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected high alert for baseline failure")
	}
}

func TestAlertScoreDropLarge(t *testing.T) {
	now := time.Now().UTC()
	projects := []ProjectSummary{
		projectSummaryHelper("dropping", 60, true, true, 0, 0, "v1.0.0", now),
	}

	yesterday := time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02")
	history := &HistoryStore{
		Entries: map[string][]TrendPoint{
			"dropping": {{Date: yesterday, Score: 75}},
		},
	}

	alerts := generateAlerts(projects, AggregateOptions{}, history)

	var foundHigh bool
	for _, a := range alerts {
		if a.Project == "dropping" && a.Severity == SeverityHigh {
			foundHigh = true
			break
		}
	}

	if !foundHigh {
		t.Error("expected high alert for >10 point score drop")
	}
}

func TestAlertScoreDropModerate(t *testing.T) {
	now := time.Now().UTC()
	projects := []ProjectSummary{
		projectSummaryHelper("slight-drop", 72, true, true, 0, 0, "v1.0.0", now),
	}

	yesterday := time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02")
	history := &HistoryStore{
		Entries: map[string][]TrendPoint{
			"slight-drop": {{Date: yesterday, Score: 80}},
		},
	}

	alerts := generateAlerts(projects, AggregateOptions{}, history)

	var foundMedium bool
	for _, a := range alerts {
		if a.Project == "slight-drop" && a.Severity == SeverityMedium {
			foundMedium = true
			break
		}
	}

	if !foundMedium {
		t.Error("expected medium alert for >5 point score drop")
	}
}

func TestAlertOutdatedGdev(t *testing.T) {
	now := time.Now().UTC()
	projects := []ProjectSummary{
		projectSummaryHelper("old-gdev", 80, true, true, 0, 0, "v1.0.0", now),
	}

	alerts := generateAlerts(projects, AggregateOptions{GdevVersion: "v1.5.0"}, nil)

	var found bool
	for _, a := range alerts {
		if a.Project == "old-gdev" && a.Severity == SeverityMedium {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected medium alert for outdated gdev version")
	}
}

func TestAlertStale(t *testing.T) {
	staleTime := time.Now().UTC().Add(-10 * 24 * time.Hour)
	projects := []ProjectSummary{
		{
			Name:  "stale-project",
			Stale: true,
			Score: makeScore(80),
			Conformance: makeConformance(true, true),
			VulnTotals:  makeVulns(0, 0),
			GdevVersion: "v1.0.0",
			LastScan:    staleTime,
		},
	}

	alerts := generateAlerts(projects, AggregateOptions{}, nil)

	var found bool
	for _, a := range alerts {
		if a.Project == "stale-project" && a.Severity == SeverityMedium {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected medium alert for stale scan")
	}
}

func TestNoAlerts(t *testing.T) {
	now := time.Now().UTC()
	projects := []ProjectSummary{
		projectSummaryHelper("healthy", 95, true, true, 0, 0, "v1.0.0", now),
	}

	alerts := generateAlerts(projects, AggregateOptions{GdevVersion: "v1.0.0"}, nil)

	if len(alerts) != 0 {
		t.Errorf("expected no alerts for healthy project, got %d: %+v", len(alerts), alerts)
	}
}

func TestAlertSortOrder(t *testing.T) {
	now := time.Now().UTC()
	projects := []ProjectSummary{
		projectSummaryHelper("medium-only", 80, true, true, 0, 0, "v1.0.0", now.Add(-10*24*time.Hour)),
		projectSummaryHelper("critical-and-high", 40, false, false, 5, 0, "v1.0.0", now),
	}
	// Mark stale manually.
	projects[0].Stale = true

	alerts := generateAlerts(projects, AggregateOptions{}, nil)

	if len(alerts) < 2 {
		t.Fatalf("expected at least 2 alerts, got %d", len(alerts))
	}

	// First alert should be critical.
	if alerts[0].Severity != SeverityCritical {
		t.Errorf("expected first alert to be critical, got %s", alerts[0].Severity)
	}

	// Second alert should be high.
	if alerts[1].Severity != SeverityHigh {
		t.Errorf("expected second alert to be high, got %s", alerts[1].Severity)
	}
}
