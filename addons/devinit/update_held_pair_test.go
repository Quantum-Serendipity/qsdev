package devinit

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
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

// heldDevenvNixProject returns a project whose devenv.nix carries a local
// edit and whose first update, after adding a language, left devenv.nix.new
// and devenv.yaml.new beside the pair, with the merge instructions printed.
func heldDevenvNixProject(t *testing.T) (dir, userNix string) {
	t.Helper()
	dir = initLifecycleProject(t)
	userNix = readProjectFile(t, dir, "devenv.nix") + "# local edit\n"
	if err := os.WriteFile(filepath.Join(dir, "devenv.nix"), []byte(userNix), 0o644); err != nil {
		t.Fatal(err)
	}
	answers := loadProjectAnswers(t, dir)
	answers.Languages = append(answers.Languages, types.LanguageChoice{Name: "rust", Version: "stable"})
	if err := saveAnswers(dir, answers); err != nil {
		t.Fatal(err)
	}
	out, err := executeInitCmd(t, dir, "--update", "--yes")
	if err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	for _, sidecar := range []string{"devenv.nix.new", "devenv.yaml.new"} {
		if _, err := os.Stat(filepath.Join(dir, sidecar)); err != nil {
			t.Fatalf("first update left no %s: %v", sidecar, err)
		}
	}
	if !strings.Contains(out, "devenv.nix.new") {
		t.Fatalf("first update printed no merge instructions for devenv.nix.new:\n%s", out)
	}
	return dir, userNix
}

// assertNoSidecar fails when rel has a .new sidecar in dir.
func assertNoSidecar(t *testing.T, dir, rel string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, rel+".new")); !os.IsNotExist(err) {
		t.Errorf("stale %s.new left behind after %s was written in place (stat err=%v)", rel, rel, err)
	}
}

// TestUpdate_ForceOverwriteDropsManualMergeSidecar: once a forced update
// writes the held devenv.nix in place, the sidecar the previous update left
// for it is stale and goes, as devenv.yaml.new does.
func TestUpdate_ForceOverwriteDropsManualMergeSidecar(t *testing.T) {
	dir, _ := heldDevenvNixProject(t)
	generated := readProjectFile(t, dir, "devenv.nix.new")

	if out, err := executeInitCmd(t, dir, "--update", "--yes", "--force"); err != nil {
		t.Fatalf("forced update: %v\n%s", err, out)
	}
	if got := readProjectFile(t, dir, "devenv.nix"); got != generated {
		t.Errorf("forced update did not write the generated devenv.nix:\n%s", got)
	}
	assertNoSidecar(t, dir, "devenv.nix")
	assertNoSidecar(t, dir, "devenv.yaml")
}

// TestUpdate_AcceptedManualMergeContentDropsSidecar: a devenv.nix the user
// set to the generated content is tracked again, and its sidecar is stale.
func TestUpdate_AcceptedManualMergeContentDropsSidecar(t *testing.T) {
	dir, _ := heldDevenvNixProject(t)
	generated := readProjectFile(t, dir, "devenv.nix.new")
	if err := os.WriteFile(filepath.Join(dir, "devenv.nix"), []byte(generated), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := executeInitCmd(t, dir, "--update", "--yes")
	if err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	assertNoSidecar(t, dir, "devenv.nix")
	assertNoSidecar(t, dir, "devenv.yaml")
}

// TestUpdate_ModifiedDevenvNixKeepsSidecarWithoutForce: without --force a
// modified devenv.nix keeps the user's edits and gets a fresh sidecar.
func TestUpdate_ModifiedDevenvNixKeepsSidecarWithoutForce(t *testing.T) {
	dir, userNix := heldDevenvNixProject(t)

	out, err := executeInitCmd(t, dir, "--update", "--yes")
	if err != nil {
		t.Fatalf("second update: %v\n%s", err, out)
	}
	if got := readProjectFile(t, dir, "devenv.nix"); got != userNix {
		t.Error("devenv.nix was rewritten despite the user's edits")
	}
	if _, err := os.Stat(filepath.Join(dir, "devenv.nix.new")); err != nil {
		t.Errorf("devenv.nix.new missing after an update without --force: %v", err)
	}
	if !strings.Contains(out, "devenv.nix.new") {
		t.Errorf("update printed no merge instructions for devenv.nix.new:\n%s", out)
	}
}

// TestUpdate_DryRunKeepsSidecar: a dry run never removes a sidecar, even one
// a real forced update would drop.
func TestUpdate_DryRunKeepsSidecar(t *testing.T) {
	dir, _ := heldDevenvNixProject(t)
	before := readProjectFile(t, dir, "devenv.nix.new")

	if out, err := executeInitCmd(t, dir, "--update", "--dry-run", "--force"); err != nil {
		t.Fatalf("dry-run update: %v\n%s", err, out)
	}
	if got := readProjectFile(t, dir, "devenv.nix.new"); got != before {
		t.Error("dry-run update changed devenv.nix.new")
	}
	if _, err := os.Stat(filepath.Join(dir, "devenv.yaml.new")); err != nil {
		t.Errorf("dry-run update removed devenv.yaml.new: %v", err)
	}
}

// TestUpdate_NonManualMergeDotNewUntouched: qsdev writes sidecars only for
// manual-merge files, so a <file>.new beside any other regenerated file is
// the user's own and survives a forced update.
func TestUpdate_NonManualMergeDotNewUntouched(t *testing.T) {
	dir := initLifecycleProject(t)
	recorded := loadProjectState(t, dir).Files
	var target string
	for _, path := range slices.Sorted(maps.Keys(recorded)) {
		if recorded[path].Strategy == types.Overwrite {
			target = path
			break
		}
	}
	if target == "" {
		t.Fatal("lifecycle project records no overwrite-strategy file")
	}
	if err := os.WriteFile(filepath.Join(dir, target), []byte("# edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	const userContent = "# the user's own file\n"
	userNew := filepath.Join(dir, target+".new")
	if err := os.WriteFile(userNew, []byte(userContent), 0o644); err != nil {
		t.Fatal(err)
	}

	if out, err := executeInitCmd(t, dir, "--update", "--yes", "--force"); err != nil {
		t.Fatalf("forced update: %v\n%s", err, out)
	}
	if got := readProjectFile(t, dir, target); got == "# edited\n" {
		t.Fatalf("forced update did not rewrite %s; the setup does not exercise a write in place", target)
	}
	if got := readProjectFile(t, dir, target+".new"); got != userContent {
		t.Errorf("%s.new was changed or removed by a forced update: %q", target, got)
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
