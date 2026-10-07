package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
)

// invalidProjectDefaults is a project defaults file the catalog rejects: it
// maps a tier to a compliance profile that does not exist.
const invalidProjectDefaults = "tier_to_compliance:\n  standard: nonexistent\n"

// TestInvalidTierComplianceNoPanic is the U08-03 regression test. A project
// whose .qsdev/defaults.yaml the catalog rejects used to crash every command
// that needs the tool registry with a panic and a goroutine dump, and to print
// a stray profile-loading ERROR from every other command. Interactive commands
// now fail with one actionable error from the root catalog gate, and commands
// the gate exempts (version, the defaults group that repairs the file) print
// no such error.
func TestInvalidTierComplianceNoPanic(t *testing.T) {
	env := guardrailEnv(t)
	dir := newGuardrailProject(t, env)
	mustQsdev(t, env, dir, "init", "--yes")
	defaultsPath := catalog.ProjectConfigPath(dir)
	if err := os.MkdirAll(filepath.Dir(defaultsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(defaultsPath, []byte(invalidProjectDefaults), 0o644); err != nil {
		t.Fatal(err)
	}

	const (
		gateErr    = "loading qsdev defaults"
		gateHint   = "(run 'qsdev defaults validate')"
		repairHint = "defaults validate"
		profileErr = "loading built-in project profiles"
		checkFail  = "[FAIL] config_catalog: loading qsdev defaults:"
	)
	tests := []struct {
		args    []string
		code    int
		want    []string
		notWant []string
		// jsonFail: the output is a JSON report whose config_integrity
		// config_catalog check failed.
		jsonFail bool
	}{
		{args: []string{"status"}, code: 1, want: []string{gateErr, gateHint}},
		{args: []string{"list"}, code: 1, want: []string{gateErr, gateHint}},
		{args: []string{"info"}, code: 1, want: []string{gateErr, gateHint}},
		// check reports the failure as a structured result instead.
		{args: []string{"check"}, code: 1, want: []string{checkFail, repairHint}, notWant: []string{gateHint}},
		{args: []string{"check", "--format", "json"}, code: 1, want: []string{repairHint}, notWant: []string{gateHint}, jsonFail: true},
		{args: []string{"version"}, code: 0, notWant: []string{gateErr, profileErr}},
		{args: []string{"defaults", "validate"}, code: 1, notWant: []string{gateErr, profileErr}},
		{args: []string{"enforce", "--help"}, code: 0, notWant: []string{gateErr, profileErr}},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, "_"), func(t *testing.T) {
			out, code := runQsdev(t, env, dir, nil, tt.args...)
			if code != tt.code {
				t.Errorf("exit code = %d, want %d", code, tt.code)
			}
			for _, s := range append([]string{"panic:", "goroutine "}, tt.notWant...) {
				if strings.Contains(out, s) {
					t.Errorf("output contains %q", s)
				}
			}
			for _, s := range tt.want {
				if !strings.Contains(out, s) {
					t.Errorf("output lacks %q", s)
				}
			}
			if tt.jsonFail {
				if got := catalogCheckStatus(t, out); got != "fail" {
					t.Errorf("config_integrity/config_catalog status = %q, want fail", got)
				}
			}
			if t.Failed() {
				t.Logf("output:\n%s", out)
			}
		})
	}
}

// catalogCheckStatus returns the status of the config_integrity
// config_catalog check in the JSON report out, or "" when it has none.
func catalogCheckStatus(t *testing.T, out string) string {
	t.Helper()
	start := strings.IndexByte(out, '{')
	if start < 0 {
		t.Fatalf("no JSON report in output:\n%s", out)
	}
	var report struct {
		Checks []struct {
			Category string `json:"category"`
			Name     string `json:"name"`
			Status   string `json:"status"`
		} `json:"checks"`
	}
	if err := json.NewDecoder(strings.NewReader(out[start:])).Decode(&report); err != nil {
		t.Fatalf("decoding the JSON report: %v\n%s", err, out)
	}
	for _, c := range report.Checks {
		if c.Category == "config_integrity" && c.Name == "config_catalog" {
			return c.Status
		}
	}
	return ""
}
