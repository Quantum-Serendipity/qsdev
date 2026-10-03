package devinit

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestRecordOrgOverlay pins who approves the org overlay a project's
// regenerations read (U18-WS1 round 3): a human's init at their own terminal
// records the overlay it resolved; an init without a terminal, or inside an
// agent session, records nothing, so an agent cannot approve an overlay of
// its own; and an overlay below the project is refused with a warning. Not
// parallel: it changes the working directory and the environment.
func TestRecordOrgOverlay(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(base, "project")
	overlay := filepath.Join(base, "org", "defaults.yaml")
	tmp := filepath.Join(base, "tmp")
	for _, d := range []string{filepath.Join(project, filepath.FromSlash(branding.Get().StateDir)), filepath.Dir(overlay), tmp} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(project, filepath.FromSlash(state.InitStateFile())), []byte("files: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(v, tmp)
	}
	env := branding.Get().EnvPrefix + "ORG_CONFIG"
	t.Setenv(env, overlay)
	t.Chdir(project)

	run := func(tty bool, agent string) string {
		t.Helper()
		t.Setenv("CLAUDECODE", agent)
		var stderr bytes.Buffer
		cmd := &cobra.Command{Use: "init"}
		cmd.SetIn(ttyInput{Reader: strings.NewReader(""), tty: tty})
		cmd.SetErr(&stderr)
		recordOrgOverlay(cmd)
		return stderr.String()
	}
	recorded := func() (string, bool) {
		t.Helper()
		p, ok, err := state.LoadOrgOverlay(project)
		if err != nil {
			t.Fatal(err)
		}
		return p, ok
	}

	run(false, "")
	run(true, "1")
	if p, ok := recorded(); ok {
		t.Fatalf("an init no human ran recorded the overlay %q", p)
	}
	run(true, "")
	if p, ok := recorded(); !ok || p != overlay {
		t.Fatalf("recorded overlay after a human's init = %q, %v; want %q", p, ok, overlay)
	}
	t.Setenv(env, filepath.Join(project, "evil.yaml"))
	if out := run(true, ""); !strings.Contains(out, "below the project") {
		t.Errorf("init with an overlay below the project warned %q, want the refusal", out)
	}
	if p, _ := recorded(); p != overlay {
		t.Errorf("recorded overlay after a refused init = %q, want %q kept", p, overlay)
	}
}
