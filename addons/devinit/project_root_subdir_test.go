package devinit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/check"
	"github.com/Quantum-Serendipity/qsdev/internal/posture"
)

// TestCommands_RunFromSubdirectory is the regression test for project commands
// resolving the project as the working directory: run from a subdirectory,
// `qsdev check` reported .qsdev.yaml missing and `qsdev status` reported the
// project uninitialized. Both must find the enclosing project root.
func TestCommands_RunFromSubdirectory(t *testing.T) {
	dir := initLifecycleProject(t)
	subdir := filepath.Join(dir, "internal", "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		run    func() (string, error)
		verify func(t *testing.T, out string)
	}{
		{
			name: "check",
			run: func() (string, error) {
				return runLifecycleCmd(t, subdir, checkCmd(), "--format", "json", "--audit-level", "none")
			},
			verify: func(t *testing.T, out string) {
				t.Helper()
				var report check.CheckReport
				decodeJSONReport(t, out, &report)
				if !slices.ContainsFunc(report.Checks, func(c check.CheckResult) bool {
					return c.Name == "config_exists" && c.Status == check.StatusPass
				}) {
					t.Errorf("check from a subdirectory did not find the project config:\n%s", out)
				}
			},
		},
		{
			name: "status",
			run: func() (string, error) {
				return runLifecycleCmd(t, subdir, statusCmd(), "--format", "json", "--audit-level", "none")
			},
			verify: func(t *testing.T, out string) {
				t.Helper()
				var report posture.PostureReport
				decodeJSONReport(t, out, &report)
				if report.ProjectName == "" {
					t.Errorf("status from a subdirectory reported no project:\n%s", out)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := tt.run()
			if err != nil {
				t.Fatalf("%s from a subdirectory: %v\n%s", tt.name, err, out)
			}
			tt.verify(t, out)
		})
	}
}

// decodeJSONReport decodes the first JSON object in a command's output, which
// may be preceded by log or warning lines.
func decodeJSONReport(t *testing.T, out string, v any) {
	t.Helper()
	start := strings.Index(out, "{")
	if start < 0 {
		t.Fatalf("no JSON report in output:\n%s", out)
	}
	if err := json.NewDecoder(strings.NewReader(out[start:])).Decode(v); err != nil {
		t.Fatalf("parsing report: %v\n%s", err, out)
	}
}
