package devinit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestUpdate_HoldsDevenvYamlWithModifiedDevenvNix reproduces the split the
// self-regeneration audit found: devenv.yaml gained a module flake input
// while devenv.nix, holding the user's edits, only got a sidecar, so the
// devenv.nix in effect never read the new input. devenv.yaml now follows
// devenv.nix: held in a sidecar while devenv.nix is, written in place (and
// its stale sidecar removed) once devenv.nix is.
func TestUpdate_HoldsDevenvYamlWithModifiedDevenvNix(t *testing.T) {
	dir := initLifecycleProject(t)
	nixPath := filepath.Join(dir, "devenv.nix")
	yamlPath := filepath.Join(dir, "devenv.yaml")
	userNix := readProjectFile(t, dir, "devenv.nix") + "# local edit\n"
	if err := os.WriteFile(nixPath, []byte(userNix), 0o644); err != nil {
		t.Fatal(err)
	}
	oldYAML := readProjectFile(t, dir, "devenv.yaml")

	answers := loadProjectAnswers(t, dir)
	answers.Languages = append(answers.Languages, types.LanguageChoice{Name: "rust", Version: "stable"})
	if err := saveAnswers(dir, answers); err != nil {
		t.Fatal(err)
	}

	out, err := executeInitCmd(t, dir, "--update", "--yes")
	if err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	if got := readProjectFile(t, dir, "devenv.nix"); got != userNix {
		t.Fatalf("devenv.nix was rewritten despite the user's edits")
	}
	if got := readProjectFile(t, dir, "devenv.yaml"); got != oldYAML {
		t.Errorf("devenv.yaml was rewritten while devenv.nix was held:\n%s", got)
	}
	sidecar := readProjectFile(t, dir, "devenv.yaml.new")
	if !strings.Contains(sidecar, "rust-overlay") {
		t.Errorf("devenv.yaml.new lacks the new input:\n%s", sidecar)
	}
	if !strings.Contains(out, "devenv.yaml.new") {
		t.Errorf("update output does not point at devenv.yaml.new:\n%s", out)
	}

	// Accepting the devenv.nix sidecar lets the pair move together, and the
	// input devenv.lock does not pin yet is called out for review.
	if err := os.Rename(nixPath+".new", nixPath); err != nil {
		t.Fatal(err)
	}
	lock := `{"root": "root", "nodes": {"root": {"inputs": {"git-hooks": "git-hooks", "go-overlay": "go-overlay", "nixpkgs": "nixpkgs"}}}, "version": 7}`
	if err := os.WriteFile(filepath.Join(dir, "devenv.lock"), []byte(lock), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err = executeInitCmd(t, dir, "--update", "--yes")
	if err != nil {
		t.Fatalf("second update: %v\n%s", err, out)
	}
	if !strings.Contains(out, "devenv update rust-overlay") {
		t.Errorf("update did not flag the input devenv.lock does not pin:\n%s", out)
	}
	if got := readProjectFile(t, dir, "devenv.yaml"); !strings.Contains(got, "rust-overlay") {
		t.Errorf("devenv.yaml not written with devenv.nix:\n%s", got)
	}
	if _, err := os.Stat(yamlPath + ".new"); err == nil {
		t.Error("stale devenv.yaml.new left behind after devenv.yaml was written")
	}
}

// TestHoldPairedFiles covers each partner outcome: a held file is written in
// place only when its partner is.
func TestHoldPairedFiles(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		partner FileUpdatePlan
		force   bool
		want    UpdateAction
	}{
		{"partner regenerated", FileUpdatePlan{Action: UpdateActionRegenerate, Status: types.Unmodified}, false, UpdateActionRegenerate},
		{"partner sidecar on unmodified file", FileUpdatePlan{Action: UpdateActionSidecar, Status: types.Unmodified}, false, UpdateActionRegenerate},
		{"partner sidecar on modified file", FileUpdatePlan{Action: UpdateActionSidecar, Status: types.Modified}, false, UpdateActionHold},
		{"partner sidecar on modified file, forced", FileUpdatePlan{Action: UpdateActionSidecar, Status: types.Modified}, true, UpdateActionRegenerate},
		{"partner deleted", FileUpdatePlan{Action: UpdateActionSidecar, Status: types.Deleted}, false, UpdateActionHold},
		{"partner skipped", FileUpdatePlan{Action: UpdateActionSkip, Status: types.Modified}, false, UpdateActionHold},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "devenv.yaml"), []byte("old\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			partner := tt.partner
			partner.Path = "devenv.nix"
			files := []FileUpdatePlan{
				{Path: "devenv.yaml", Action: UpdateActionRegenerate, Status: types.Unmodified, HeldWith: "devenv.nix"},
				partner,
			}
			holdPairedFiles(files, dir, UpdateOptions{Force: tt.force})
			if got := files[0].Action; got != tt.want {
				t.Errorf("devenv.yaml action = %s, want %s", updateActionString(got), updateActionString(tt.want))
			}
		})
	}
}
