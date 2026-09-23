package posture

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestFileCategory_MachineOwned(t *testing.T) {
	machineOwned := []types.MergeStrategy{
		types.Overwrite,
		types.LibraryManaged,
		types.Skip,
		types.Append,
	}
	for _, s := range machineOwned {
		got := FileCategory(s)
		if got != "machine-owned" {
			t.Errorf("FileCategory(%s) = %q, want %q", s, got, "machine-owned")
		}
	}
}

func TestFileCategory_HumanEdited(t *testing.T) {
	humanEdited := []types.MergeStrategy{
		types.SectionMarker,
		types.ThreeWayMerge,
		types.ManualMerge,
		types.Merge,
	}
	for _, s := range humanEdited {
		got := FileCategory(s)
		if got != "human-edited" {
			t.Errorf("FileCategory(%s) = %q, want %q", s, got, "human-edited")
		}
	}
}

// TestFileCategory_AgreesWithDrift guards against config health and drift
// detection classifying the same modified file differently: a file healthy in
// config health must not raise a machine-owned drift warning, and vice versa.
func TestFileCategory_AgreesWithDrift(t *testing.T) {
	t.Parallel()
	for s := types.MergeStrategy(0); s.String() != "unknown"; s++ {
		t.Run(s.String(), func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "f"), []byte("edited\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			genState := types.GeneratedState{Files: map[string]types.FileState{
				"f": {Hash: "stale", Strategy: s},
			}}

			report := drift.Detect(dir, genState, nil)

			driftHumanEdited := false
			for _, cat := range report.Categories {
				for _, f := range cat.Findings {
					if f.Subject == "f" {
						driftHumanEdited = f.Severity == drift.Info
					}
				}
			}
			if healthHumanEdited := FileCategory(s) == "human-edited"; healthHumanEdited != driftHumanEdited {
				t.Errorf("strategy %s: config health human-edited=%v, drift human-edited=%v",
					s, healthHumanEdited, driftHumanEdited)
			}
		})
	}
}

func TestFileCategory_UnknownStrategy(t *testing.T) {
	got := FileCategory(types.MergeStrategy(99))
	if got != "machine-owned" {
		t.Errorf("FileCategory(99) = %q, want %q", got, "machine-owned")
	}
}

func TestComputeConfigScore_AllCurrent(t *testing.T) {
	files := []ConfigFileInfo{
		{Path: "a", State: "current", Category: "machine-owned"},
		{Path: "b", State: "current", Category: "human-edited"},
	}
	got := ComputeConfigScore(files)
	if got != 100.0 {
		t.Errorf("all current: got %f, want 100.0", got)
	}
}

func TestComputeConfigScore_Empty(t *testing.T) {
	got := ComputeConfigScore(nil)
	if got != 100.0 {
		t.Errorf("empty: got %f, want 100.0", got)
	}
}

func TestComputeConfigScore_AllMissing(t *testing.T) {
	files := []ConfigFileInfo{
		{Path: "a", State: "missing", Category: "machine-owned"},
		{Path: "b", State: "missing", Category: "machine-owned"},
	}
	got := ComputeConfigScore(files)
	if got != 0.0 {
		t.Errorf("all missing: got %f, want 0.0", got)
	}
}

func TestComputeConfigScore_AllCorrupt(t *testing.T) {
	files := []ConfigFileInfo{
		{Path: "a", State: "corrupt", Category: "machine-owned"},
	}
	got := ComputeConfigScore(files)
	if got != 0.0 {
		t.Errorf("all corrupt: got %f, want 0.0", got)
	}
}

func TestComputeConfigScore_ModifiedMachineOwned(t *testing.T) {
	files := []ConfigFileInfo{
		{Path: "a", State: "modified", Category: "machine-owned"},
	}
	got := ComputeConfigScore(files)
	if got != 50.0 {
		t.Errorf("modified machine-owned: got %f, want 50.0", got)
	}
}

