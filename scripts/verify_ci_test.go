package scripts_test

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/shelltest"
)

// These tests run scripts/verify-ci.sh against a stub gh that prints canned
// check-run pages, so no GitHub API request is made.

const (
	verifyRepo = "example/repo"
	verifySHA  = "0123456789abcdef0123456789abcdef01234567"
)

// verifyRequired is the required-checks file the tests gate on, with a
// comment and CRLF line endings as a Windows checkout could leave them.
const verifyRequired = "# Required checks.\r\nlint\r\n\r\ntest (ubuntu-latest)\r\n"

// checkRun is one entry of the check-runs API's check_runs array.
type checkRun struct {
	Name       string  `json:"name"`
	Status     string  `json:"status"`
	Conclusion *string `json:"conclusion"`
}

func completed(name, conclusion string) checkRun {
	return checkRun{Name: name, Status: "completed", Conclusion: &conclusion}
}

// checkRunPages renders pages the way `gh api --paginate` prints them: one
// JSON object per page, back to back.
func checkRunPages(t *testing.T, pages ...[]checkRun) string {
	t.Helper()

	var b strings.Builder
	for _, runs := range pages {
		page, err := json.Marshal(map[string]any{"total_count": len(runs), "check_runs": runs})
		if err != nil {
			t.Fatal(err)
		}
		b.Write(page)
		b.WriteByte('\n')
	}
	return b.String()
}

// runVerifyCI runs verify-ci.sh with gh stubbed by the given stub. With
// crlfJQ, jq is wrapped to end every line with CRLF, as a native Windows jq
// does, so that behaviour is exercised on every OS.
func runVerifyCI(t *testing.T, required string, gh shelltest.Stub, crlfJQ bool) shelltest.Result {
	t.Helper()
	jq, err := exec.LookPath("jq")
	if err != nil {
		t.Skip("jq is not installed; the CI runner provides it")
	}
	script, err := filepath.Abs("verify-ci.sh")
	if err != nil {
		t.Fatal(err)
	}
	dir := shelltest.WriteTree(t, t.TempDir(), map[string]string{"required-checks.txt": required})
	cmd := "bash " + shelltest.QuotePath(script) + " " + verifyRepo + " " + verifySHA + " " +
		shelltest.QuotePath(filepath.Join(dir, "required-checks.txt"))
	stubs := map[string]shelltest.Stub{"gh": gh}
	if crlfJQ {
		stubs["jq"] = shelltest.Stub{Script: shelltest.QuotePath(jq) +
			` "$@" | while IFS= read -r line; do printf '%s\r\n' "$line"; done`}
	}
	return shelltest.Run(t, dir, cmd, stubs)
}

