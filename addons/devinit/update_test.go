package devinit

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestBuildUpdatePlan_UnmodifiedRegenerate(t *testing.T) {
	files := []types.GeneratedFile{
		{Path: "devenv.yaml", Content: []byte("new"), Mode: 0o644, Strategy: types.Overwrite},
	}
	modStatus := map[string]state.FileStatus{
		"devenv.yaml": {Path: "devenv.yaml", Status: types.Unmodified},
	}
	stored := types.GeneratedState{
		Files: map[string]types.FileState{
			"devenv.yaml": {Hash: "sha256:abc", Strategy: types.Overwrite},
		},
	}
	plan := buildUpdatePlan(files, modStatus, stored, t.TempDir(), UpdateOptions{})
	if len(plan.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(plan.Files))
	}
	if plan.Files[0].Action != UpdateActionRegenerate {
		t.Errorf("expected Regenerate, got %v", plan.Files[0].Action)
	}
	if plan.Files[0].Status != types.Unmodified {
		t.Errorf("expected Unmodified status, got %v", plan.Files[0].Status)
	}
}

func TestBuildUpdatePlan_ModifiedNoForce_ThreeWayMerge(t *testing.T) {
	files := []types.GeneratedFile{
		{Path: ".claude/settings.json", Content: []byte("new"), Mode: 0o644, Strategy: types.ThreeWayMerge},
	}
	modStatus := map[string]state.FileStatus{
		".claude/settings.json": {Path: ".claude/settings.json", Status: types.Modified},
	}
	stored := types.GeneratedState{
		Files: map[string]types.FileState{
			".claude/settings.json": {
				Hash:        "sha256:abc",
				Strategy:    types.ThreeWayMerge,
				BaseContent: []byte("base"),
			},
		},
	}
	plan := buildUpdatePlan(files, modStatus, stored, t.TempDir(), UpdateOptions{})
	if len(plan.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(plan.Files))
	}
	if plan.Files[0].Action != UpdateActionMerge {
		t.Errorf("expected Merge, got %v", plan.Files[0].Action)
	}
	if string(plan.Files[0].OldContent) != "base" {
		t.Errorf("expected OldContent to be 'base', got %q", string(plan.Files[0].OldContent))
	}
}

func TestBuildUpdatePlan_ModifiedNoForce_SectionMarker(t *testing.T) {
	files := []types.GeneratedFile{
		{Path: "CLAUDE.md", Content: []byte("new"), Mode: 0o644, Strategy: types.SectionMarker},
	}
	modStatus := map[string]state.FileStatus{
		"CLAUDE.md": {Path: "CLAUDE.md", Status: types.Modified},
	}
	stored := types.GeneratedState{
		Files: map[string]types.FileState{
			"CLAUDE.md": {Hash: "sha256:abc", Strategy: types.SectionMarker},
		},
	}
	plan := buildUpdatePlan(files, modStatus, stored, t.TempDir(), UpdateOptions{})
	if len(plan.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(plan.Files))
	}
	if plan.Files[0].Action != UpdateActionMerge {
		t.Errorf("expected Merge, got %v", plan.Files[0].Action)
	}
	if plan.Files[0].Reason != "modified, section marker merge" {
		t.Errorf("unexpected reason: %q", plan.Files[0].Reason)
	}
}

func TestBuildUpdatePlan_ModifiedNoForce_ManualMerge(t *testing.T) {
	files := []types.GeneratedFile{
		{Path: "devenv.nix", Content: []byte("new"), Mode: 0o644, Strategy: types.ManualMerge},
	}
	modStatus := map[string]state.FileStatus{
		"devenv.nix": {Path: "devenv.nix", Status: types.Modified},
	}
	stored := types.GeneratedState{
		Files: map[string]types.FileState{
			"devenv.nix": {Hash: "sha256:abc", Strategy: types.ManualMerge},
		},
	}
	plan := buildUpdatePlan(files, modStatus, stored, t.TempDir(), UpdateOptions{})
	if len(plan.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(plan.Files))
	}
	if plan.Files[0].Action != UpdateActionSidecar {
		t.Errorf("expected Sidecar, got %v", plan.Files[0].Action)
	}
}

func TestBuildUpdatePlan_ModifiedNoForce_LibraryManaged(t *testing.T) {
	files := []types.GeneratedFile{
		{Path: ".claude/skills/deploy.md", Content: []byte("new"), Mode: 0o644, Strategy: types.LibraryManaged},
	}
	modStatus := map[string]state.FileStatus{
		".claude/skills/deploy.md": {Path: ".claude/skills/deploy.md", Status: types.Modified},
	}
	stored := types.GeneratedState{
		Files: map[string]types.FileState{
			".claude/skills/deploy.md": {Hash: "sha256:abc", Strategy: types.LibraryManaged},
		},
	}
	plan := buildUpdatePlan(files, modStatus, stored, t.TempDir(), UpdateOptions{})
	if len(plan.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(plan.Files))
	}
	if plan.Files[0].Action != UpdateActionRegenerate {
		t.Errorf("expected Regenerate, got %v", plan.Files[0].Action)
	}
	if plan.Files[0].Reason != "library-managed, updating to latest" {
		t.Errorf("unexpected reason: %q", plan.Files[0].Reason)
	}
}

