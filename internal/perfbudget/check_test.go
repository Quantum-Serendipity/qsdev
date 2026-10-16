package perfbudget

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// judgeDoc is a budgets document whose limits are easy to scale by hand:
// 40 ms × allowance 1.25 is 50 ms at calibration 1, a covered row may
// remove its skip at 40 / 1.25 = 32 ms, and XD-WS3's ceiling of 80 ms
// scales to 100 ms.
const judgeDoc = validHeader + `budgets:
  - id: plain
    surface: no skips
    cpu_p95_ms: 40
  - id: tracked
    surface: three skips, the first matching one wins
    cpu_p95_ms: 40
    skips:
      - workstream: XD-WS3
        subjects: [selfprotect]
        payloads: [bash_ls]
        reason: start-up
        ceiling_ms: 80
      - workstream: XD-WS2
        subjects: [selfprotect]
        reason: any payload
        ceiling_ms: 100
      - workstream: U17-WS8
        payloads: [edit_2k]
        reason: any subject
        ceiling_ms: 100
  - id: footprint
    surface: a size
    max_bytes: 10
    skips:
      - {workstream: XD-WS11, reason: owner-only}
`

func judgeBudgets(t *testing.T) Budgets {
	t.Helper()
	b, err := Parse([]byte(judgeDoc))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return b
}

// statsAt returns twenty-run stats whose CPU p95 is p95.
func statsAt(p95 time.Duration) Stats {
	return Stats{
		N:    20,
		CPU:  Dist{P50: p95 / 2, P95: p95, Max: p95 + time.Millisecond},
		Wall: Dist{P50: p95 / 2, P95: p95, Max: p95 + time.Millisecond},
	}
}

