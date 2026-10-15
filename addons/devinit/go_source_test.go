package devinit

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

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
// generate, and stays committed. The undeclared extra beside it is never
// applied.
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
					t.Error("the undeclared committed directory=docs reached the answers")
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
