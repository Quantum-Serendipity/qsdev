package claudecode

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// writeRegenTestFile writes content to path, creating its parent directories.
func writeRegenTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestApplyRegenPlanLegacyRemovalConfined is the claude regen half of
// U14-07: a legacy flat skill file whose directory is a committed symlink
// into .git (.claude/skills -> ../.git/skills) is not deleted, and the
// refusal is reported as a warning naming the redirect.
func TestApplyRegenPlanLegacyRemovalConfined(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	gitFile := filepath.Join(root, ".git", "skills", "old.md")
	writeRegenTestFile(t, gitFile, "keep\n")
	testutil.SymlinkOrSkip(t, filepath.Join("..", ".git", "skills"), filepath.Join(root, ".claude", "skills"))

	plan := &regenPlan{legacyRemovals: []string{".claude/skills/old.md"}}
	var warn bytes.Buffer
	if err := applyRegenPlan(plan, root, &warn); err != nil {
		t.Fatalf("applyRegenPlan: %v", err)
	}
	if _, err := os.Stat(gitFile); err != nil {
		t.Errorf(".git/skills/old.md was removed: %v", err)
	}
	for _, want := range []string{"could not remove stale skill file .claude/skills/old.md", "inside .git"} {
		if !strings.Contains(warn.String(), want) {
			t.Errorf("warnings = %q, want them to contain %q", warn.String(), want)
		}
	}
}

// TestApplyRegenPlanLegacyRemovalThroughAlias verifies a legacy removal
// through an in-root directory alias, absolute or stepping out of the root
// and back in, still removes the file behind it, as before the containment
// rule was shared.
func TestApplyRegenPlanLegacyRemovalThroughAlias(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		target func(root string) string
	}{
		{"absolute", func(root string) string { return filepath.Join(root, "inside") }},
		{"relative", func(root string) string { return filepath.Join("..", "..", filepath.Base(root), "inside") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			behind := filepath.Join(root, "inside", "old.md")
			writeRegenTestFile(t, behind, "stale\n")
			testutil.SymlinkOrSkip(t, tt.target(root), filepath.Join(root, ".claude", "skills"))

			plan := &regenPlan{legacyRemovals: []string{".claude/skills/old.md"}}
			var warn bytes.Buffer
			if err := applyRegenPlan(plan, root, &warn); err != nil {
				t.Fatalf("applyRegenPlan: %v", err)
			}
			if warn.Len() != 0 {
				t.Errorf("unexpected warnings: %q", warn.String())
			}
			if _, err := os.Lstat(behind); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("%s should be removed (lstat err=%v)", behind, err)
			}
		})
	}
}

// TestApplyRegenPlanValidatesBeforeLegacyRemoval is the U14-12 interim: a
// plan whose writes fail validation changes nothing, not even its legacy
// removals.
func TestApplyRegenPlanValidatesBeforeLegacyRemoval(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	legacy := filepath.Join(root, ".claude", "skills", "old.md")
	writeRegenTestFile(t, legacy, "stale\n")

	plan := &regenPlan{
		legacyRemovals: []string{".claude/skills/old.md"},
		writes:         []types.GeneratedFile{{Path: ".claude/settings.json", Content: []byte("{not json"), Mode: 0o644}},
	}
	err := applyRegenPlan(plan, root, io.Discard)
	if !errors.Is(err, generate.ErrInvalidContent) {
		t.Fatalf("applyRegenPlan error = %v, want ErrInvalidContent", err)
	}
	if _, err := os.Stat(legacy); err != nil {
		t.Errorf("legacy file removed before validation failed: %v", err)
	}
}

// TestSaveAnswersLegacyCopyConfined verifies saveAnswers' cleanup of the
// legacy per-addon answers copy does not delete through a symlinked
// directory into .git.
func TestSaveAnswersLegacyCopyConfined(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	legacy := filepath.FromSlash(answers.LegacyClaudeCopyFile())
	gitFile := filepath.Join(root, ".git", filepath.Base(legacy))
	writeRegenTestFile(t, gitFile, "keep\n")
	testutil.SymlinkOrSkip(t, ".git", filepath.Join(root, filepath.Dir(legacy)))

	if err := saveAnswers(root, types.WizardAnswers{}); err != nil {
		t.Fatalf("saveAnswers: %v", err)
	}
	if _, err := os.Stat(gitFile); err != nil {
		t.Errorf("%s was removed through the symlinked directory: %v", gitFile, err)
	}
}
