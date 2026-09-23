package generate_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestWriteFiles_EnforcesStrategyOnExistingFile covers every MergeStrategy
// against an existing file with user content, with no merge callbacks set.
func TestWriteFiles_EnforcesStrategyOnExistingFile(t *testing.T) {
	t.Parallel()

	const userSection = "<!-- BEGIN GENERATED SECTION -->\nold\n<!-- END GENERATED SECTION -->\nUser notes\n"
	const genSection = "<!-- BEGIN GENERATED SECTION -->\nnew\n<!-- END GENERATED SECTION -->\nDefault\n"

	tests := []struct {
		name        string
		path        string
		strategy    types.MergeStrategy
		force       bool
		existing    string
		generated   string
		wantAction  generate.FileAction
		wantContent string
		wantSidecar bool
		// wantRecorded: SuccessfulFiles reports the file for state recording.
		wantRecorded bool
	}{
		{
			name: "overwrite replaces", path: "o.txt", strategy: types.Overwrite,
			existing: "user\n", generated: "gen\n",
			wantAction: generate.ActionUpdated, wantContent: "gen\n", wantRecorded: true,
		},
		{
			name: "library-managed replaces", path: "l.txt", strategy: types.LibraryManaged,
			existing: "user\n", generated: "gen\n",
			wantAction: generate.ActionUpdated, wantContent: "gen\n", wantRecorded: true,
		},
		{
			name: "skip keeps user file", path: "Directory.Build.props", strategy: types.Skip,
			existing: "<Project>user</Project>\n", generated: "<Project>qsdev</Project>\n",
			wantAction: generate.ActionSkipped, wantContent: "<Project>user</Project>\n",
		},
		{
			name: "skip is not overridden by force", path: ".envrc", strategy: types.Skip, force: true,
			existing: "use flake\n", generated: "use devenv\n",
			wantAction: generate.ActionSkipped, wantContent: "use flake\n",
		},
		{
			// Identical content is qsdev output: it is not rewritten (F392 keeps
			// the mtime) but is still recorded in state.
			name: "skip records identical content", path: "same.txt", strategy: types.Skip,
			existing: "gen\n", generated: "gen\n",
			wantAction: generate.ActionSkipped, wantContent: "gen\n", wantRecorded: true,
		},
		{
			name: "manual merge writes sidecar for unrecorded changes", path: "devenv.nix", strategy: types.ManualMerge,
			existing: "{ user }\n", generated: "{ gen }\n",
			wantAction: generate.ActionSkipped, wantContent: "{ user }\n", wantSidecar: true,
		},
		{
			name: "manual merge overwrites with force", path: "devenv.nix", strategy: types.ManualMerge, force: true,
			existing: "{ user }\n", generated: "{ gen }\n",
			wantAction: generate.ActionUpdated, wantContent: "{ gen }\n", wantRecorded: true,
		},
		{
			name: "section marker merges by default", path: "CLAUDE.md", strategy: types.SectionMarker,
			existing: userSection, generated: genSection,
			wantAction:  generate.ActionUpdated,
			wantContent: "<!-- BEGIN GENERATED SECTION -->\nnew\n<!-- END GENERATED SECTION -->\nUser notes\n", wantRecorded: true,
		},
		{
			name: "three-way merge preserves user keys by default", path: ".claude/settings.json", strategy: types.ThreeWayMerge,
			existing:   `{"env":{"TOKEN":"x"},"permissions":{"allow":[],"deny":[]}}`,
			generated:  `{"permissions":{"allow":["Read(*)"],"deny":[]}}`,
			wantAction: generate.ActionUpdated, wantContent: `"TOKEN": "x"`, wantRecorded: true,
		},
		{
			name: "append fails instead of overwriting", path: "a.txt", strategy: types.Append,
			existing: "user\n", generated: "gen\n",
			wantAction: generate.ActionFailed, wantContent: "user\n",
		},
		{
			name: "merge fails instead of overwriting", path: "m.txt", strategy: types.Merge,
			existing: "user\n", generated: "gen\n",
			wantAction: generate.ActionFailed, wantContent: "user\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			full := filepath.Join(dir, tt.path)
			writeTestFile(t, full, tt.existing, 0o644)

			files := []types.GeneratedFile{{
				Path: tt.path, Content: []byte(tt.generated), Strategy: tt.strategy, SkipValidation: true,
			}}
			result, err := generate.WriteFiles(files, generate.PipelineOptions{ProjectRoot: dir, Force: tt.force})
			if err != nil {
				t.Fatalf("WriteFiles: %v", err)
			}
			if recorded := len(result.SuccessfulFiles(files)) == 1; recorded != tt.wantRecorded {
				t.Errorf("recorded in state = %v, want %v", recorded, tt.wantRecorded)
			}
			if len(result.Files) != 1 {
				t.Fatalf("expected 1 result, got %d", len(result.Files))
			}
			fr := result.Files[0]
			if fr.Action != tt.wantAction {
				t.Errorf("action = %v, want %v (err: %v)", fr.Action, tt.wantAction, fr.Error)
			}

			got, err := os.ReadFile(full)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(got), tt.wantContent) {
				t.Errorf("content = %q, want it to contain %q", got, tt.wantContent)
			}

			sidecar := full + generate.SidecarSuffix
			_, statErr := os.Stat(sidecar)
			if tt.wantSidecar {
				if statErr != nil {
					t.Fatalf("expected sidecar %s: %v", sidecar, statErr)
				}
				if data, _ := os.ReadFile(sidecar); string(data) != tt.generated {
					t.Errorf("sidecar content = %q, want %q", data, tt.generated)
				}
				if fr.SidecarPath != tt.path+generate.SidecarSuffix {
					t.Errorf("SidecarPath = %q, want %q", fr.SidecarPath, tt.path+generate.SidecarSuffix)
				}
				if !strings.Contains(result.Summary(), fr.SidecarPath) {
					t.Errorf("Summary() should mention the sidecar, got %q", result.Summary())
				}
			} else if statErr == nil {
				t.Errorf("unexpected sidecar %s", sidecar)
			}

		})
	}
}

