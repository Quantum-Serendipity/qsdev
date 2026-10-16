package perfbudget

import (
	"os"
	"strings"
	"testing"
	"time"
)

// TestReportStepSummary cannot be parallel: it sets GITHUB_STEP_SUMMARY.
func TestReportStepSummary(t *testing.T) {
	stats := statsAt(30 * time.Millisecond)
	pass := Result{Verdict: VerdictPass, Limit: 50 * time.Millisecond}
	skip := Result{Verdict: VerdictSkip, Workstream: "XD-WS3", Limit: 50 * time.Millisecond, Ceiling: 75 * time.Millisecond}

	t.Run("appends rows under one header", func(t *testing.T) {
		path := summaryFile(t)
		Report(t, Row{Budget: "cli-startup", Subject: "qsdev version"}, stats, pass)
		Report(t, Row{Budget: "hook-script-miss", Subject: "a|b", Payload: "bash_ls"}, stats, skip)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading the step summary: %v", err)
		}
		text := string(data)
		if n := strings.Count(text, "| budget |"); n != 1 {
			t.Errorf("the step summary has %d headers, want 1:\n%s", n, text)
		}
		lines := strings.Split(strings.TrimSpace(text), "\n")
		if len(lines) != 4 {
			t.Fatalf("the step summary has %d lines, want a header, a rule and two rows:\n%s", len(lines), text)
		}
		for _, want := range []string{
			"| cli-startup | qsdev version |  | 15.0 / 30.0 / 31.0 | 15.0 / 30.0 / 31.0 | 50.0 | pass |",
			`| hook-script-miss | a\|b | bash_ls | 15.0 / 30.0 / 31.0 | 15.0 / 30.0 / 31.0 | 50.0 (skip ceiling 75.0) | skip XD-WS3 |`,
		} {
			if !strings.Contains(text, want) {
				t.Errorf("the step summary lacks row %q:\n%s", want, text)
			}
		}
	})

	t.Run("falls back to the test log", func(t *testing.T) {
		t.Setenv(StepSummaryEnv, "")
		rec := &recorder{TB: t}
		Report(rec, Row{Budget: "cli-startup", Subject: "qsdev version"}, stats, pass)
		if len(rec.logs) != 1 || !strings.Contains(rec.logs[0], "| cli-startup | qsdev version |") {
			t.Errorf("Report logged %q, want one markdown row", rec.logs)
		}
	})
}

func TestResultLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		r    Result
		want string
	}{
		{Result{Verdict: VerdictPass}, "pass"},
		{Result{Verdict: VerdictPassMayRemoveSkip, Workstream: "XD-WS3"}, "pass (XD-WS3 may remove its skip)"},
		{Result{Verdict: VerdictSkip, Workstream: "XD-WS3"}, "skip XD-WS3"},
		{Result{Verdict: VerdictFail}, "FAIL"},
		{Result{Verdict: VerdictFail, Workstream: "XD-WS3"}, "FAIL (over the XD-WS3 skip's ceiling)"},
	}
	for _, tt := range tests {
		if got := tt.r.label(); got != tt.want {
			t.Errorf("label(%+v) = %q, want %q", tt.r, got, tt.want)
		}
	}
}