func TestBuildUpdatePlan_ModifiedNoForce_Overwrite(t *testing.T) {
	files := []types.GeneratedFile{
		{Path: "devenv.yaml", Content: []byte("new"), Mode: 0o644, Strategy: types.Overwrite},
	}
	modStatus := map[string]state.FileStatus{
		"devenv.yaml": {Path: "devenv.yaml", Status: types.Modified},
	}
	stored := types.GeneratedState{
		Files: map[string]types.FileState{
			"devenv.yaml": {Hash: "sha256:abc", Strategy: types.Overwrite},
		},
	}
	plan := buildUpdatePlan(files, modStatus, stored, t.TempDir(), UpdateOptions{})
	if len(plan.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(plan.Files))
	}
	if plan.Files[0].Action != UpdateActionSkip {
		t.Errorf("expected Skip, got %v", plan.Files[0].Action)
	}
	if plan.Files[0].Reason != "modified, use --force to overwrite" {
		t.Errorf("unexpected reason: %q", plan.Files[0].Reason)
	}
}

func TestBuildUpdatePlan_ModifiedWithForce(t *testing.T) {
	files := []types.GeneratedFile{
		{Path: "devenv.yaml", Content: []byte("new"), Mode: 0o644, Strategy: types.Overwrite},
	}
	modStatus := map[string]state.FileStatus{
		"devenv.yaml": {Path: "devenv.yaml", Status: types.Modified},
	}
	stored := types.GeneratedState{
		Files: map[string]types.FileState{
			"devenv.yaml": {Hash: "sha256:abc", Strategy: types.Overwrite},
		},
	}
	plan := buildUpdatePlan(files, modStatus, stored, t.TempDir(), UpdateOptions{Force: true})
	if len(plan.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(plan.Files))
	}
	if plan.Files[0].Action != UpdateActionRegenerate {
		t.Errorf("expected Regenerate with --force, got %v", plan.Files[0].Action)
	}
	if plan.Files[0].Reason != "modified, force overwrite" {
		t.Errorf("unexpected reason: %q", plan.Files[0].Reason)
	}
}

func TestBuildUpdatePlan_Deleted(t *testing.T) {
	files := []types.GeneratedFile{
		{Path: "devenv.yaml", Content: []byte("new"), Mode: 0o644, Strategy: types.Overwrite},
	}
	modStatus := map[string]state.FileStatus{
		"devenv.yaml": {Path: "devenv.yaml", Status: types.Deleted},
	}
	stored := types.GeneratedState{
		Files: map[string]types.FileState{
			"devenv.yaml": {Hash: "sha256:abc", Strategy: types.Overwrite},
		},
	}
	plan := buildUpdatePlan(files, modStatus, stored, t.TempDir(), UpdateOptions{})
	if len(plan.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(plan.Files))
	}
	if plan.Files[0].Action != UpdateActionSkip {
		t.Errorf("expected Skip for deleted file, got %v", plan.Files[0].Action)
	}
}

func TestBuildUpdatePlan_DeletedForce(t *testing.T) {
	files := []types.GeneratedFile{
		{Path: "devenv.yaml", Content: []byte("new"), Mode: 0o644, Strategy: types.Overwrite},
	}
	modStatus := map[string]state.FileStatus{
		"devenv.yaml": {Path: "devenv.yaml", Status: types.Deleted},
	}
	stored := types.GeneratedState{
		Files: map[string]types.FileState{
			"devenv.yaml": {Hash: "sha256:abc", Strategy: types.Overwrite},
		},
	}
	plan := buildUpdatePlan(files, modStatus, stored, t.TempDir(), UpdateOptions{Force: true})
	if len(plan.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(plan.Files))
	}
	if plan.Files[0].Action != UpdateActionCreate {
		t.Errorf("expected Create with --force for deleted file, got %v", plan.Files[0].Action)
	}
}