func TestJudge(t *testing.T) {
	t.Parallel()
	b := judgeBudgets(t)
	ms := func(v float64) time.Duration { return time.Duration(v * float64(time.Millisecond)) }
	tests := []struct {
		name    string
		row     Row
		stats   Stats
		factor  float64
		want    Verdict
		owner   string
		limit   time.Duration
		ceiling time.Duration
		message []string
		prefix  string
	}{
		{
			name: "at the limit passes", row: Row{Budget: "plain", Subject: "selfprotect", Payload: "bash_ls"},
			stats: statsAt(ms(50)), factor: 1, want: VerdictPass, limit: ms(50),
			message: []string{"selfprotect", "bash_ls", "within budget"},
		},
		{
			name: "over the limit fails", row: Row{Budget: "plain", Subject: "selfprotect", Payload: "bash_ls"},
			stats: statsAt(ms(50.1)), factor: 1, want: VerdictFail, limit: ms(50),
			message: []string{"selfprotect", "bash_ls", "CPU p95 50.1 ms", "50.0 ms", "budget 40.0 ms × calibration 1.00 × allowance 1.25", "20 runs"},
		},
		{
			name: "a fail without a payload names the subject", row: Row{Budget: "plain", Subject: "qsdev version"},
			stats: statsAt(ms(60)), factor: 1, want: VerdictFail, limit: ms(50),
			message: []string{"qsdev version", "plain"},
		},
		{
			name: "the factor raises the limit", row: Row{Budget: "plain", Subject: "s", Payload: "p"},
			stats: statsAt(ms(99)), factor: 2, want: VerdictPass, limit: ms(100),
		},
		{
			name: "the factor lowers the limit", row: Row{Budget: "plain", Subject: "s", Payload: "p"},
			stats: statsAt(ms(26)), factor: 0.5, want: VerdictFail, limit: ms(25),
			message: []string{"× calibration 0.50 × allowance 1.25"},
		},
		{
			name: "over budget with a skip skips", row: Row{Budget: "tracked", Subject: "selfprotect", Payload: "bash_ls"},
			stats: statsAt(ms(80)), factor: 1, want: VerdictSkip, owner: "XD-WS3", limit: ms(50), ceiling: ms(100),
			prefix: "XD-WS3: ", message: []string{"selfprotect", "bash_ls", "CPU p95 80.0 ms", "start-up"},
		},
		{
			name: "at the skip's ceiling skips", row: Row{Budget: "tracked", Subject: "selfprotect", Payload: "bash_ls"},
			stats: statsAt(ms(100)), factor: 1, want: VerdictSkip, owner: "XD-WS3", limit: ms(50), ceiling: ms(100),
			prefix: "XD-WS3: ",
		},
		{
			name: "over the skip's ceiling fails", row: Row{Budget: "tracked", Subject: "selfprotect", Payload: "bash_ls"},
			stats: statsAt(ms(100.1)), factor: 1, want: VerdictFail, owner: "XD-WS3", limit: ms(50), ceiling: ms(100),
			message: []string{"selfprotect", "bash_ls", "over the ceiling of its XD-WS3 skip", "CPU p95 100.1 ms > ceiling 100.0 ms", "ceiling 80.0 ms × calibration 1.00 × allowance 1.25", "20 runs"},
		},
		{
			name: "the factor scales the ceiling", row: Row{Budget: "tracked", Subject: "selfprotect", Payload: "bash_ls"},
			stats: statsAt(ms(199)), factor: 2, want: VerdictSkip, owner: "XD-WS3", limit: ms(100), ceiling: ms(200),
			prefix: "XD-WS3: ",
		},
		{
			name: "the factor lowers the ceiling", row: Row{Budget: "tracked", Subject: "selfprotect", Payload: "bash_ls"},
			stats: statsAt(ms(51)), factor: 0.5, want: VerdictFail, owner: "XD-WS3", limit: ms(25), ceiling: ms(50),
			message: []string{"over the ceiling"},
		},
		{
			name: "the first matching skip wins", row: Row{Budget: "tracked", Subject: "selfprotect", Payload: "edit_2k"},
			stats: statsAt(ms(110)), factor: 1, want: VerdictSkip, owner: "XD-WS2", limit: ms(50), ceiling: ms(125),
			prefix: "XD-WS2: ", message: []string{"any payload"},
		},
		{
			name: "a payload filter matches any subject", row: Row{Budget: "tracked", Subject: "package-guard", Payload: "edit_2k"},
			stats: statsAt(ms(80)), factor: 1, want: VerdictSkip, owner: "U17-WS8", limit: ms(50), ceiling: ms(125),
			prefix: "U17-WS8: ",
		},
		{
			name: "a row no skip covers fails", row: Row{Budget: "tracked", Subject: "package-guard", Payload: "bash_ls"},
			stats: statsAt(ms(80)), factor: 1, want: VerdictFail, limit: ms(50),
			message: []string{"package-guard", "bash_ls"},
		},
		{
			name: "within budget with the allowance to spare may remove the skip", row: Row{Budget: "tracked", Subject: "selfprotect", Payload: "bash_ls"},
			stats: statsAt(ms(32)), factor: 1, want: VerdictPassMayRemoveSkip, owner: "XD-WS3", limit: ms(50), ceiling: ms(100),
			prefix: "XD-WS3 may remove its skip: ", message: []string{"selfprotect", "bash_ls", "within budget with the allowance to spare"},
		},
		{
			// 32.1 ms is within the 50 ms limit, but × 1.25 it is over the
			// 40 ms budget: a row this close would not land unskipped, so
			// its skip stays and the verdict does not flap between runs.
			name: "within the limit but not the allowance keeps the skip", row: Row{Budget: "tracked", Subject: "selfprotect", Payload: "bash_ls"},
			stats: statsAt(ms(32.1)), factor: 1, want: VerdictSkip, owner: "XD-WS3", limit: ms(50), ceiling: ms(100),
			prefix: "XD-WS3: ", message: []string{"within it by less than the allowance", "CPU p95 32.1 ms <= limit 50.0 ms"},
		},
		{
			name: "the factor scales the margin", row: Row{Budget: "tracked", Subject: "selfprotect", Payload: "bash_ls"},
			stats: statsAt(ms(64)), factor: 2, want: VerdictPassMayRemoveSkip, owner: "XD-WS3", limit: ms(100), ceiling: ms(200),
			prefix: "XD-WS3 may remove its skip: ",
		},
		{
			name: "an unknown budget fails", row: Row{Budget: "nope", Subject: "s"},
			stats: statsAt(ms(1)), factor: 1, want: VerdictFail, message: []string{`no budget "nope"`},
		},
		{
			name: "a size budget is not measured here", row: Row{Budget: "footprint", Subject: "s"},
			stats: statsAt(ms(1)), factor: 1, want: VerdictFail, message: []string{"not a cpu_p95_ms budget"},
		},
		{
			name: "a zero factor fails", row: Row{Budget: "plain", Subject: "s"},
			stats: statsAt(ms(1)), factor: 0, want: VerdictFail, message: []string{"calibration factor"},
		},
		{
			name: "no runs fails", row: Row{Budget: "plain", Subject: "s"},
			stats: Stats{}, factor: 1, want: VerdictFail, message: []string{"no runs"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := b.Judge(tt.row, tt.stats, tt.factor)
			if r.Verdict != tt.want {
				t.Errorf("Verdict = %v, want %v (message %q)", r.Verdict, tt.want, r.Message)
			}
			if r.Workstream != tt.owner {
				t.Errorf("Workstream = %q, want %q", r.Workstream, tt.owner)
			}
			if r.Limit != tt.limit {
				t.Errorf("Limit = %v, want %v", r.Limit, tt.limit)
			}
			if r.Ceiling != tt.ceiling {
				t.Errorf("Ceiling = %v, want %v", r.Ceiling, tt.ceiling)
			}
			if !strings.HasPrefix(r.Message, tt.prefix) {
				t.Errorf("Message = %q, want prefix %q", r.Message, tt.prefix)
			}
			for _, w := range tt.message {
				if !strings.Contains(r.Message, w) {
					t.Errorf("Message = %q, want it to mention %q", r.Message, w)
				}
			}
		})
	}
}

