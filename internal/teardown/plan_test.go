package teardown

import (
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/toolreg"
)

func TestBuildPlan_QuickProfile(t *testing.T) {
	classified := []ClassifiedFile{
		{Path: "file1.txt", Ownership: toolreg.Exclusive, Modified: false},
		{Path: "file2.md", Ownership: toolreg.Shared, Modified: false},
	}

	opts := TeardownOptions{Profile: ProfileQuick}
	plan := BuildPlan(classified, opts)

	if plan.Profile != ProfileQuick {
		t.Errorf("Profile = %q, want %q", plan.Profile, ProfileQuick)
	}
	if len(plan.Remove) != 0 {
		t.Errorf("Remove = %d items, want 0 (quick profile only removes dirs)", len(plan.Remove))
	}
	if len(plan.Clean) != 0 {
		t.Errorf("Clean = %d items, want 0 (quick profile only removes dirs)", len(plan.Clean))
	}
	if len(plan.Preserve) != 0 {
		t.Errorf("Preserve = %d items, want 0 (quick profile only removes dirs)", len(plan.Preserve))
	}
	if len(plan.Dirs) != 1 || plan.Dirs[0] != ".devinit" {
		t.Errorf("Dirs = %v, want [.devinit]", plan.Dirs)
	}
}

func TestBuildPlan_DefaultProfile_UnmodifiedExclusive(t *testing.T) {
	classified := []ClassifiedFile{
		{Path: "exclusive.txt", Ownership: toolreg.Exclusive, Modified: false},
	}

	opts := TeardownOptions{Profile: ProfileDefault}
	plan := BuildPlan(classified, opts)

	// Should be in Remove (not modified, exclusive).
	found := false
	for _, fa := range plan.Remove {
		if fa.Path == "exclusive.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected exclusive.txt in Remove list, but not found")
	}
}

func TestBuildPlan_DefaultProfile_ModifiedExclusive(t *testing.T) {
	classified := []ClassifiedFile{
		{Path: "modified.txt", Ownership: toolreg.Exclusive, Modified: true},
	}

	opts := TeardownOptions{Profile: ProfileDefault}
	plan := BuildPlan(classified, opts)

	// Should be in Preserve.
	if len(plan.Preserve) != 1 {
		t.Fatalf("Preserve = %d items, want 1", len(plan.Preserve))
	}
	if plan.Preserve[0].Path != "modified.txt" {
		t.Errorf("Preserve[0].Path = %q, want %q", plan.Preserve[0].Path, "modified.txt")
	}
	if !plan.Preserve[0].Modified {
		t.Errorf("Preserve[0].Modified = false, want true")
	}
}

func TestBuildPlan_DefaultProfile_SharedFile(t *testing.T) {
	classified := []ClassifiedFile{
		{Path: "CLAUDE.md", Ownership: toolreg.Shared, Modified: false},
	}

	opts := TeardownOptions{Profile: ProfileDefault}
	plan := BuildPlan(classified, opts)

	if len(plan.Clean) != 1 {
		t.Fatalf("Clean = %d items, want 1", len(plan.Clean))
	}
	if plan.Clean[0].Path != "CLAUDE.md" {
		t.Errorf("Clean[0].Path = %q, want %q", plan.Clean[0].Path, "CLAUDE.md")
	}
}

func TestBuildPlan_DefaultProfile_DeletedFile(t *testing.T) {
	classified := []ClassifiedFile{
		{Path: "gone.txt", Ownership: toolreg.Exclusive, Deleted: true},
	}

	opts := TeardownOptions{Profile: ProfileDefault}
	plan := BuildPlan(classified, opts)

	// Deleted files should be skipped.
	if len(plan.Remove) != len(stateFiles) {
		t.Errorf("Remove = %d items, want %d (only state files, deleted file skipped)",
			len(plan.Remove), len(stateFiles))
	}
	if len(plan.Preserve) != 0 {
		t.Errorf("Preserve = %d items, want 0 (deleted file skipped)", len(plan.Preserve))
	}
}

func TestBuildPlan_DefaultProfile_IncludesStateFiles(t *testing.T) {
	opts := TeardownOptions{Profile: ProfileDefault}
	plan := BuildPlan(nil, opts)

	// State files should always be in the remove list.
	statePathSet := make(map[string]bool)
	for _, fa := range plan.Remove {
		statePathSet[fa.Path] = true
	}
	for _, sf := range stateFiles {
		if !statePathSet[sf] {
			t.Errorf("expected state file %q in Remove list", sf)
		}
	}
}

func TestBuildPlan_ComplianceProfile(t *testing.T) {
	classified := []ClassifiedFile{
		{Path: "exclusive.txt", Ownership: toolreg.Exclusive, Modified: false},
		{Path: "CLAUDE.md", Ownership: toolreg.Shared, Modified: false},
	}

	opts := TeardownOptions{Profile: ProfileCompliance}
	plan := BuildPlan(classified, opts)

	if plan.Profile != ProfileCompliance {
		t.Errorf("Profile = %q, want %q", plan.Profile, ProfileCompliance)
	}

	// Should behave same as default for file classification.
	hasExclusive := false
	for _, fa := range plan.Remove {
		if fa.Path == "exclusive.txt" {
			hasExclusive = true
			break
		}
	}
	if !hasExclusive {
		t.Errorf("expected exclusive.txt in Remove list for compliance profile")
	}

	if len(plan.Clean) != 1 {
		t.Errorf("Clean = %d items, want 1", len(plan.Clean))
	}
}

func TestBuildPlan_DefaultProfile_DirsIncludeDevinit(t *testing.T) {
	opts := TeardownOptions{Profile: ProfileDefault}
	plan := BuildPlan(nil, opts)

	if len(plan.Dirs) != 1 || plan.Dirs[0] != ".devinit" {
		t.Errorf("Dirs = %v, want [.devinit]", plan.Dirs)
	}
}
