package devinit

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestExecuteUpdatePlan_RefusesInvalidContent is the regression test for
// update writing content the init pipeline rejects: every write action now
// validates, records a per-file failure and leaves the file untouched.
func TestExecuteUpdatePlan_RefusesInvalidContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		action UpdateAction
	}{
		{"regenerate", UpdateActionRegenerate},
		{"create", UpdateActionCreate},
		{"sidecar", UpdateActionSidecar},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			const rel = ".claude/settings.json"
			const original = "{\"permissions\": {}}\n"
			if tt.action != UpdateActionCreate {
				if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, rel), []byte(original), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			plan := UpdatePlan{Files: []FileUpdatePlan{{
				Path: rel, Action: tt.action, Strategy: types.Overwrite, NewContent: []byte(`{"permissions": `),
			}}}

			out, err := executeUpdatePlan(plan, root, UpdateOptions{})
			if err != nil {
				t.Fatalf("executeUpdatePlan: %v", err)
			}
			if len(out.failures) != 1 || !errors.Is(out.failures[0].Err, generate.ErrInvalidContent) {
				t.Fatalf("failures = %+v, want one invalid-content failure", out.failures)
			}
			if len(out.written) != 0 {
				t.Errorf("invalid content recorded as written: %+v", out.written)
			}
			data, readErr := os.ReadFile(filepath.Join(root, rel))
			if tt.action == UpdateActionCreate {
				if !os.IsNotExist(readErr) {
					t.Errorf("invalid file was created (err=%v)", readErr)
				}
				return
			}
			if string(data) != original {
				t.Errorf("file overwritten with invalid content: %q", data)
			}
		})
	}
}

// TestValidateToolChange_RefusesWholeChange checks enable/disable refuse a
// tool change carrying invalid content before writing any file.
func TestValidateToolChange_RefusesWholeChange(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	change := toolChange{
		exclusive: []types.GeneratedFile{
			{Path: "ok.json", Content: []byte(`{}`)},
			{Path: "bad.yaml", Content: []byte("a: [\n")},
		},
	}
	_, err := applyToolChange(root, change, types.GeneratedState{Files: map[string]types.FileState{}})
	if !errors.Is(err, generate.ErrInvalidContent) {
		t.Fatalf("err = %v, want invalid content", err)
	}
	for _, name := range []string{"ok.json", "bad.yaml"} {
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Errorf("%s written by a refused change (err=%v)", name, err)
		}
	}
}

// TestUpdate_PoisonedPackageDoesNotBreakDevenvNix reproduces the reported
// sequence: answers carrying a package name that is not a Nix attribute path
// made the next `qsdev update` write an unparseable devenv.nix and exit 0.
// Needs nix-instantiate, the validator for .nix files.
func TestUpdate_PoisonedPackageDoesNotBreakDevenvNix(t *testing.T) {
	if _, err := exec.LookPath("nix-instantiate"); err != nil {
		t.Skip("nix-instantiate not on PATH")
	}
	dir := initLifecycleProject(t)
	before := readProjectFile(t, dir, "devenv.nix")

	answers := loadProjectAnswers(t, dir)
	answers.ExtraPackages = append(answers.ExtraPackages, "foo; rm -rf ~")
	if err := saveAnswers(dir, answers); err != nil {
		t.Fatal(err)
	}

	out, err := executeInitCmd(t, dir, "--update")
	if err == nil {
		t.Fatalf("update succeeded while generating an unparseable devenv.nix:\n%s", out)
	}
	if !strings.Contains(out, "devenv.nix") {
		t.Errorf("update output does not name devenv.nix:\n%s", out)
	}
	if got := readProjectFile(t, dir, "devenv.nix"); got != before {
		t.Errorf("devenv.nix was rewritten:\n%s", got)
	}
}
