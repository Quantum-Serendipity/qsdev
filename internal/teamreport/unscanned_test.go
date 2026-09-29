package teamreport

import (
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// unscannedSummary summarizes a posture report whose dependencies were never
// scanned: the dependency score is unknown and the baseline, whose
// no-critical-vulns check cannot be evaluated, is unknown.
func unscannedSummary(t *testing.T) ProjectSummary {
	t.Helper()
	deps := posture.ComputeDepScore([]posture.EcosystemStatus{{Name: "go", Detected: true, LockFile: "go.sum"}})
	generated := types.GeneratedState{Files: map[string]types.FileState{
		"CLAUDE.md": {}, ".claude/settings.json": {}, ".pre-commit-config.yaml": {},
	}}
	conformance := posture.EvaluateConformance(posture.DefenseCoverage{}, deps, nil, generated)
	report := &posture.PostureReport{
		ProjectName:  "unscanned",
		QsdevVersion: "v1.0.0",
		Score:        posture.ComputeAggregateScore(90, 90, deps.Score),
		Conformance:  conformance,
		Dependencies: deps,
	}
	return summarizeProject(report, time.Now())
}

// TestUnscannedProjectReadsUnknown pins F328 across the team report: an
// unscanned member's dependency health and baseline read unknown, it is not
// counted as passing baseline, and it raises the not-scanned alert rather
// than a baseline-failure alert.
func TestUnscannedProjectReadsUnknown(t *testing.T) {
	t.Parallel()
	p := unscannedSummary(t)
	if p.Conformance.Baseline.Verdict() != posture.CheckUnknown {
		t.Fatalf("baseline verdict = %q, want unknown", p.Conformance.Baseline.Verdict())
	}

	t.Run("summary", func(t *testing.T) {
		t.Parallel()
		s := computeSummary([]ProjectSummary{p}, AggregateOptions{})
		if s.BaselinePassRate != 0 {
			t.Errorf("BaselinePassRate = %.1f, want 0 (unknown is not a pass)", s.BaselinePassRate)
		}
	})
	t.Run("alerts", func(t *testing.T) {
		t.Parallel()
		alerts := generateAlerts([]ProjectSummary{p}, AggregateOptions{QsdevVersion: "v1.0.0"}, nil)
		var notScanned bool
		for _, a := range alerts {
			if strings.Contains(a.Message, "Baseline conformance check failed") {
				t.Errorf("unexpected baseline-failure alert for an unknown baseline: %+v", a)
			}
			notScanned = notScanned || strings.Contains(a.Message, "not scanned")
		}
		if !notScanned {
			t.Errorf("expected a not-scanned alert, got %+v", alerts)
		}
	})
	t.Run("issue body", func(t *testing.T) {
		t.Parallel()
		body := buildIssueBody(p, nil, nil)
		for _, want := range []string{
			"| Dependency Health | n/a (not scanned) |",
			"| Baseline Conformance | UNKNOWN |",
		} {
			if !strings.Contains(body, want) {
				t.Errorf("issue body lacks %q:\n%s", want, body)
			}
		}
	})
	t.Run("markdown", func(t *testing.T) {
		t.Parallel()
		md := RenderMarkdown(&TeamReport{Projects: []ProjectSummary{p}})
		if !strings.Contains(md, "| unscanned | 90.0 | A- | UNKNOWN |") {
			t.Errorf("project row does not show an UNKNOWN baseline:\n%s", md)
		}
	})
}
