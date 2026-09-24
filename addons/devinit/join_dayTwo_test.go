package devinit

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
)

// cloneCommitted copies the committed files of project src into a fresh
// directory with the same base name (generated content embeds it), as a
// teammate's clone would see them.
func cloneCommitted(t *testing.T, src string, files ...string) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), filepath.Base(src))
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(src, rel))
		if err != nil {
			t.Fatalf("reading %s: %v", rel, err)
		}
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dst, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, rel), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dst
}

// TestJoin_KeepsCommittedPackagesAndServices is the regression test for a
// teammate's join rebuilding devenv.nix without the packages recorded only in
// the gitignored answers: .qsdev.yaml now carries them, so the joiner's
// devenv.nix matches the committed one instead of dropping them (or leaving a
// .new sidecar to merge).
func TestJoin_KeepsCommittedPackagesAndServices(t *testing.T) {
	src := initLifecycleProject(t)
	if out, err := executeInitCmd(t, src, "--yes", "--force", "--lang", "go", "--tier", "full",
		"--packages", "jq", "--service", "redis"); err != nil {
		t.Fatalf("re-init with packages: %v\n%s", err, out)
	}
	committed := readProjectFile(t, src, "devenv.nix")
	if !strings.Contains(committed, "pkgs.jq") || !strings.Contains(committed, "redis") {
		t.Fatalf("fixture devenv.nix lacks jq/redis:\n%s", committed)
	}

	dst := cloneCommitted(t, src, branding.Get().ConfigFile, "go.mod", "devenv.nix")
	if out, err := executeInitCmd(t, dst, "--mode", "join", "--yes"); err != nil {
		t.Fatalf("join: %v\n%s", err, out)
	}

	if got := readProjectFile(t, dst, "devenv.nix"); got != committed {
		t.Errorf("join changed the committed devenv.nix:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(dst, "devenv.nix"+generate.SidecarSuffix)); !os.IsNotExist(err) {
		t.Errorf("join left a devenv.nix sidecar (err=%v)", err)
	}
}

// TestLifecycle_EnableDisableRecordedInProjectConfig checks qsdev
// enable/disable record the tool decision in the committed .qsdev.yaml, so a
// joiner gets the same tool set.
func TestLifecycle_EnableDisableRecordedInProjectConfig(t *testing.T) {
	dir := initLifecycleProject(t)
	cfgPath := filepath.Join(dir, branding.Get().ConfigFile)
	if loadProjectAnswers(t, dir).EnabledTools["semble"] {
		mustDisable(t, dir, "semble", "--force")
	}

	mustEnable(t, dir, "semble")
	cfg, err := qsdevconfig.ParseQsdevConfig(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(cfg.Tools.Enabled, "semble") {
		t.Errorf("tools.enabled = %v, want semble after enable", cfg.Tools.Enabled)
	}

	mustDisable(t, dir, "semble", "--force")
	cfg, err = qsdevconfig.ParseQsdevConfig(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(cfg.Tools.Disabled, "semble") || slices.Contains(cfg.Tools.Enabled, "semble") {
		t.Errorf("tools = %+v, want semble disabled after disable", cfg.Tools)
	}
}

// TestJoin_HandEditedDevenvNixIsKeptAndReported checks a committed devenv.nix
// hand edit that .qsdev.yaml does not describe survives a join (the
// regenerated content goes to a sidecar) and that join does not report
// unqualified success.
func TestJoin_HandEditedDevenvNixIsKeptAndReported(t *testing.T) {
	src := initLifecycleProject(t)
	edited := strings.Replace(readProjectFile(t, src, "devenv.nix"), "packages = [", "packages = [ pkgs.hello", 1)
	if err := os.WriteFile(filepath.Join(src, "devenv.nix"), []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	dst := cloneCommitted(t, src, branding.Get().ConfigFile, "go.mod", "devenv.nix")
	out, err := executeInitCmd(t, dst, "--mode", "join", "--yes")
	if err != nil {
		t.Fatalf("join: %v\n%s", err, out)
	}
	if got := readProjectFile(t, dst, "devenv.nix"); got != edited {
		t.Error("join overwrote the hand-edited devenv.nix")
	}
	requireFileExists(t, dst, "devenv.nix"+generate.SidecarSuffix)
	if strings.Contains(out, "Joined project successfully") || !strings.Contains(out, "differ from what it generates") {
		t.Errorf("join should report the file needing a merge, got:\n%s", out)
	}
}
