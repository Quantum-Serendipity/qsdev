package devinit

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/claudesettings"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestPlanExclusiveRemoval_KeepsClaudeSettings verifies disable never plans
// to delete the Claude Code settings file, which registers the self-protection
// hook (U18-01), however a tool claims it: a declared exclusive directory
// holding it, or an owner recorded in the agent-writable state file. The
// catalog rejects such a declaration; this is the guard at the deletion point,
// so a registry or state file that slipped past it still cannot reach the file.
func TestPlanExclusiveRemoval_KeepsClaudeSettings(t *testing.T) {
	t.Parallel()
	const name = "hostile"
	hookScript := ".claude/hooks/hostile.py"
	reg := toolreg.NewRegistry()
	if err := reg.Register(toolreg.Tool{
		Name:     name,
		Category: toolreg.CategoryDevEx,
		OwnedFiles: []toolreg.FileOwnership{
			{Path: ".claude", Ownership: toolreg.Exclusive},
			{Path: hookScript, Ownership: toolreg.Exclusive},
		},
	}); err != nil {
		t.Fatal(err)
	}
	tool, _ := reg.ByName(name)

	dir := t.TempDir()
	st := types.GeneratedState{Files: map[string]types.FileState{}}
	for _, rel := range []string{claudesettings.ProjectRelPath, hookScript} {
		content := []byte("generated " + rel + "\n")
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, content, 0o644); err != nil {
			t.Fatal(err)
		}
		st.Files[rel] = types.FileState{Hash: state.ComputeHash(content), Mode: 0o644, Owner: name}
	}

	for _, force := range []bool{false, true} {
		r, err := planExclusiveRemoval(reg, tool, name, dir, st, force)
		if err != nil {
			t.Fatalf("force=%v: planExclusiveRemoval: %v", force, err)
		}
		if slices.ContainsFunc(slices.Concat(r.files, r.dirs), claudesettings.HoldsProjectSettings) {
			t.Errorf("force=%v: plan reaches %s: files %v, dirs %v", force, claudesettings.ProjectRelPath, r.files, r.dirs)
		}
		if !slices.Contains(r.files, hookScript) {
			t.Errorf("force=%v: plan files %v lack the tool's own %s", force, r.files, hookScript)
		}
	}
}
