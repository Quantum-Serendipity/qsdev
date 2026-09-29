package claudecode

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestPlanRegenWrites_SkipKeepsUserFile is the F503 regression for the Claude
// Code regeneration path: an enabled tool's skip-if-exists file (the PR
// template) that the user owns is kept, even with force, and is not recorded
// as qsdev output; qsdev's own unmodified output (recorded in the claude or
// any other project state) and a missing file are still written.
func TestPlanRegenWrites_SkipKeepsUserFile(t *testing.T) {
	t.Parallel()
	const (
		path    = ".github/pull_request_template.md"
		oldGen  = "generated v1\n"
		newGen  = "generated v2\n"
		userOwn = "MY OWN TEMPLATE\n"
	)
	tests := []struct {
		name        string
		onDisk      string // "" means absent
		claudeRec   string // content recorded in the claude state ("" = none)
		devinitRec  string // content recorded in the devinit state file ("" = none)
		force       bool
		wantWritten bool
	}{
		{name: "user file", onDisk: userOwn, wantWritten: false},
		{name: "user file with force", onDisk: userOwn, force: true, wantWritten: false},
		{name: "user edited claude-recorded file", onDisk: userOwn, claudeRec: oldGen, wantWritten: false},
		{name: "unmodified claude-recorded output", onDisk: oldGen, claudeRec: oldGen, wantWritten: true},
		{name: "unmodified devinit-recorded output", onDisk: oldGen, devinitRec: oldGen, wantWritten: true},
		{name: "missing file", wantWritten: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if tt.onDisk != "" {
				writeSkipTestFile(t, filepath.Join(root, path), tt.onDisk)
			}
			claudeState := types.GeneratedState{Files: map[string]types.FileState{}}
			if tt.claudeRec != "" {
				claudeState.Files[path] = types.FileState{Hash: state.ComputeHash([]byte(tt.claudeRec)), Strategy: types.Skip}
			}
			if tt.devinitRec != "" {
				rec := state.RecordFiles([]types.GeneratedFile{{Path: path, Content: []byte(tt.devinitRec), Strategy: types.Skip}})
				if err := state.SaveStateToFile(filepath.Join(root, state.InitStateFile()), rec); err != nil {
					t.Fatal(err)
				}
			}

			files := []types.GeneratedFile{{Path: path, Content: []byte(newGen), Mode: 0o644, Strategy: types.Skip}}
			var warn bytes.Buffer
			plan, err := planRegenWrites(files, claudeState, root, regenWriteOptions{force: tt.force, keepUserChanges: true}, &warn)
			if err != nil {
				t.Fatalf("planRegenWrites: %v", err)
			}
			if got := len(plan.writes) == 1; got != tt.wantWritten {
				t.Fatalf("written = %v, want %v (warnings: %s)", got, tt.wantWritten, warn.String())
			}
			if tt.wantWritten {
				return
			}
			if len(plan.recorded) != 0 {
				t.Error("a kept user file must not be recorded as qsdev output")
			}
			if plan.skipped != 1 || !strings.Contains(warn.String(), "kept your existing "+path) {
				t.Errorf("skipped = %d, warnings = %q; want the kept file reported", plan.skipped, warn.String())
			}
		})
	}
}

func writeSkipTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
