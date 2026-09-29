package devinit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestExecuteUpdatePlan_SkipKeepsUnrecordedUserFile verifies that update
// never replaces an existing, unrecorded file whose strategy is Skip (a
// project's own .npmrc), while still creating it when absent and still
// regenerating a Skip file qsdev itself recorded.
func TestExecuteUpdatePlan_SkipKeepsUnrecordedUserFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		existing string // "" = file absent
		status   *state.FileStatus
		want     string
	}{
		{name: "existing user file kept", existing: "@scope:registry=https://npm.example\n", want: "@scope:registry=https://npm.example\n"},
		{name: "absent file created", want: "ignore-scripts=true\n"},
		{name: "recorded unmodified file regenerated", existing: "old\n", status: &state.FileStatus{Path: ".npmrc", Status: types.Unmodified}, want: "ignore-scripts=true\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, ".npmrc")
			if tt.existing != "" {
				if err := os.WriteFile(path, []byte(tt.existing), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			modStatus := map[string]state.FileStatus{}
			if tt.status != nil {
				modStatus[".npmrc"] = *tt.status
			}
			files := []types.GeneratedFile{{Path: ".npmrc", Content: []byte("ignore-scripts=true\n"), Mode: 0o644, Strategy: types.Skip}}
			plan := buildUpdatePlan(files, modStatus, types.GeneratedState{}, dir, UpdateOptions{})

			if _, err := executeUpdatePlan(plan, dir, UpdateOptions{}); err != nil {
				t.Fatalf("executeUpdatePlan: %v", err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf(".npmrc = %q, want %q", got, tt.want)
			}
		})
	}
}