// TestWriteFiles_ManualMergeRegeneratesRecordedOutput verifies a ManualMerge
// file that still matches a hash recorded in the project's state is
// regenerated in place (e.g. devenv add-package), while one that doesn't match
// gets a sidecar.
func TestWriteFiles_ManualMergeRegeneratesRecordedOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		onDisk      string
		wantContent string
		wantAction  generate.FileAction
	}{
		{name: "unmodified recorded output", onDisk: "{ v1 }\n", wantContent: "{ v2 }\n", wantAction: generate.ActionUpdated},
		{name: "hand-edited", onDisk: "{ v1 + edits }\n", wantContent: "{ v1 + edits }\n", wantAction: generate.ActionSkipped},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			writeTestFile(t, filepath.Join(dir, "devenv.nix"), tt.onDisk, 0o644)

			recorded := state.RecordFiles([]types.GeneratedFile{
				{Path: "devenv.nix", Content: []byte("{ v1 }\n"), Mode: 0o644, Strategy: types.ManualMerge},
			})
			statePath := filepath.Join(dir, state.StateFilePaths()[1])
			if err := state.SaveStateToFile(statePath, recorded); err != nil {
				t.Fatal(err)
			}

			result, err := generate.WriteFiles([]types.GeneratedFile{{
				Path: "devenv.nix", Content: []byte("{ v2 }\n"), Strategy: types.ManualMerge, SkipValidation: true,
			}}, generate.PipelineOptions{ProjectRoot: dir})
			if err != nil {
				t.Fatalf("WriteFiles: %v", err)
			}
			if got := result.Files[0].Action; got != tt.wantAction {
				t.Errorf("action = %v, want %v", got, tt.wantAction)
			}
			got, _ := os.ReadFile(filepath.Join(dir, "devenv.nix"))
			if string(got) != tt.wantContent {
				t.Errorf("devenv.nix = %q, want %q", got, tt.wantContent)
			}
		})
	}
}

