package devinit

import (
	"path/filepath"
	"strings"
	"testing"

	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestUpdate_KeepsCommittedNewerGoPin reproduces the self-regeneration that
// rewrote this repository's committed Go 1.26.7 pin to 1.26.3: the saved
// answers file is local and stale, and `init --update` must keep the version
// .qsdev.yaml pins, in the committed config and in the generated devenv.nix.
func TestUpdate_KeepsCommittedNewerGoPin(t *testing.T) {
	dir := initLifecycleProject(t) // go.mod: go 1.24

	cfgPath := filepath.Join(dir, branding.Get().ConfigFile)
	cfg, err := qsdevconfig.ParseQsdevConfig(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	for i := range cfg.Languages {
		if cfg.Languages[i].Name == "go" {
			cfg.Languages[i].Version = "1.26.7"
		}
	}
	if err := qsdevconfig.WriteProjectConfig(dir, *cfg); err != nil {
		t.Fatal(err)
	}
	answers := loadProjectAnswers(t, dir)
	for i := range answers.Languages {
		if answers.Languages[i].Name == "go" {
			answers.Languages[i].Version = "1.26.3"
		}
	}
	if err := saveAnswers(dir, answers); err != nil {
		t.Fatal(err)
	}

	if out, err := executeInitCmd(t, dir, "--update", "--yes"); err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}

	got, err := qsdevconfig.ParseQsdevConfig(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range got.Languages {
		if l.Name == "go" && l.Version != "1.26.7" {
			t.Errorf(".qsdev.yaml go version = %q, want the committed 1.26.7", l.Version)
		}
	}
	for _, l := range loadProjectAnswers(t, dir).Languages {
		if l.Name == "go" && l.Version != "1.26.7" {
			t.Errorf("saved answers go version = %q, want 1.26.7", l.Version)
		}
	}
	nix := readProjectFile(t, dir, "devenv.nix")
	if !strings.Contains(nix, `"1.26.7"`) || strings.Contains(nix, `"1.26.3"`) {
		t.Errorf("devenv.nix does not pin Go 1.26.7:\n%s", nix)
	}
}
