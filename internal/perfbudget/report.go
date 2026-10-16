package perfbudget

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
)

// StepSummaryEnv names the file GitHub Actions renders as the job summary.
// GitHub sets it in every step, so tests that report point it at a
// temporary file.
const StepSummaryEnv = "GITHUB_STEP_SUMMARY"

const summaryHeader = "\n| budget | subject | payload | CPU p50 / p95 / max (ms) | wall p50 / p95 / max (ms) | limit (ms) | verdict |\n" +
	"|---|---|---|---|---|---|---|\n"

var (
	summaryMu sync.Mutex
	// headed holds the summary files this process has written a table
	// header to.
	headed = map[string]bool{}
)

// label is the verdict as the report's verdict column shows it.
func (r Result) label() string {
	switch r.Verdict {
	case VerdictPassMayRemoveSkip:
		return "pass (" + r.Workstream + " may remove its skip)"
	case VerdictSkip:
		return "skip " + r.Workstream
	case VerdictFail:
		if r.Workstream != "" {
			return "FAIL (over the " + r.Workstream + " skip's ceiling)"
		}
		return "FAIL"
	case VerdictPass:
	}
	return "pass"
}

func markdownRow(row Row, s Stats, r Result) string {
	limit := "-"
	if r.Limit > 0 {
		limit = ms(r.Limit)
	}
	if r.Ceiling > 0 {
		limit += " (skip ceiling " + ms(r.Ceiling) + ")"
	}
	cells := []string{
		row.Budget, row.Subject, row.Payload,
		ms(s.CPU.P50) + " / " + ms(s.CPU.P95) + " / " + ms(s.CPU.Max),
		ms(s.Wall.P50) + " / " + ms(s.Wall.P95) + " / " + ms(s.Wall.Max),
		limit, r.label(),
	}
	for i, c := range cells {
		cells[i] = strings.ReplaceAll(c, "|", `\|`)
	}
	return "| " + strings.Join(cells, " | ") + " |"
}

// Report appends row, its statistics and its verdict as a markdown table
// row to the file $GITHUB_STEP_SUMMARY names, starting the table on this
// process's first row there, or logs the row to t when that is unset.
func Report(t testing.TB, row Row, s Stats, r Result) {
	t.Helper()
	line := markdownRow(row, s, r)
	path := os.Getenv(StepSummaryEnv)
	if path == "" {
		t.Log(line)
		return
	}
	if err := appendSummary(path, line); err != nil {
		t.Errorf("perfbudget: writing the step summary: %v", err)
	}
}

func appendSummary(path, line string) (err error) {
	summaryMu.Lock()
	defer summaryMu.Unlock()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	text := line + "\n"
	if !headed[path] {
		text = summaryHeader + text
	}
	if _, err := f.WriteString(text); err != nil {
		return fmt.Errorf("appending to %s: %w", path, err)
	}
	headed[path] = true
	return nil
}
