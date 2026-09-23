package repair

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// writeFile writes content to root/rel, creating parent directories.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fileModificationReport(findings ...drift.Finding) *drift.Report {
	return &drift.Report{
		Categories:    []drift.Category{{Name: "File Modification", Findings: findings}},
		TotalFindings: len(findings),
	}
}

// TestRepair_HookAndMarkerDriftNeverFail runs repair against real drift
// detection in a directory that is not a git repository and has no CLAUDE.md.
// Hook and marker findings cannot be fixed by writing a generated file, so they
// must be reported (skipped) rather than failing with exit code 2.
func TestRepair_HookAndMarkerDriftNeverFail(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	report := drift.Detect(root, types.GeneratedState{}, map[string]bool{})

	result, _, err := Repair(root, RepairOptions{}, types.GeneratedState{}, map[string]types.GeneratedFile{}, report)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Failed) != 0 {
		t.Errorf("failed = %+v, want none", result.Failed)
	}
	if code := result.ExitCode(); code == 2 {
		t.Errorf("ExitCode() = 2 for unfixable-by-design findings")
	}
	for _, a := range result.Skipped {
		if a.File == ".git" && a.Severity != drift.Info {
			t.Errorf(".git finding severity = %q, want info", a.Severity)
		}
	}
}

// TestRepair_MarkerDriftResolvedByClaudeMDRegeneration verifies that marker
// findings are dropped once CLAUDE.md itself is regenerated in the same run.
func TestRepair_MarkerDriftResolvedByClaudeMDRegeneration(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	genState := state.RecordFiles([]types.GeneratedFile{{Path: "CLAUDE.md", Content: []byte("old"), Strategy: types.SectionMarker}})
	report := &drift.Report{Categories: []drift.Category{
		{Name: "File Modification", Findings: []drift.Finding{
			{Subject: "CLAUDE.md", Severity: drift.Error, FileStatus: types.Deleted, Description: "deleted"},
		}},
		{Name: "Section Marker Integrity", Findings: []drift.Finding{
			{Subject: "CLAUDE.md", Severity: drift.Error, Description: "CLAUDE.md does not exist"},
		}},
	}}
	fresh := map[string]types.GeneratedFile{"CLAUDE.md": {Path: "CLAUDE.md", Content: []byte("fresh"), Strategy: types.SectionMarker}}

	result, _, err := Repair(root, RepairOptions{}, genState, fresh, report)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Fixed) != 1 || len(result.Skipped) != 0 || result.ExitCode() != 0 {
		t.Errorf("fixed=%d skipped=%+v exit=%d; want the marker finding resolved", len(result.Fixed), result.Skipped, result.ExitCode())
	}
}

// TestRepair_BackupsOfSameBasenameAreKeptApart repairs two drifted files that
// share a basename; each must keep its own backup of its own content.
func TestRepair_BackupsOfSameBasenameAreKeptApart(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	paths := []string{".claude/skills/a/SKILL.md", ".claude/skills/b/SKILL.md"}
	genState := types.GeneratedState{Files: map[string]types.FileState{}}
	fresh := map[string]types.GeneratedFile{}
	var findings []drift.Finding
	for _, p := range paths {
		writeFile(t, root, p, "USER EDIT "+p)
		genState.Files[p] = types.FileState{Hash: state.ComputeHash([]byte("orig")), Strategy: types.LibraryManaged}
		fresh[p] = types.GeneratedFile{Path: p, Content: []byte("fresh"), Strategy: types.LibraryManaged}
		findings = append(findings, drift.Finding{Subject: p, Severity: drift.Warning, FileStatus: types.Modified})
	}

	result, _, err := Repair(root, RepairOptions{}, genState, fresh, fileModificationReport(findings...))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Fixed) != 2 {
		t.Fatalf("fixed = %d, want 2 (failed: %+v)", len(result.Fixed), result.Failed)
	}

	seen := map[string]bool{}
	for _, a := range result.Fixed {
		if seen[a.BackupPath] {
			t.Fatalf("two files share backup %s", a.BackupPath)
		}
		seen[a.BackupPath] = true
		data, err := os.ReadFile(a.BackupPath)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "USER EDIT "+a.File {
			t.Errorf("backup of %s = %q, want its own original content", a.File, data)
		}
	}
}