func TestBuildUpdatePlan_NewFile(t *testing.T) {
	files := []types.GeneratedFile{
		{Path: "new-file.txt", Content: []byte("content"), Mode: 0o644, Strategy: types.Overwrite},
	}
	modStatus := map[string]state.FileStatus{
		// new-file.txt is NOT in modStatus
	}
	stored := types.GeneratedState{
		Files: map[string]types.FileState{
			// new-file.txt is NOT in stored state
		},
	}
	plan := buildUpdatePlan(files, modStatus, stored, t.TempDir(), UpdateOptions{})
	if len(plan.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(plan.Files))
	}
	if plan.Files[0].Action != UpdateActionCreate {
		t.Errorf("expected Create for new file, got %v", plan.Files[0].Action)
	}
	if plan.Files[0].Status != types.New {
		t.Errorf("expected New status, got %v", plan.Files[0].Status)
	}
}

func TestBuildUpdatePlan_Unknown(t *testing.T) {
	files := []types.GeneratedFile{
		{Path: "devenv.yaml", Content: []byte("new"), Mode: 0o644, Strategy: types.Overwrite},
	}
	modStatus := map[string]state.FileStatus{
		"devenv.yaml": {Path: "devenv.yaml", Status: types.Unknown, Error: errors.New("permission denied")},
	}
	stored := types.GeneratedState{
		Files: map[string]types.FileState{
			"devenv.yaml": {Hash: "sha256:abc", Strategy: types.Overwrite},
		},
	}
	plan := buildUpdatePlan(files, modStatus, stored, t.TempDir(), UpdateOptions{})
	if len(plan.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(plan.Files))
	}
	if plan.Files[0].Action != UpdateActionSkip {
		t.Errorf("expected Skip for unknown status, got %v", plan.Files[0].Action)
	}
}

func TestPreviewUpdatePlan_Output(t *testing.T) {
	plan := UpdatePlan{
		Files: []FileUpdatePlan{
			{Path: "devenv.yaml", Status: types.Unmodified, Action: UpdateActionRegenerate, Reason: "unmodified, safe to update"},
			{Path: ".claude/settings.json", Status: types.Modified, Action: UpdateActionMerge, Reason: "modified, three-way merge"},
			{Path: "new-file.txt", Status: types.New, Action: UpdateActionCreate, Reason: "new file"},
		},
	}

	var buf bytes.Buffer
	previewUpdatePlan(plan, &buf)
	output := buf.String()

	// Verify column headers.
	expectedHeaders := []string{"File", "Status", "Action", "Reason"}
	for _, h := range expectedHeaders {
		if !bytes.Contains([]byte(output), []byte(h)) {
			t.Errorf("output missing header %q:\n%s", h, output)
		}
	}

	// Verify file names appear.
	expectedFiles := []string{"devenv.yaml", ".claude/settings.json", "new-file.txt"}
	for _, f := range expectedFiles {
		if !bytes.Contains([]byte(output), []byte(f)) {
			t.Errorf("output missing file %q:\n%s", f, output)
		}
	}

	// Verify action strings appear.
	expectedActions := []string{"regenerate", "merge", "create"}
	for _, a := range expectedActions {
		if !bytes.Contains([]byte(output), []byte(a)) {
			t.Errorf("output missing action %q:\n%s", a, output)
		}
	}
}

func TestBuildUpdatePlan_UnmodifiedSectionMarker(t *testing.T) {
	files := []types.GeneratedFile{
		{Path: "CLAUDE.md", Content: []byte("new generated"), Mode: 0o644, Strategy: types.SectionMarker},
	}
	modStatus := map[string]state.FileStatus{
		"CLAUDE.md": {Path: "CLAUDE.md", Status: types.Unmodified},
	}
	stored := types.GeneratedState{
		Files: map[string]types.FileState{
			"CLAUDE.md": {Hash: "sha256:abc", Strategy: types.SectionMarker},
		},
	}
	plan := buildUpdatePlan(files, modStatus, stored, t.TempDir(), UpdateOptions{})
	if len(plan.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(plan.Files))
	}
	if plan.Files[0].Action != UpdateActionMerge {
		t.Errorf("expected Merge for unmodified SectionMarker file, got %v", plan.Files[0].Action)
	}
	if plan.Files[0].Reason != "unmodified, section marker merge" {
		t.Errorf("unexpected reason: %q", plan.Files[0].Reason)
	}
}

func TestBuildUpdatePlan_UnmodifiedThreeWayMerge(t *testing.T) {
	files := []types.GeneratedFile{
		{Path: ".claude/settings.json", Content: []byte("new generated"), Mode: 0o644, Strategy: types.ThreeWayMerge},
	}
	modStatus := map[string]state.FileStatus{
		".claude/settings.json": {Path: ".claude/settings.json", Status: types.Unmodified},
	}
	stored := types.GeneratedState{
		Files: map[string]types.FileState{
			".claude/settings.json": {
				Hash:        "sha256:abc",
				Strategy:    types.ThreeWayMerge,
				BaseContent: []byte("previous base"),
			},
		},
	}
	plan := buildUpdatePlan(files, modStatus, stored, t.TempDir(), UpdateOptions{})
	if len(plan.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(plan.Files))
	}
	fp := plan.Files[0]
	if fp.Action != UpdateActionMerge {
		t.Errorf("expected Merge for unmodified ThreeWayMerge file, got %v", fp.Action)
	}
	if fp.Reason != "unmodified, three-way merge" {
		t.Errorf("unexpected reason: %q", fp.Reason)
	}
	if string(fp.OldContent) != "previous base" {
		t.Errorf("expected OldContent from stored BaseContent, got %q", string(fp.OldContent))
	}
}