// recorder is a testing.TB that records what Check does to it instead of
// failing or skipping the test running it.
type recorder struct {
	testing.TB
	errors, skips, logs []string
}

func (r *recorder) Errorf(format string, args ...any) {
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

func (r *recorder) Skipf(format string, args ...any) {
	r.skips = append(r.skips, fmt.Sprintf(format, args...))
}

func (r *recorder) Logf(format string, args ...any) {
	r.logs = append(r.logs, fmt.Sprintf(format, args...))
}

func (r *recorder) Log(args ...any) { r.logs = append(r.logs, fmt.Sprint(args...)) }

// summaryFile points GITHUB_STEP_SUMMARY, which GitHub sets in every CI
// step, at a fresh file and returns its path.
func summaryFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "summary.md")
	t.Setenv(StepSummaryEnv, path)
	return path
}

// TestCheckAppliesVerdict cannot be parallel: it sets GITHUB_STEP_SUMMARY.
func TestCheckAppliesVerdict(t *testing.T) {
	b := judgeBudgets(t)
	summary := summaryFile(t)
	tracked := Row{Budget: "tracked", Subject: "selfprotect", Payload: "bash_ls"}

	t.Run("skip ends a real subtest", func(t *testing.T) {
		var sub *testing.T
		t.Run("row", func(t *testing.T) {
			sub = t
			b.Check(t, tracked, statsAt(80*time.Millisecond), 1)
			t.Error("Check returned from an over-budget tracked row")
		})
		if !sub.Skipped() {
			t.Error("the over-budget tracked row did not skip its subtest")
		}
	})

	tests := []struct {
		name                string
		row                 Row
		p95                 time.Duration
		errors, skips, logs int
		want                Verdict
	}{
		{name: "fail", row: Row{Budget: "plain", Subject: "selfprotect", Payload: "bash_ls"}, p95: 80 * time.Millisecond, errors: 1, want: VerdictFail},
		{name: "skip", row: tracked, p95: 80 * time.Millisecond, skips: 1, want: VerdictSkip},
		{name: "fail over the skip's ceiling", row: tracked, p95: 120 * time.Millisecond, errors: 1, want: VerdictFail},
		// A covered row skips whether or not its owner may remove the skip,
		// so noise around the budget cannot change which rows skip.
		{name: "pass may remove the skip", row: tracked, p95: 10 * time.Millisecond, skips: 1, want: VerdictPassMayRemoveSkip},
		{name: "within the limit but not the allowance", row: tracked, p95: 45 * time.Millisecond, skips: 1, want: VerdictSkip},
		{name: "pass", row: Row{Budget: "plain", Subject: "selfprotect", Payload: "bash_ls"}, p95: 10 * time.Millisecond, want: VerdictPass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &recorder{TB: t}
			r := b.Check(rec, tt.row, statsAt(tt.p95), 1)
			if r.Verdict != tt.want {
				t.Errorf("Verdict = %v, want %v", r.Verdict, tt.want)
			}
			if len(rec.errors) != tt.errors || len(rec.skips) != tt.skips || len(rec.logs) != tt.logs {
				t.Errorf("Check made %d errors, %d skips and %d logs (%q %q %q), want %d, %d and %d",
					len(rec.errors), len(rec.skips), len(rec.logs), rec.errors, rec.skips, rec.logs, tt.errors, tt.skips, tt.logs)
			}
			for _, got := range [][]string{rec.errors, rec.skips, rec.logs} {
				for _, m := range got {
					if m != r.Message {
						t.Errorf("Check reported %q, want the verdict's message %q", m, r.Message)
					}
				}
			}
		})
	}

	data, err := os.ReadFile(summary)
	if err != nil {
		t.Fatalf("reading the step summary: %v", err)
	}
	// One row per Check: the real subtest plus the recorded ones.
	if rows, want := strings.Count(string(data), "| tracked |")+strings.Count(string(data), "| plain |"), 1+len(tests); rows != want {
		t.Errorf("the step summary has %d rows, want %d:\n%s", rows, want, data)
	}
}
