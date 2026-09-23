package teamreport

import (
	"fmt"
	"strings"
	"testing"
)

// TestGenerateIssues_DeterministicOrder guards against ranging over the
// per-project alert map: issues must follow the report's project order on
// every run.
func TestGenerateIssues_DeterministicOrder(t *testing.T) {
	t.Parallel()
	report := &TeamReport{}
	var want []string
	for i := range 12 {
		name := fmt.Sprintf("project-%02d", i)
		report.Projects = append(report.Projects, ProjectSummary{Name: name})
		report.Alerts = append(report.Alerts, PostureAlert{Project: name, Severity: SeverityCritical})
		want = append(want, name)
	}

	for range 10 {
		issues := GenerateIssues(report, nil)
		var got []string
		for _, is := range issues {
			for _, name := range want {
				if strings.Contains(is.Title, name) {
					got = append(got, name)
				}
			}
		}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("issue order = %v, want %v", got, want)
		}
	}
}