func TestBuildUpdatePlan_UnmodifiedOverwrite_StillRegenerates(t *testing.T) {
	files := []types.GeneratedFile{
		{Path: "devenv.yaml", Content: []byte("new"), Mode: 0o644, Strategy: types.Overwrite},
		{Path: "changelog.md", Content: []byte("new"), Mode: 0o644, Strategy: types.LibraryManaged},
	}
	modStatus := map[string]state.FileStatus{
		"devenv.yaml":  {Path: "devenv.yaml", Status: types.Unmodified},
		"changelog.md": {Path: "changelog.md", Status: types.Unmodified},
	}
	stored := types.GeneratedState{
		Files: map[string]types.FileState{
			"devenv.yaml":  {Hash: "sha256:abc", Strategy: types.Overwrite},
			"changelog.md": {Hash: "sha256:def", Strategy: types.LibraryManaged},
		},
	}
	plan := buildUpdatePlan(files, modStatus, stored, t.TempDir(), UpdateOptions{})
	if len(plan.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(plan.Files))
	}
	for _, fp := range plan.Files {
		if fp.Action != UpdateActionRegenerate {
			t.Errorf("%s: expected Regenerate for unmodified non-SectionMarker file, got %v", fp.Path, fp.Action)
		}
	}
}

func TestUpdateActionString(t *testing.T) {
	tests := []struct {
		action   UpdateAction
		expected string
	}{
		{UpdateActionRegenerate, "regenerate"},
		{UpdateActionMerge, "merge"},
		{UpdateActionSkip, "skip"},
		{UpdateActionCreate, "create"},
		{UpdateActionSidecar, "sidecar"},
		{UpdateAction(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("action_%d", int(tt.action)), func(t *testing.T) {
			got := updateActionString(tt.action)
			if got != tt.expected {
				t.Errorf("updateActionString(%d) = %q, want %q", int(tt.action), got, tt.expected)
			}
		})
	}
}

// TestBuildUpdatePlan_SkipKeepsUserFile is the F503 regression for update: a
// skip-if-exists file (e.g. the PR template) the user wrote or edited is never
// replaced, not even with --force, while qsdev's own unmodified output and a
// missing file are still (re)generated.
func TestBuildUpdatePlan_SkipKeepsUserFile(t *testing.T) {
	t.Parallel()
	const path = ".github/pull_request_template.md"
	tests := []struct {
		name       string
		status     types.ModificationStatus // zero with untracked=true means no state entry
		untracked  bool
		onDisk     string // "" means the file is absent
		force      bool
		wantAction UpdateAction
	}{
		{name: "untracked user file", untracked: true, onDisk: "mine", wantAction: UpdateActionSkip},
		{name: "untracked user file with force", untracked: true, onDisk: "mine", force: true, wantAction: UpdateActionSkip},
		{name: "untracked missing file", untracked: true, wantAction: UpdateActionCreate},
		{name: "modified", status: types.Modified, onDisk: "edited", wantAction: UpdateActionSkip},
		{name: "modified with force", status: types.Modified, onDisk: "edited", force: true, wantAction: UpdateActionSkip},
		{name: "unmodified qsdev output", status: types.Unmodified, onDisk: "old", wantAction: UpdateActionRegenerate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if tt.onDisk != "" {
				if err := os.MkdirAll(filepath.Join(root, ".github"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, path), []byte(tt.onDisk), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			modStatus := map[string]state.FileStatus{}
			stored := types.GeneratedState{Files: map[string]types.FileState{}}
			if !tt.untracked {
				modStatus[path] = state.FileStatus{Path: path, Status: tt.status}
				stored.Files[path] = types.FileState{Hash: "sha256:abc", Strategy: types.Skip}
			}
			files := []types.GeneratedFile{{Path: path, Content: []byte("new"), Mode: 0o644, Strategy: types.Skip}}
			plan := buildUpdatePlan(files, modStatus, stored, root, UpdateOptions{Force: tt.force})
			if len(plan.Files) != 1 {
				t.Fatalf("expected 1 file, got %d", len(plan.Files))
			}
			if got := plan.Files[0].Action; got != tt.wantAction {
				t.Errorf("action = %v (%s), want %v", got, plan.Files[0].Reason, tt.wantAction)
			}
		})
	}
}
