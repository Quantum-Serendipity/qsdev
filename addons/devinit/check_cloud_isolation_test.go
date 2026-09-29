package devinit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/check"
)

// cloudIsolationResults runs `qsdev check --format json` in dir and returns
// the cloud isolation results by name.
func cloudIsolationResults(t *testing.T, dir string) (map[string]check.CheckResult, error) {
	t.Helper()
	out, runErr := runLifecycleCmd(t, dir, checkCmd(), "--format", "json", "--audit-level", "medium")
	start := strings.Index(out, "{")
	if start < 0 {
		t.Fatalf("no JSON report in output:\n%s", out)
	}
	var report check.CheckReport
	if err := json.NewDecoder(strings.NewReader(out[start:])).Decode(&report); err != nil {
		t.Fatalf("parsing report: %v\n%s", err, out)
	}
	results := make(map[string]check.CheckResult)
	for _, c := range report.Checks {
		if strings.HasPrefix(c.Name, "cloud_isolation_") {
			results[c.Name] = c
		}
	}
	return results, runErr
}

func TestCheckCmd_CloudIsolation(t *testing.T) {
	const (
		envLayer     = "cloud_isolation_aws_environment_separation"
		maskingLayer = "cloud_isolation_aws_credential_file_masking"
		denyLayer    = "cloud_isolation_aws_agent_deny_rules"
	)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/cloud\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Cloud providers are detected, not chosen with --lang: cdk.json marks
	// an AWS CDK project.
	if err := os.WriteFile(filepath.Join(dir, "cdk.json"), []byte(`{"app": "go run ."}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := executeInitCmd(t, dir, "--yes", "--tier", "full"); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}

	// A freshly generated project carries the masking and deny layers; with
	// no profile declared, environment separation only warns.
	results, _ := cloudIsolationResults(t, dir)
	wantStatus := map[string]check.CheckStatus{
		envLayer:     check.StatusWarn,
		maskingLayer: check.StatusPass,
		denyLayer:    check.StatusPass,
	}
	for name, want := range wantStatus {
		if got := results[name].Status; got != want {
			t.Errorf("fresh project: %s = %q, want %q (%s)", name, got, want, results[name].Message)
		}
	}

	// A placeholder in devenv.local.nix is not isolation; a real value is.
	localNix := filepath.Join(dir, "devenv.local.nix")
	for _, tc := range []struct {
		value string
		want  check.CheckStatus
	}{
		{"PLACEHOLDER -- set your AWS profile", check.StatusWarn},
		{"cloud-dev", check.StatusPass},
	} {
		src := "{ ... }:\n{\n  env.AWS_PROFILE = \"" + tc.value + "\";\n}\n"
		if err := os.WriteFile(localNix, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		results, _ = cloudIsolationResults(t, dir)
		if got := results[envLayer].Status; got != tc.want {
			t.Errorf("AWS_PROFILE=%q: %s = %q, want %q", tc.value, envLayer, got, tc.want)
		}
	}

	// Removing the generated credential rules fails the run.
	settingsPath := filepath.Join(dir, ".claude", "settings.json")
	var settings map[string]any
	if err := json.Unmarshal([]byte(readProjectFile(t, dir, ".claude/settings.json")), &settings); err != nil {
		t.Fatal(err)
	}
	perms := settings["permissions"].(map[string]any)
	var kept []any
	for _, r := range perms["deny"].([]any) {
		rule, _ := r.(string)
		if !strings.Contains(rule, ".aws") && !strings.HasPrefix(rule, "Bash(aws ") {
			kept = append(kept, r)
		}
	}
	perms["deny"] = kept
	delete(settings, "sandbox")
	data, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	results, err = cloudIsolationResults(t, dir)
	if err == nil {
		t.Error("check passed with the AWS credential rules removed")
	}
	for _, name := range []string{maskingLayer, denyLayer} {
		if r := results[name]; r.Status != check.StatusFail || r.Severity != check.SeverityHigh {
			t.Errorf("stripped rules: %s = %s/%s, want fail/high", name, r.Status, r.Severity)
		}
	}
}

// A project set up without Claude Code has no .claude/settings.json, so the
// agent layers do not apply: only environment separation is reported and the
// run passes.
func TestCheckCmd_CloudIsolation_DevenvOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/cloud\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cdk.json"), []byte(`{"app": "go run ."}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := executeInitCmd(t, dir, "--yes", "--devenv-only"); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "settings.json")); !os.IsNotExist(err) {
		t.Fatalf("devenv-only init wrote .claude/settings.json (stat err %v)", err)
	}

	results, _ := cloudIsolationResults(t, dir)
	if r := results["cloud_isolation_aws_environment_separation"]; r.Status != check.StatusWarn {
		t.Errorf("environment separation = %q, want warn (%s)", r.Status, r.Message)
	}
	for name, r := range results {
		if r.Status == check.StatusFail {
			t.Errorf("%s failed on a project without Claude Code: %s", name, r.Message)
		}
	}
}