// TestVerifyCI covers the U26-06 release gate: verify-ci.sh passes only when
// every required check has a successful latest run on the commit, across
// every page of the API's results, and names each one that has not.
func TestVerifyCI(t *testing.T) {
	t.Parallel()

	pending := checkRun{Name: "test (ubuntu-latest)", Status: "in_progress"}
	tests := []struct {
		name     string
		required string
		gh       func(t *testing.T) shelltest.Stub
		wantExit int
		// wantOut must all appear in the output; notOut must not.
		wantOut []string
		notOut  []string
	}{
		{
			name: "allSuccess",
			gh: func(t *testing.T) shelltest.Stub {
				return shelltest.Stub{Stdout: checkRunPages(t, []checkRun{
					completed("lint", "success"), completed("test (ubuntu-latest)", "success"),
				})}
			},
			wantOut: []string{"2 required checks succeeded"},
		},
		{
			name: "failureNamed",
			gh: func(t *testing.T) shelltest.Stub {
				return shelltest.Stub{Stdout: checkRunPages(t, []checkRun{
					completed("lint", "success"), completed("test (ubuntu-latest)", "failure"),
				})}
			},
			wantExit: 1,
			wantOut:  []string{`"test (ubuntu-latest)" concluded failure`},
			notOut:   []string{`"lint"`},
		},
		{
			name: "missingCheck",
			gh: func(t *testing.T) shelltest.Stub {
				return shelltest.Stub{Stdout: checkRunPages(t, []checkRun{completed("lint", "success")})}
			},
			wantExit: 1,
			wantOut:  []string{`"test (ubuntu-latest)" has no check run`},
		},
		{
			name: "pending",
			gh: func(t *testing.T) shelltest.Stub {
				return shelltest.Stub{Stdout: checkRunPages(t, []checkRun{completed("lint", "success"), pending})}
			},
			wantExit: 1,
			wantOut:  []string{`"test (ubuntu-latest)" is in_progress`},
		},
		{
			name: "neutralSkipped",
			gh: func(t *testing.T) shelltest.Stub {
				return shelltest.Stub{Stdout: checkRunPages(t, []checkRun{
					completed("lint", "neutral"), completed("test (ubuntu-latest)", "skipped"),
				})}
			},
			wantExit: 1,
			wantOut:  []string{`"lint" concluded neutral`, `"test (ubuntu-latest)" concluded skipped`},
		},
		{
			name: "paginated",
			gh: func(t *testing.T) shelltest.Stub {
				return shelltest.Stub{Stdout: checkRunPages(t,
					[]checkRun{completed("lint", "success"), completed("other", "success")},
					[]checkRun{completed("test (ubuntu-latest)", "success")},
				)}
			},
			wantOut: []string{"2 required checks succeeded"},
		},
		{
			name: "paginatedFailureOnLastPage",
			gh: func(t *testing.T) shelltest.Stub {
				return shelltest.Stub{Stdout: checkRunPages(t,
					[]checkRun{completed("lint", "success")},
					[]checkRun{completed("test (ubuntu-latest)", "cancelled")},
				)}
			},
			wantExit: 1,
			wantOut:  []string{`"test (ubuntu-latest)" concluded cancelled`},
		},
		{
			name: "ignoresUnlisted",
			gh: func(t *testing.T) shelltest.Stub {
				return shelltest.Stub{Stdout: checkRunPages(t, []checkRun{
					completed("lint", "success"), completed("test (ubuntu-latest)", "success"),
					completed("dependency-review", "failure"), {Name: "labeler", Status: "queued"},
				})}
			},
			wantOut: []string{"2 required checks succeeded"},
			notOut:  []string{"dependency-review", "labeler"},
		},
		{
			name: "ghFails",
			gh: func(*testing.T) shelltest.Stub {
				return shelltest.Stub{Script: "echo 'HTTP 404' >&2", Exit: 1}
			},
			wantExit: 1,
			wantOut:  []string{"cannot list check runs"},
		},
		{
			name:     "emptyList",
			required: "# nothing\n",
			gh: func(t *testing.T) shelltest.Stub {
				return shelltest.Stub{Stdout: checkRunPages(t, []checkRun{completed("lint", "success")})}
			},
			wantExit: 1,
			wantOut:  []string{"lists no checks"},
		},
	}
	// Each case also runs with a jq whose output ends in CRLF.
	for _, tc := range tests {
		for _, crlfJQ := range []bool{false, true} {
			name := tc.name
			if crlfJQ {
				name += "/crlfJQ"
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				checkVerifyCI(t, tc.required, tc.gh(t), crlfJQ, tc.wantExit, tc.wantOut, tc.notOut)
			})
		}
	}
}

// checkVerifyCI runs one TestVerifyCI case and checks its exit code, output
// and gh call.
func checkVerifyCI(t *testing.T, required string, gh shelltest.Stub, crlfJQ bool, wantExit int, wantOut, notOut []string) {
	t.Helper()
	checkCall := required == ""
	if required == "" {
		required = verifyRequired
	}
	res := runVerifyCI(t, required, gh, crlfJQ)
	if res.Exit != wantExit {
		t.Errorf("exit code = %d, want %d\noutput:\n%s", res.Exit, wantExit, res.Output)
	}
	for _, s := range wantOut {
		if !strings.Contains(res.Output, s) {
			t.Errorf("output does not contain %q:\n%s", s, res.Output)
		}
	}
	for _, s := range notOut {
		if strings.Contains(res.Output, s) {
			t.Errorf("output contains %q:\n%s", s, res.Output)
		}
	}
	if !checkCall {
		return
	}
	var ghCalls []string
	for _, c := range res.Calls {
		if strings.HasPrefix(c, "gh ") {
			ghCalls = append(ghCalls, c)
		}
	}
	wantCall := "gh api --paginate repos/" + verifyRepo + "/commits/" + verifySHA + "/check-runs?filter=latest&per_page=100"
	if len(ghCalls) != 1 || ghCalls[0] != wantCall {
		t.Errorf("gh calls = %q, want [%q]", ghCalls, wantCall)
	}
}