// TestCreateBackup_SameSecondDoesNotOverwrite verifies that repeated backups of
// the same file never overwrite each other.
func TestCreateBackup_SameSecondDoesNotOverwrite(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	var backups []string
	for i := range 3 {
		writeFile(t, root, "f.txt", strings.Repeat("x", i+1))
		b, err := createBackup(root, "f.txt")
		if err != nil {
			t.Fatal(err)
		}
		backups = append(backups, b)
	}
	for i, b := range backups {
		data, err := os.ReadFile(b)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != strings.Repeat("x", i+1) {
			t.Errorf("backup %d = %q, overwritten by a later backup", i, data)
		}
	}
}

func TestCreateBackup_RejectsPathOutsideProject(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile(t, root, "inside.txt", "x")
	if _, err := createBackup(filepath.Join(root, "sub"), "../inside.txt"); err == nil {
		t.Error("expected an error for a path escaping the project")
	}
}

// TestRepair_RecordsThreeWayMergeBase verifies that a regenerated
// three-way-merge file keeps its merge base in state, as generation does.
func TestRepair_RecordsThreeWayMergeBase(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	const rel = ".claude/settings.json"
	genState := state.RecordFiles([]types.GeneratedFile{{Path: rel, Content: []byte("{}"), Strategy: types.ThreeWayMerge}})
	fresh := map[string]types.GeneratedFile{rel: {Path: rel, Content: []byte(`{"fresh":true}`), Strategy: types.ThreeWayMerge}}
	report := fileModificationReport(drift.Finding{Subject: rel, Severity: drift.Error, FileStatus: types.Deleted})

	result, updated, err := Repair(root, RepairOptions{}, genState, fresh, report)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Fixed) != 1 {
		t.Fatalf("fixed = %d, want 1", len(result.Fixed))
	}
	if got := string(updated.Files[rel].BaseContent); got != `{"fresh":true}` {
		t.Errorf("BaseContent = %q, want the regenerated content", got)
	}
}

// TestRepair_UserEditedFileExitsZero verifies that an expected edit to a
// user-owned file (an Info finding) does not make repair exit non-zero.
func TestRepair_UserEditedFileExitsZero(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile(t, root, "CLAUDE.md", "edited")
	genState := state.RecordFiles([]types.GeneratedFile{{Path: "CLAUDE.md", Content: []byte("orig"), Strategy: types.SectionMarker}})
	genState.QsdevVersion = "0.1.0"
	report := drift.Detect(root, genState, map[string]bool{})
	// Keep only the categories this test is about.
	var cats []drift.Category
	for _, c := range report.Categories {
		if c.Name == "File Modification" || c.Name == "Version Drift" {
			cats = append(cats, c)
		}
	}
	report.Categories = cats

	result, _, err := Repair(root, RepairOptions{}, genState, map[string]types.GeneratedFile{}, report)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Skipped) == 0 {
		t.Fatal("expected the user-edited file to be reported as skipped")
	}
	if code := result.ExitCode(); code != 0 {
		t.Errorf("ExitCode() = %d, want 0 for informational findings", code)
	}
}

