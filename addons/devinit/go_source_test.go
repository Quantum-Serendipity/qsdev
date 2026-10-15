package devinit

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// goSourceOutput reports whether dir's generated devenv files take Go from
// nixpkgs (the guarded package line, no go-overlay input) and fails the test
// when devenv.yaml and devenv.nix disagree about it.
func goSourceOutput(t *testing.T, dir string) (nixpkgs bool) {
	t.Helper()
	overlayInput := strings.Contains(readProjectFile(t, dir, "devenv.yaml"), "go-overlay")
	nix := readProjectFile(t, dir, "devenv.nix")
	guarded := strings.Contains(nix, "package = lib.mkOverride 999 (if lib.versionAtLeast pkgs.go.version")
	versionLine := strings.Contains(nix, "version = lib.mkIf (!(lib.versionAtLeast pkgs.go.version")
	if overlayInput != versionLine || guarded == versionLine {
		t.Fatalf("inconsistent Go source output: go-overlay input %v, version line %v, guarded package %v",
			overlayInput, versionLine, guarded)
	}
	return guarded
}

// committedGoExtras returns the extras of the go entry of dir's .qsdev.yaml.
func committedGoExtras(t *testing.T, dir string) []string {
	t.Helper()
	cfg, err := qsdevconfig.ParseQsdevConfig(filepath.Join(dir, branding.Get().ConfigFile))
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range cfg.Languages {
		if l.Name == "go" {
			return l.Extras
		}
	}
	return nil
}

// TestCommittedGoSource_RemovedRevertsOnUpdate checks .qsdev.yaml is the
// single source of truth for go_source: removing it there and running
// `init --update` returns to go-overlay, although the saved answers still
// hold nixpkgs.
func TestCommittedGoSource_RemovedRevertsOnUpdate(t *testing.T) {
	dir := initLifecycleProject(t)
	commitGoExtras(t, dir, "go_source=nixpkgs")
	if out, err := executeInitCmd(t, dir, "--update", "--yes"); err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	if !goSourceOutput(t, dir) {
		t.Fatal("committed go_source=nixpkgs did not reach generation")
	}

	commitGoExtras(t, dir)
	if out, err := executeInitCmd(t, dir, "--update", "--yes"); err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	if goSourceOutput(t, dir) {
		t.Error("removing go_source from .qsdev.yaml kept the nixpkgs output")
	}
	for _, l := range loadProjectAnswers(t, dir).Languages {
		if v, _ := l.Setting("go_source"); l.Name == "go" && v == "nixpkgs" {
			t.Error("saved answers still hold go_source=nixpkgs")
		}
	}
}