// TestWriteFiles_SymlinkedAncestorEscape verifies the escape check resolves
// the longest existing ancestor, so a symlinked directory pointing outside the
// root is caught even when deeper directories do not exist yet.
func TestWriteFiles_SymlinkedAncestorEscape(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on Windows")
	}

	tests := []struct {
		name string
		path string
	}{
		{name: "existing parent", path: ".claude/y.sh"},
		{name: "missing parent", path: ".claude/hooks/x.sh"},
		{name: "missing deep parent", path: ".claude/a/b/c.sh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			outside := t.TempDir()
			if err := os.Symlink(outside, filepath.Join(root, ".claude")); err != nil {
				t.Fatal(err)
			}

			result, err := generate.WriteFiles([]types.GeneratedFile{{
				Path: tt.path, Content: []byte("#!/bin/sh\n"), Mode: 0o755, SkipValidation: true,
			}}, generate.PipelineOptions{ProjectRoot: root})
			if err != nil {
				t.Fatalf("WriteFiles: %v", err)
			}
			if result.Failed != 1 {
				t.Errorf("Failed = %d, want 1", result.Failed)
			}
			rel, _ := filepath.Rel(".claude", tt.path)
			if _, err := os.Stat(filepath.Join(outside, rel)); err == nil {
				t.Errorf("file was written outside the project root at %s", filepath.Join(outside, rel))
			}
		})
	}
}

// TestWriteFiles_SymlinkedLeafEscape verifies a symlinked target file that
// points outside the root is rejected rather than read for merging.
func TestWriteFiles_SymlinkedLeafEscape(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on Windows")
	}

	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.json")
	writeTestFile(t, outside, `{"permissions":{"allow":[],"deny":[]}}`, 0o600)
	if err := os.Symlink(outside, filepath.Join(root, "settings.json")); err != nil {
		t.Fatal(err)
	}

	result, err := generate.WriteFiles([]types.GeneratedFile{{
		Path: "settings.json", Content: []byte(`{"permissions":{"allow":[],"deny":[]}}`), Strategy: types.ThreeWayMerge,
	}}, generate.PipelineOptions{ProjectRoot: root})
	if err != nil {
		t.Fatalf("WriteFiles: %v", err)
	}
	if result.Failed != 1 {
		t.Errorf("Failed = %d, want 1", result.Failed)
	}
}

// TestWriteFiles_NeverWidensExistingMode verifies rewriting an existing file
// keeps a restrictive mode, and that the written mode is what SuccessfulFiles
// reports (and therefore what state records).
func TestWriteFiles_NeverWidensExistingMode(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not enforced on Windows")
	}

	tests := []struct {
		name      string
		existing  os.FileMode
		generated os.FileMode
		want      os.FileMode
	}{
		{name: "private file stays private", existing: 0o600, generated: 0o644, want: 0o600},
		{name: "default mode unchanged", existing: 0o644, generated: 0o644, want: 0o644},
		{name: "owner exec bit still applied", existing: 0o644, generated: 0o755, want: 0o744},
		{name: "group-writable not widened to others", existing: 0o660, generated: 0o644, want: 0o640},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			full := filepath.Join(dir, "settings.json")
			writeTestFile(t, full, `{"permissions":{"allow":[],"deny":[]},"env":{"K":"v"}}`, tt.existing)

			files := []types.GeneratedFile{{
				Path: "settings.json", Content: []byte(`{"permissions":{"allow":[],"deny":[]}}`),
				Mode: tt.generated, Strategy: types.ThreeWayMerge,
			}}
			result, err := generate.WriteFiles(files, generate.PipelineOptions{ProjectRoot: dir})
			if err != nil {
				t.Fatalf("WriteFiles: %v", err)
			}
			if result.Failed != 0 {
				t.Fatalf("unexpected failure: %v", result.Files[0].Error)
			}

			info, err := os.Stat(full)
			if err != nil {
				t.Fatal(err)
			}
			if got := info.Mode().Perm(); got != tt.want {
				t.Errorf("mode = %o, want %o", got, tt.want)
			}
			written := result.SuccessfulFiles(files)
			if len(written) != 1 || written[0].Mode != tt.want {
				t.Errorf("SuccessfulFiles mode = %v, want %o", written, tt.want)
			}
		})
	}
}

func writeTestFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}
