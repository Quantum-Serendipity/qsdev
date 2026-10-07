package generate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestWriteFiles_HeldWithFollowsPartner checks a file held with a partner
// (devenv.yaml with devenv.nix) is written in place only when the partner is:
// when the partner keeps the user's edits and gets a sidecar, the file goes
// to a sidecar too, so devenv.yaml never gains inputs for options the
// devenv.nix in effect does not set. The order of the files does not matter.
func TestWriteFiles_HeldWithFollowsPartner(t *testing.T) {
	t.Parallel()

	const (
		oldYAML = "inputs: {}\n"
		newYAML = "inputs:\n  go-overlay: {}\n"
		newNix  = "{ languages.go.version = \"1.26.7\"; }\n"
	)
	tests := []struct {
		name         string
		existingNix  string // "" means devenv.nix does not exist
		existingYAML string // "" means devenv.yaml does not exist
		force        bool
		yamlFirst    bool
		wantYAML     string
		wantSidecar  bool
		wantAction   generate.FileAction
	}{
		{"partner held: sidecar", "{ user }\n", oldYAML, false, false, oldYAML, true, generate.ActionSidecar},
		{"partner held, listed after: sidecar", "{ user }\n", oldYAML, false, true, oldYAML, true, generate.ActionSidecar},
		{"partner held, already current: unchanged", "{ user }\n", newYAML, false, false, newYAML, false, generate.ActionUnchanged},
		{"partner held, file missing: created", "{ user }\n", "", false, false, newYAML, false, generate.ActionCreated},
		{"partner written in place: updated", newNix, oldYAML, false, false, newYAML, false, generate.ActionUpdated},
		{"partner forced: updated", "{ user }\n", oldYAML, true, false, newYAML, false, generate.ActionUpdated},
		{"partner created: updated", "", oldYAML, false, false, newYAML, false, generate.ActionUpdated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tt.existingNix != "" {
				writeTestFile(t, filepath.Join(dir, "devenv.nix"), tt.existingNix, 0o644)
			}
			if tt.existingYAML != "" {
				writeTestFile(t, filepath.Join(dir, "devenv.yaml"), tt.existingYAML, 0o644)
			}
			nix := types.GeneratedFile{Path: "devenv.nix", Content: []byte(newNix), Strategy: types.ManualMerge, SkipValidation: true}
			yml := types.GeneratedFile{Path: "devenv.yaml", Content: []byte(newYAML), Strategy: types.Overwrite, SkipValidation: true, HeldWith: "devenv.nix"}
			files := []types.GeneratedFile{nix, yml}
			if tt.yamlFirst {
				files = []types.GeneratedFile{yml, nix}
			}

			result, err := generate.WriteFiles(files, generate.PipelineOptions{ProjectRoot: dir, Force: tt.force})
			if err != nil {
				t.Fatalf("WriteFiles: %v", err)
			}
			var fr generate.FileResult
			for _, f := range result.Files {
				if f.Path == "devenv.yaml" {
					fr = f
				}
			}
			if fr.Action != tt.wantAction {
				t.Errorf("devenv.yaml action = %v, want %v (err %v)", fr.Action, tt.wantAction, fr.Error)
			}
			got, err := os.ReadFile(filepath.Join(dir, "devenv.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.wantYAML {
				t.Errorf("devenv.yaml = %q, want %q", got, tt.wantYAML)
			}
			sidecar, err := os.ReadFile(filepath.Join(dir, "devenv.yaml"+generate.SidecarSuffix))
			switch {
			case tt.wantSidecar && string(sidecar) != newYAML:
				t.Errorf("devenv.yaml.new = %q (%v), want the generated content", sidecar, err)
			case !tt.wantSidecar && err == nil:
				t.Errorf("unexpected devenv.yaml.new: %q", sidecar)
			}
		})
	}
}