// TestUpdate_WarnsAboutIgnoredExtras checks `init --update` reports each
// committed extra it ignores instead of dropping it silently.
func TestUpdate_WarnsAboutIgnoredExtras(t *testing.T) {
	dir := initLifecycleProject(t)
	commitGoExtras(t, dir, "go_sorce=nixpkgs", "go_source=nixpgks")
	out, err := executeInitCmd(t, dir, "--update", "--yes")
	if err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	for _, want := range []string{
		`Warning: languages[0].extras: "go_sorce" is not a committable setting for go; committable settings: go_source; ignored`,
		`Warning: languages[0].extras: setting "go_source" for go: invalid value "nixpgks"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("update output lacks %q:\n%s", want, out)
		}
	}
	if goSourceOutput(t, dir) {
		t.Error("an invalid committed go_source changed the output")
	}
}

// TestAnswersFileGoSource_RecordedForJoin checks a go_source=nixpkgs chosen
// when the project is created (here in an answers file) is recorded in
// .qsdev.yaml, so a teammate's join of a clean clone generates the same
// devenv files instead of a go-overlay devenv.yaml.new.
func TestAnswersFileGoSource_RecordedForJoin(t *testing.T) {
	a := loadProjectAnswers(t, initLifecycleProject(t))
	for i := range a.Languages {
		if a.Languages[i].Name == "go" {
			a.Languages[i] = a.Languages[i].WithSetting("go_source", "nixpkgs")
		}
	}
	data, err := yaml.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	answersFile := filepath.Join(t.TempDir(), "answers.yaml")
	if err := os.WriteFile(answersFile, data, 0o644); err != nil {
		t.Fatal(err)
	}

	src := testutil.IsolatedDir(t)
	if err := os.WriteFile(filepath.Join(src, "go.mod"), []byte("module example.com/lc\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := executeInitCmd(t, src, "--yes", "--answers-file", answersFile); err != nil {
		t.Fatalf("init --answers-file: %v\n%s", err, out)
	}
	if got := committedGoExtras(t, src); !slices.Equal(got, []string{"go_source=nixpkgs"}) {
		t.Errorf(".qsdev.yaml go extras = %v, want [go_source=nixpkgs]", got)
	}
	if !goSourceOutput(t, src) {
		t.Error("the answers file's go_source=nixpkgs did not reach generation")
	}

	dst := cloneCommitted(t, src, branding.Get().ConfigFile, "go.mod")
	if out, err := executeInitCmd(t, dst, "--mode", "join", "--yes"); err != nil {
		t.Fatalf("join: %v\n%s", err, out)
	}
	for _, f := range []string{"devenv.yaml", "devenv.nix"} {
		if readProjectFile(t, dst, f) != readProjectFile(t, src, f) {
			t.Errorf("joined %s differs from the creator's", f)
		}
		if _, err := os.Stat(filepath.Join(dst, f+".new")); err == nil {
			t.Errorf("join left %s.new", f)
		}
	}
}

// commitGoExtras records extras on the go entry of dir's .qsdev.yaml.
func commitGoExtras(t *testing.T, dir string, extras ...string) {
	t.Helper()
	path := filepath.Join(dir, branding.Get().ConfigFile)
	cfg, err := qsdevconfig.ParseQsdevConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := range cfg.Languages {
		if cfg.Languages[i].Name == "go" {
			cfg.Languages[i].Extras = extras
		}
	}
	if err := qsdevconfig.WriteProjectConfig(dir, *cfg); err != nil {
		t.Fatal(err)
	}
}

// TestCommittedGoSource_ReachesGeneration checks that a committed
// go_source=nixpkgs shapes what create (re-creating over the committed file,
// which kept the extra but generated the go-overlay input), join and update
// generate, and stays committed. The non-committable extra beside it is
// never applied.
func TestCommittedGoSource_ReachesGeneration(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, dir string) string // returns the project generated
	}{
		{"create --force", func(t *testing.T, dir string) string {
			if out, err := executeInitCmd(t, dir, "--yes", "--force", "--lang", "go", "--tier", "full"); err != nil {
				t.Fatalf("re-init: %v\n%s", err, out)
			}
			return dir
		}},
		{"join", func(t *testing.T, dir string) string {
			dst := cloneCommitted(t, dir, branding.Get().ConfigFile, "go.mod")
			if out, err := executeInitCmd(t, dst, "--mode", "join", "--yes"); err != nil {
				t.Fatalf("join: %v\n%s", err, out)
			}
			return dst
		}},
		{"update", func(t *testing.T, dir string) string {
			if out, err := executeInitCmd(t, dir, "--update", "--yes"); err != nil {
				t.Fatalf("update: %v\n%s", err, out)
			}
			return dir
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := initLifecycleProject(t) // go.mod: go 1.24
			if !strings.Contains(readProjectFile(t, dir, "devenv.yaml"), "go-overlay") {
				t.Fatal("fixture devenv.yaml lacks the go-overlay input")
			}
			commitGoExtras(t, dir, "go_source=nixpkgs", "directory=docs")

			got := tt.run(t, dir)

			if yaml := readProjectFile(t, got, "devenv.yaml"); strings.Contains(yaml, "go-overlay") {
				t.Errorf("devenv.yaml still declares the go-overlay input:\n%s", yaml)
			}
			nix := readProjectFile(t, got, "devenv.nix")
			if want := `package = lib.mkOverride 999 (if lib.versionAtLeast pkgs.go.version "1.24.0" then pkgs.go else throw`; !strings.Contains(nix, want) {
				t.Errorf("devenv.nix lacks the guarded package line %q", want)
			}
			if strings.Contains(nix, "version = lib.mkIf") {
				t.Error("devenv.nix still sets languages.go.version")
			}
			for _, l := range loadProjectAnswers(t, got).Languages {
				if l.Name != "go" {
					continue
				}
				if v, _ := l.Setting("go_source"); v != "nixpkgs" {
					t.Errorf("saved answers go_source = %q, want nixpkgs", v)
				}
				if v, _ := l.Setting("directory"); v == "docs" {
					t.Error("the non-committable directory=docs reached the answers")
				}
			}
			cfg, err := qsdevconfig.ParseQsdevConfig(filepath.Join(got, branding.Get().ConfigFile))
			if err != nil {
				t.Fatal(err)
			}
			for _, l := range cfg.Languages {
				if l.Name == "go" && !slices.Contains(l.Extras, "go_source=nixpkgs") {
					t.Errorf(".qsdev.yaml go extras = %v, want go_source=nixpkgs kept", l.Extras)
				}
			}
		})
	}
}
