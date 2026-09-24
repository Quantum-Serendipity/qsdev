package claudecode_test

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	claudecode "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// runClaude executes `qsdev claude <args>` in the current directory and fails
// the test on error.
func runClaude(t *testing.T, args ...string) string {
	t.Helper()
	cmd := claudecode.ExportClaudeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("claude %v failed: %v\n%s", args, err, buf.String())
	}
	return buf.String()
}

// simulateEnable rewrites only the primary answers file, exactly as
// `qsdev enable` and `qsdev init --update` do.
func simulateEnable(t *testing.T, root string) {
	t.Helper()
	primary, err := answers.LoadPrimary(root)
	if err != nil {
		t.Fatalf("loading primary answers: %v", err)
	}
	primary.Tier = "full"
	primary.Languages = []types.LanguageChoice{{Name: "go"}}
	if primary.EnabledTools == nil {
		primary.EnabledTools = map[string]bool{}
	}
	primary.EnabledTools["pr-templates"] = true
	if err := answers.SavePrimary(root, primary); err != nil {
		t.Fatalf("saving primary answers: %v", err)
	}
}

// assertEnableSurvived checks that the state written by simulateEnable is still
// present in the primary answers file.
func assertEnableSurvived(t *testing.T, root string) types.WizardAnswers {
	t.Helper()
	primary, err := answers.LoadPrimary(root)
	if err != nil {
		t.Fatalf("loading primary answers: %v", err)
	}
	if !primary.EnabledTools["pr-templates"] {
		t.Errorf("EnabledTools[pr-templates] was reverted in the primary answers: %v", primary.EnabledTools)
	}
	if len(primary.Languages) != 1 || primary.Languages[0].Name != "go" {
		t.Errorf("Languages were reverted in the primary answers: %+v", primary.Languages)
	}
	return primary
}

// TestAddSkill_KeepsPrimaryAnswersWrittenAfterInit pins F515: claude
// subcommands must read the primary answers file, not a stale per-addon copy,
// so they never write an old snapshot over `qsdev enable` changes.
func TestAddSkill_KeepsPrimaryAnswersWrittenAfterInit(t *testing.T) {
	root := t.TempDir()
	chdir(t, root)

	runClaude(t, "init", "--yes", "--permission-preset", "standard")
	simulateEnable(t, root)
	runClaude(t, "add-skill", "deploy")

	primary := assertEnableSurvived(t, root)
	if !slices.Contains(primary.Skills, "deploy") {
		t.Errorf("Skills = %v, want deploy added", primary.Skills)
	}
}

// TestClaudeInitForce_KeepsNonClaudeAnswers pins F515: re-running
// `claude init --force` must overlay only the Claude Code settings it owns
// instead of replacing the whole primary answers file.
func TestClaudeInitForce_KeepsNonClaudeAnswers(t *testing.T) {
	root := t.TempDir()
	chdir(t, root)

	runClaude(t, "init", "--yes", "--permission-preset", "standard")
	simulateEnable(t, root)
	runClaude(t, "init", "--yes", "--force", "--permission-preset", "minimal")

	primary := assertEnableSurvived(t, root)
	if primary.PermissionLevel != "minimal" {
		t.Errorf("PermissionLevel = %q, want the flag value %q", primary.PermissionLevel, "minimal")
	}
	if primary.Tier != "full" {
		t.Errorf("Tier = %q, want the saved value %q", primary.Tier, "full")
	}
}

// TestLoadAnswers_IgnoresLegacyCopy pins F515: the legacy
// .claude/.qsdev-claude-answers.yaml copy is never read, and saving removes it
// so teardown cannot leave a copy behind that resurrects old configuration.
func TestLoadAnswers_IgnoresLegacyCopy(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, ".claude", ".qsdev-claude-answers.yaml")
	if err := answers.SaveToDir(root, ".claude", ".qsdev-claude-answers.yaml", types.WizardAnswers{ProjectName: "stale"}); err != nil {
		t.Fatal(err)
	}

	if _, err := claudecode.ExportLoadAnswers(root); err == nil {
		t.Fatal("loadAnswers succeeded from the legacy copy alone; want a 'run init first' error")
	} else if !strings.Contains(err.Error(), "init' first") {
		t.Errorf("error = %q, want a hint to run init first", err)
	}

	if err := answers.SavePrimary(root, types.WizardAnswers{ProjectName: "fresh"}); err != nil {
		t.Fatal(err)
	}
	got, err := claudecode.ExportLoadAnswers(root)
	if err != nil {
		t.Fatalf("loadAnswers: %v", err)
	}
	if got.ProjectName != "fresh" {
		t.Errorf("ProjectName = %q, want the primary value %q", got.ProjectName, "fresh")
	}

	if err := claudecode.ExportSaveAnswers(root, got); err != nil {
		t.Fatalf("saveAnswers: %v", err)
	}
	if _, err := os.Stat(legacy); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("legacy answers copy still present after save (stat err = %v)", err)
	}
}