// TestRepair_ResetRegeneratesAllFiles verifies that --reset rewrites every
// generated file, not only drifted ones, while still protecting devenv.nix.
func TestRepair_ResetRegeneratesAllFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile(t, root, "a.txt", "stale template")
	writeFile(t, root, "devenv.nix", "user devenv")
	genState := state.RecordFiles([]types.GeneratedFile{
		{Path: "a.txt", Content: []byte("stale template")},
		{Path: "devenv.nix", Content: []byte("user devenv")},
	})
	fresh := map[string]types.GeneratedFile{
		"a.txt":      {Path: "a.txt", Content: []byte("new template")},
		"b.txt":      {Path: "b.txt", Content: []byte("new file")},
		"devenv.nix": {Path: "devenv.nix", Content: []byte("fresh devenv")},
	}

	for _, report := range []*drift.Report{nil, {}} {
		dir := root
		result, _, err := Repair(dir, RepairOptions{Reset: true}, genState, fresh, report)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Fixed) != 2 || len(result.Failed) != 0 {
			t.Errorf("fixed=%+v failed=%+v; want a.txt and b.txt regenerated", result.Fixed, result.Failed)
		}
		for rel, want := range map[string]string{"a.txt": "new template", "b.txt": "new file", "devenv.nix": "user devenv"} {
			data, err := os.ReadFile(filepath.Join(dir, rel))
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != want {
				t.Errorf("%s = %q, want %q", rel, data, want)
			}
		}
	}
}

// TestRepair_TargetFileNormalized verifies that --file accepts non-canonical
// spellings of a project path and rejects paths qsdev does not manage.
func TestRepair_TargetFileNormalized(t *testing.T) {
	t.Parallel()

	const rel = ".claude/settings.json"
	root := t.TempDir()

	tests := []struct {
		name      string
		target    string
		wantFixed int
		wantErr   bool
	}{
		{name: "canonical", target: rel, wantFixed: 1},
		{name: "dot slash", target: "./" + rel, wantFixed: 1},
		{name: "backslashes", target: `.claude\settings.json`, wantFixed: 1},
		{name: "absolute", target: filepath.Join(root, ".claude", "settings.json"), wantFixed: 1},
		{name: "redundant segments", target: ".claude/../.claude/settings.json", wantFixed: 1},
		{name: "managed but healthy", target: "other.txt", wantFixed: 0},
		{name: "not managed", target: "nope.txt", wantErr: true},
		{name: "outside project", target: "../x", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			genState := state.RecordFiles([]types.GeneratedFile{
				{Path: rel, Content: []byte("{}")},
				{Path: "other.txt", Content: []byte("x")},
			})
			fresh := map[string]types.GeneratedFile{rel: {Path: rel, Content: []byte("{}")}}
			report := fileModificationReport(drift.Finding{Subject: rel, Severity: drift.Error, FileStatus: types.Deleted})

			result, _, err := Repair(root, RepairOptions{DryRun: true, TargetFile: tt.target}, genState, fresh, report)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Fixed) != tt.wantFixed {
				t.Errorf("fixed = %d, want %d", len(result.Fixed), tt.wantFixed)
			}
		})
	}
}

// TestClassifyFileModification_UsesTypedStatus verifies that deletion is
// decided by the finding's typed status, not by its human-readable text.
func TestClassifyFileModification_UsesTypedStatus(t *testing.T) {
	t.Parallel()

	genState := types.GeneratedState{Files: map[string]types.FileState{
		"CLAUDE.md": {Strategy: types.SectionMarker},
	}}

	tests := []struct {
		name    string
		finding drift.Finding
		want    RepairActionType
	}{
		{
			name:    "deleted with reworded description",
			finding: drift.Finding{Subject: "CLAUDE.md", FileStatus: types.Deleted, Description: "CLAUDE.md was removed"},
			want:    ActionRegenerate,
		},
		{
			name:    "modified whose text mentions deletion",
			finding: drift.Finding{Subject: "CLAUDE.md", FileStatus: types.Modified, Description: `Human-edited file "has been deleted.md" has been modified`},
			want:    ActionSkip,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := classifyFileModification(tt.finding, genState, RepairOptions{}).ActionType; got != tt.want {
				t.Errorf("ActionType = %d, want %d", got, tt.want)
			}
		})
	}
}
