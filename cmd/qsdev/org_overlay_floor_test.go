package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/check"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestOrgOverlayLoosening_RefusesGeneration is the XS-WS2 B6 / GC-WS1 E3
// test: an org overlay that tries to loosen the built-in security floor
// (keeping a credential variable in the shell) is rejected when it loads,
// and the catalog then skips it. No command may generate past it on the
// built-in defaults alone, which would also drop the overlay's tightening
// entries: init exits non-zero and writes nothing, and check fails
// config_catalog. A read-only command still runs, and says it skipped the
// overlay.
func TestOrgOverlayLoosening_RefusesGeneration(t *testing.T) {
	t.Parallel()
	env := guardrailEnv(t)
	dir := newGuardrailProject(t, env)
	overlay := filepath.Join(t.TempDir(), "org", "defaults.yaml")
	if err := os.MkdirAll(filepath.Dir(overlay), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "unset_vars: [MY_ORG_TOKEN]\nkeep_vars: [PATH, AWS_SECRET_ACCESS_KEY]\n"
	if err := os.WriteFile(overlay, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	env = append(slices.Clone(env), branding.Get().EnvPrefix+"ORG_CONFIG="+overlay)
	// A human pins the overlay, so every run reads it (catalog.OrgConfigPin).
	mustQsdev(t, env, dir, "defaults", "pin", "--global")

	const loosens = "loosens the built-in security floor"
	out, code := runQsdev(t, env, dir, nil, "init", "--yes", "--lang", "go", "--tier", "full", "--claude-code=false")
	if code == 0 || !strings.Contains(out, loosens) || !strings.Contains(out, "defaults validate") {
		t.Fatalf("init with a loosening org overlay: exit %d, want non-zero naming the violation and 'defaults validate'\n%s", code, out)
	}
	for _, rel := range []string{"devenv.nix", branding.Get().ConfigFile, ".claude"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); !os.IsNotExist(err) {
			t.Errorf("init refused but %s exists (stat err %v)", rel, err)
		}
	}

	stdout, stderr, code := runQsdevSplit(t, env, dir, "check", "--format", "json")
	var report check.CheckReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("check stdout is not a JSON report: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	i := slices.IndexFunc(report.Checks, func(r check.CheckResult) bool { return r.Name == "config_catalog" })
	if i < 0 {
		t.Fatalf("check reports no config_catalog result for the skipped overlay: %+v", report.Checks)
	}
	if r := report.Checks[i]; r.Status != check.StatusFail || r.Severity != check.SeverityHigh || !strings.Contains(r.Message, loosens) {
		t.Errorf("config_catalog = %+v, want a high-severity failure naming the violation", r)
	}
	if code == 0 {
		t.Errorf("check exit 0 with a skipped org overlay")
	}

	out, code = runQsdev(t, env, dir, nil, "init", "--yes", "--dry-run", "--lang", "go")
	if code != 0 || !strings.Contains(out, loosens) {
		t.Errorf("init --dry-run (read-only): exit %d, want 0 with a warning naming the skipped overlay\n%s", code, out)
	}
}
