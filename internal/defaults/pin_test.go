package defaults

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// ttyInput is stdin a test presents as an interactive terminal, or not.
type ttyInput struct {
	io.Reader
	tty bool
}

func (in ttyInput) IsTerminal() bool { return in.tty }

// TestDefaultsPin pins who approves the org overlay every run reads (U18-WS1
// round 4): 'defaults pin' is sensitive, so a run without a terminal or
// inside an agent session is refused and records nothing; a human's run
// records the overlay it resolved, for the project or, with --global, the
// account; and an overlay below the project is refused. Not parallel: it
// changes the working directory and the environment.
func TestDefaultsPin(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(base, "project")
	overlay := filepath.Join(base, "org", "defaults.yaml")
	other := filepath.Join(base, "org2", "defaults.yaml")
	home := filepath.Join(base, "home")
	tmp := filepath.Join(base, "tmp")
	for _, d := range []string{filepath.Join(project, ".git"), filepath.Dir(overlay), filepath.Dir(other), home, tmp} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, v := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(v, tmp)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	env := branding.Get().EnvPrefix + "ORG_CONFIG"
	t.Setenv(env, overlay)
	t.Chdir(project)

	run := func(tty bool, agent string, args ...string) (string, error) {
		t.Helper()
		t.Setenv("CLAUDECODE", agent)
		root := &cobra.Command{Use: "qsdev", SilenceErrors: true, SilenceUsage: true}
		root.AddCommand(Command())
		cmdutil.InstallHumanGate(root)
		var out bytes.Buffer
		root.SetIn(ttyInput{Reader: strings.NewReader(""), tty: tty})
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs(append([]string{"defaults", "pin"}, args...))
		err := root.Execute()
		return out.String(), err
	}
	pinned := func(projectRoot string) catalog.OrgConfigPin {
		t.Helper()
		pin, err := catalog.LoadOrgConfigPin(projectRoot)
		if err != nil {
			t.Fatal(err)
		}
		return pin
	}

	for _, tt := range []struct {
		tty   bool
		agent string
	}{{false, ""}, {true, "1"}, {false, "1"}} {
		if _, err := run(tt.tty, tt.agent); err == nil || !strings.Contains(err.Error(), "requires a human") {
			t.Errorf("defaults pin with tty=%v CLAUDECODE=%q: %v, want a refusal", tt.tty, tt.agent, err)
		}
	}
	if pin := pinned(project); pin.Recorded {
		t.Fatalf("a run no human started pinned %+v", pin)
	}

	if _, err := run(true, ""); err != nil {
		t.Fatalf("a human's defaults pin: %v", err)
	}
	if pin := pinned(project); !pin.Recorded || pin.Global || pin.Path != overlay {
		t.Fatalf("pin after a human's defaults pin = %+v, want %q for the project", pin, overlay)
	}
	if pin := pinned(""); pin.Recorded {
		t.Errorf("a project pin also pinned the account: %+v", pin)
	}

	t.Setenv(env, other)
	if _, err := run(true, "", "--global"); err != nil {
		t.Fatalf("a human's defaults pin --global: %v", err)
	}
	if pin := pinned(base); !pin.Recorded || !pin.Global || pin.Path != other {
		t.Errorf("pin of another directory after --global = %+v, want %q for the account", pin, other)
	}
	if pin := pinned(project); pin.Path != overlay {
		t.Errorf("the project's pin after --global = %+v, want %q kept", pin, overlay)
	}

	t.Setenv(env, filepath.Join(project, "evil.yaml"))
	if _, err := run(true, ""); err == nil || !strings.Contains(err.Error(), "below the project") {
		t.Errorf("defaults pin of an overlay below the project = %v, want the refusal", err)
	}
	if pin := pinned(project); pin.Path != overlay {
		t.Errorf("pin after a refused defaults pin = %+v, want %q kept", pin, overlay)
	}
}
