package generate_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/merge"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestWriteFiles_RecordedStateMatchesDiskAfterMerge is the init/join flow:
// WriteFiles merges into pre-existing files, then the caller records
// SuccessfulFiles. The recorded hash must match what is on disk, and the
// three-way merge base must stay the generated content.
func TestWriteFiles_RecordedStateMatchesDiskAfterMerge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		file     types.GeneratedFile
		existing string
	}{
		{
			name: "section_marker_with_user_text",
			file: types.GeneratedFile{
				Path:     "CLAUDE.md",
				Content:  []byte("<!-- BEGIN GENERATED SECTION -->\nnew\n<!-- END GENERATED SECTION -->\n"),
				Mode:     0o644,
				Strategy: types.SectionMarker,
			},
			existing: "# my notes\n\n<!-- BEGIN GENERATED SECTION -->\nold\n<!-- END GENERATED SECTION -->\n\nmore notes\n",
		},
		{
			name: "section_marker_handwritten_no_markers",
			file: types.GeneratedFile{
				Path:     "CLAUDE.md",
				Content:  []byte("<!-- BEGIN GENERATED SECTION -->\nnew\n<!-- END GENERATED SECTION -->\n"),
				Mode:     0o644,
				Strategy: types.SectionMarker,
			},
			existing: "# my handwritten notes\n",
		},
		{
			name: "three_way_settings_with_user_env",
			file: types.GeneratedFile{
				Path:     ".claude/settings.json",
				Content:  []byte(`{"permissions":{"allow":[],"deny":["Read(./.env)"]}}`),
				Mode:     0o644,
				Strategy: types.ThreeWayMerge,
			},
			existing: `{"env":{"FOO":"bar"},"permissions":{"allow":[],"deny":[]}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			abs := filepath.Join(dir, tt.file.Path)
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(abs, []byte(tt.existing), 0o644); err != nil {
				t.Fatal(err)
			}

			files := []types.GeneratedFile{tt.file}
			res, err := generate.WriteFiles(files, generate.PipelineOptions{
				ProjectRoot:       dir,
				SkipValidate:      true,
				SectionMergeFunc:  merge.SectionMarkersOrAppend,
				ThreeWayMergeFunc: merge.MergeOnCreate,
			})
			if err != nil {
				t.Fatalf("WriteFiles: %v", err)
			}
			if res.HasFailures() {
				t.Fatalf("unexpected failures: %+v", res.FailedFiles())
			}

			gs := state.RecordFiles(res.SuccessfulFiles(files))
			status := state.CheckModified(gs, dir)[tt.file.Path]
			if status.Status != types.Unmodified {
				t.Errorf("status right after write = %v, want unmodified (stored %s, disk %s)",
					status.Status, status.StoredHash, status.CurrentHash)
			}
			if tt.file.Strategy == types.ThreeWayMerge {
				if base := gs.Files[tt.file.Path].BaseContent; !bytes.Equal(base, tt.file.Content) {
					t.Errorf("BaseContent = %q, want generated content %q", base, tt.file.Content)
				}
			}
		})
	}
}

func TestWriteFiles_IdenticalFileIsSkipped(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	files := []types.GeneratedFile{
		{Path: "devenv.nix", Content: []byte("{ }\n"), Mode: 0o644},
	}
	opts := generate.PipelineOptions{ProjectRoot: dir, SkipValidate: true}

	if _, err := generate.WriteFiles(files, opts); err != nil {
		t.Fatalf("first WriteFiles: %v", err)
	}
	abs := filepath.Join(dir, "devenv.nix")
	past := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(abs, past, past); err != nil {
		t.Fatal(err)
	}

	res, err := generate.WriteFiles(files, opts)
	if err != nil {
		t.Fatalf("second WriteFiles: %v", err)
	}
	if res.Skipped != 1 || res.Updated != 0 || res.Created != 0 {
		t.Fatalf("second run = %s, want 1 skipped only", res.Summary())
	}
	fr := res.Files[0]
	if fr.Action != generate.ActionSkipped {
		t.Errorf("Action = %v, want skipped", fr.Action)
	}
	if want := state.ComputeHash(files[0].Content); fr.PrevHash != want {
		t.Errorf("PrevHash = %q, want %q", fr.PrevHash, want)
	}
	info, err := os.Stat(abs)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(past) {
		t.Errorf("identical file was rewritten: mtime %v, want %v", info.ModTime(), past)
	}
	// A skipped file is still present with its intended content, so it must
	// be recorded in state rather than dropped.
	if got := res.SuccessfulFiles(files); len(got) != 1 {
		t.Errorf("SuccessfulFiles = %d entries, want 1 (skipped file must be recorded)", len(got))
	}
}

func TestWriteFiles_ChangedFileRecordsPrevHash(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	abs := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(abs, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := generate.WriteFiles([]types.GeneratedFile{
		{Path: "a.txt", Content: []byte("new\n"), Mode: 0o644},
	}, generate.PipelineOptions{ProjectRoot: dir, SkipValidate: true})
	if err != nil {
		t.Fatalf("WriteFiles: %v", err)
	}
	fr := res.Files[0]
	if fr.Action != generate.ActionUpdated {
		t.Fatalf("Action = %v, want updated", fr.Action)
	}
	if want := state.ComputeHash([]byte("old\n")); fr.PrevHash != want {
		t.Errorf("PrevHash = %q, want %q", fr.PrevHash, want)
	}
}
