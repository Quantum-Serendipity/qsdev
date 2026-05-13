package devinit

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/state"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
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
	plan := buildUpdatePlan(files, modStatus, stored, UpdateOptions{})
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
	plan := buildUpdatePlan(files, modStatus, stored, UpdateOptions{})
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
	plan := buildUpdatePlan(files, modStatus, stored, UpdateOptions{})
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
	plan := buildUpdatePlan(files, modStatus, stored, UpdateOptions{})
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
	plan := buildUpdatePlan(files, modStatus, stored, UpdateOptions{})
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
	plan := buildUpdatePlan(files, modStatus, stored, UpdateOptions{})
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
	plan := buildUpdatePlan(files, modStatus, stored, UpdateOptions{Force: true})
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
	plan := buildUpdatePlan(files, modStatus, stored, UpdateOptions{})
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
	plan := buildUpdatePlan(files, modStatus, stored, UpdateOptions{Force: true})
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
	plan := buildUpdatePlan(files, modStatus, stored, UpdateOptions{})
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
	plan := buildUpdatePlan(files, modStatus, stored, UpdateOptions{})
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