func TestComputeConfigScore_ModifiedHumanEdited(t *testing.T) {
	files := []ConfigFileInfo{
		{Path: "a", State: "modified", Category: "human-edited"},
	}
	got := ComputeConfigScore(files)
	if got != 100.0 {
		t.Errorf("modified human-edited: got %f, want 100.0", got)
	}
}

func TestComputeConfigScore_Outdated(t *testing.T) {
	files := []ConfigFileInfo{
		{Path: "a", State: "outdated", Category: "machine-owned"},
	}
	got := ComputeConfigScore(files)
	if got != 50.0 {
		t.Errorf("outdated: got %f, want 50.0", got)
	}
}

func TestComputeConfigScore_MixedStates(t *testing.T) {
	files := []ConfigFileInfo{
		{Path: "a", State: "current", Category: "machine-owned"},  // 100
		{Path: "b", State: "modified", Category: "machine-owned"}, // 50  (machine-owned)
		{Path: "c", State: "modified", Category: "human-edited"},  // 100 (human-edited)
		{Path: "d", State: "outdated", Category: "machine-owned"}, // 50
		{Path: "e", State: "missing", Category: "machine-owned"},  // 0
		{Path: "f", State: "corrupt", Category: "machine-owned"},  // 0
	}
	// Total = (100 + 50 + 100 + 50 + 0 + 0) / 6 = 300 / 6 = 50
	got := ComputeConfigScore(files)
	if got != 50.0 {
		t.Errorf("mixed: got %f, want 50.0", got)
	}
}

func TestComputeConfigScore_ModifiedThreeWayMerge(t *testing.T) {
	files := []ConfigFileInfo{
		{Path: "a", State: "modified", Category: "human-edited"},
	}
	got := ComputeConfigScore(files)
	if got != 100.0 {
		t.Errorf("modified ThreeWayMerge: got %f, want 100.0", got)
	}
}

func TestComputeConfigScore_ModifiedManualMerge(t *testing.T) {
	files := []ConfigFileInfo{
		{Path: "a", State: "modified", Category: "human-edited"},
	}
	got := ComputeConfigScore(files)
	if got != 100.0 {
		t.Errorf("modified ManualMerge: got %f, want 100.0", got)
	}
}

func TestComputeConfigScore_ModifiedMerge(t *testing.T) {
	files := []ConfigFileInfo{
		{Path: "a", State: "modified", Category: "human-edited"},
	}
	got := ComputeConfigScore(files)
	if got != 100.0 {
		t.Errorf("modified Merge: got %f, want 100.0", got)
	}
}

func TestComputeConfigScore_ModifiedLibraryManaged(t *testing.T) {
	files := []ConfigFileInfo{
		{Path: "a", State: "modified", Category: "machine-owned"},
	}
	got := ComputeConfigScore(files)
	if got != 50.0 {
		t.Errorf("modified LibraryManaged: got %f, want 50.0", got)
	}
}

func TestComputeConfigScore_ModifiedSkip(t *testing.T) {
	files := []ConfigFileInfo{
		{Path: "a", State: "modified", Category: "machine-owned"},
	}
	got := ComputeConfigScore(files)
	if got != 50.0 {
		t.Errorf("modified Skip: got %f, want 50.0", got)
	}
}

func TestComputeConfigScore_ModifiedAppend(t *testing.T) {
	files := []ConfigFileInfo{
		{Path: "a", State: "modified", Category: "machine-owned"},
	}
	got := ComputeConfigScore(files)
	if got != 50.0 {
		t.Errorf("modified Append: got %f, want 50.0", got)
	}
}

func TestComputeConfigScore_TwoFiles(t *testing.T) {
	files := []ConfigFileInfo{
		{Path: "a", State: "current", Category: "machine-owned"},
		{Path: "b", State: "missing", Category: "machine-owned"},
	}
	got := ComputeConfigScore(files)
	// (100 + 0) / 2 = 50
	if math.Abs(got-50.0) > 0.001 {
		t.Errorf("two files: got %f, want 50.0", got)
	}
}
