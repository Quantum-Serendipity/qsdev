package toolreg

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

var gitWorkflowToolNames = []string{
	"pr-templates",
	"branch-naming",
	"commit-ticket",
	"pr-labels",
}

func TestGitWorkflowToolsRegistered(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range gitWorkflowToolNames {
		tool, ok := reg.ByName(name)
		if !ok {
			t.Errorf("git workflow tool %q not found in DefaultRegistry", name)
			continue
		}
		if tool.DisplayName == "" {
			t.Errorf("tool %q has empty DisplayName", name)
		}
		if tool.Description == "" {
			t.Errorf("tool %q has empty Description", name)
		}
		if tool.Category != CategoryDevEx {
			t.Errorf("tool %q has category %q, want %q", name, tool.Category, CategoryDevEx)
		}
		if len(tool.OwnedFiles) == 0 {
			t.Errorf("tool %q has no OwnedFiles", name)
		}
	}
}

func TestGitWorkflowToolDefaults(t *testing.T) {
	reg := DefaultRegistry()

	expectations := map[string]DefaultPolicy{
		"pr-templates":  AlwaysOn,
		"branch-naming": AlwaysOn,
		"commit-ticket": OptIn,
		"pr-labels":     AlwaysOn,
	}

	for name, wantPolicy := range expectations {
		tool, ok := reg.ByName(name)
		if !ok {
			t.Errorf("tool %q not found", name)
			continue
		}
		if tool.Default != wantPolicy {
			t.Errorf("tool %q has Default %v, want %v", name, tool.Default, wantPolicy)
		}
	}
}

func TestGitWorkflowToolsLifecycleOnly(t *testing.T) {
	assertLifecycleOnly(t, DefaultRegistry(), gitWorkflowToolNames...)
}

// renderInDevenvModule wraps a devenv.nix shared section in a minimal devenv
// module, the context lifecycle surgery inserts it into.
func renderInDevenvModule(t *testing.T, fn SharedContentFunc) string {
	t.Helper()
	content, err := fn(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("SharedContent function returned error: %v", err)
	}
	return "{ pkgs, lib, config, ... }:\n{\n" + string(content) + "\n}\n"
}

// hookScriptBody extracts the shell script body: the Nix indented string
// passed to pkgs.writeShellScript in a hook entry.
func hookScriptBody(t *testing.T, nix string) string {
	t.Helper()
	_, rest, ok := strings.Cut(nix, "writeShellScript")
	if !ok {
		t.Fatal("no writeShellScript in hook content")
	}
	_, rest, ok = strings.Cut(rest, "''\n")
	if !ok {
		t.Fatal("no indented-string script body in hook content")
	}
	body, _, ok := strings.Cut(rest, "''}")
	if !ok {
		t.Fatal("unterminated indented-string script body in hook content")
	}
	return body
}

// TestGitWorkflowNixHooksParse is the regression guard for hook sections
// that `enable` splices into devenv.nix: the Nix must parse and the embedded
// script must be valid shell, or the whole devenv shell stops evaluating.
func TestGitWorkflowNixHooksParse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fn   SharedContentFunc
	}{
		{"branch-naming", branchNamingNixContent},
		{"commit-ticket", commitTicketNixContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			module := renderInDevenvModule(t, tt.fn)

			// A backslash-escaped quote is not a Nix token outside a string;
			// inside the ${ } interpolation it breaks parsing.
			if strings.Contains(module, `\"`) {
				t.Errorf("hook content contains a backslash-escaped quote:\n%s", module)
			}

			if nixInstantiate, err := exec.LookPath("nix-instantiate"); err == nil {
				cmd := exec.Command(nixInstantiate, "--parse", "-")
				cmd.Stdin = strings.NewReader(module)
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Errorf("nix-instantiate --parse failed: %v\n%s\n--- module ---\n%s", err, out, module)
				}
			}

			if sh, err := exec.LookPath("sh"); err == nil {
				cmd := exec.Command(sh, "-n")
				cmd.Stdin = strings.NewReader(hookScriptBody(t, module))
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Errorf("hook script is not valid shell: %v\n%s", err, out)
				}
			}
		})
	}
}

// TestBranchNamingHookScript runs the rendered branch-naming script against
// branch names, proving the pattern reaches the shell intact.
func TestBranchNamingHookScript(t *testing.T) {
	t.Parallel()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not available")
	}
	module := renderInDevenvModule(t, branchNamingNixContent)
	// Substitute the git call so the script checks a fixed branch name.
	script := strings.Replace(hookScriptBody(t, module),
		"$(git rev-parse --abbrev-ref HEAD)", `"$1"`, 1)

	tests := []struct {
		branch string
		wantOK bool
	}{
		{"main", true},
		{"feat/add-login", true},
		{"fix/v1.2_patch", true},
		{"audit/deep-review", false},
		{"feature/Upper", false},
	}
	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			t.Parallel()
			cmd := exec.Command(sh, "-c", script, "branch-naming", tt.branch)
			out, err := cmd.CombinedOutput()
			if gotOK := err == nil; gotOK != tt.wantOK {
				t.Errorf("branch %q accepted=%v, want %v (output: %s)", tt.branch, gotOK, tt.wantOK, out)
			}
		})
	}
}

func TestGitWorkflowSharedContent_BranchNaming(t *testing.T) {
	reg := DefaultRegistry()

	tool, ok := reg.ByName("branch-naming")
	if !ok {
		t.Fatal("branch-naming not found in registry")
	}

	if tool.SharedContent == nil {
		t.Fatal("branch-naming should have SharedContent map")
	}

	fn, ok := tool.SharedContent[SharedSection{Path: DevenvNixFile, SectionID: "branch-naming"}]
	if !ok {
		t.Fatal("SharedContent missing 'branch-naming' key")
	}

	answers := types.WizardAnswers{}
	content, err := fn(answers)
	if err != nil {
		t.Fatalf("SharedContent function returned error: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("SharedContent function returned empty content")
	}

	s := string(content)
	if !strings.Contains(s, "branch-naming") {
		t.Error("branch-naming nix content should reference 'branch-naming'")
	}
	if !strings.Contains(s, "pre-push") {
		t.Error("branch-naming nix content should use pre-push stage")
	}
}

func TestGitWorkflowSharedContent_CommitTicket(t *testing.T) {
	reg := DefaultRegistry()

	tool, ok := reg.ByName("commit-ticket")
	if !ok {
		t.Fatal("commit-ticket not found in registry")
	}

	if tool.SharedContent == nil {
		t.Fatal("commit-ticket should have SharedContent map")
	}

	fn, ok := tool.SharedContent[SharedSection{Path: DevenvNixFile, SectionID: "commit-ticket"}]
	if !ok {
		t.Fatal("SharedContent missing 'commit-ticket' key")
	}

	answers := types.WizardAnswers{}
	content, err := fn(answers)
	if err != nil {
		t.Fatalf("SharedContent function returned error: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("SharedContent function returned empty content")
	}

	s := string(content)
	if !strings.Contains(s, "commit-ticket") {
		t.Error("commit-ticket nix content should reference 'commit-ticket'")
	}
	if !strings.Contains(s, "prepare-commit-msg") {
		t.Error("commit-ticket nix content should use prepare-commit-msg stage")
	}
	if !strings.Contains(s, "[A-Z]+-[0-9]+") {
		t.Error("commit-ticket nix content should contain ticket pattern")
	}
}

func TestGitWorkflowGenerateFunc_PRTemplates(t *testing.T) {
	reg := DefaultRegistry()

	tool, ok := reg.ByName("pr-templates")
	if !ok {
		t.Fatal("pr-templates not found in registry")
	}

	if tool.GenerateFunc == nil {
		t.Fatal("pr-templates should have GenerateFunc")
	}

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go"},
		},
	}
	files, err := tool.GenerateFunc(answers)
	if err != nil {
		t.Fatalf("GenerateFunc returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != ".github/pull_request_template.md" {
		t.Errorf("file path = %q, want .github/pull_request_template.md", files[0].Path)
	}
}

func TestGitWorkflowGenerateFunc_PRLabels(t *testing.T) {
	reg := DefaultRegistry()

	tool, ok := reg.ByName("pr-labels")
	if !ok {
		t.Fatal("pr-labels not found in registry")
	}

	if tool.GenerateFunc == nil {
		t.Fatal("pr-labels should have GenerateFunc")
	}

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go"},
			{Name: "python"},
		},
	}
	files, err := tool.GenerateFunc(answers)
	if err != nil {
		t.Fatalf("GenerateFunc returned error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0].Path != ".github/labeler.yml" {
		t.Errorf("files[0].Path = %q, want .github/labeler.yml", files[0].Path)
	}
	if files[1].Path != ".github/workflows/labeler.yml" {
		t.Errorf("files[1].Path = %q, want .github/workflows/labeler.yml", files[1].Path)
	}
}

func TestGitWorkflowBranchNamingNoGenerateFunc(t *testing.T) {
	reg := DefaultRegistry()

	tool, ok := reg.ByName("branch-naming")
	if !ok {
		t.Fatal("branch-naming not found in registry")
	}
	if tool.GenerateFunc != nil {
		t.Error("branch-naming should not have GenerateFunc (uses SharedContent only)")
	}
}

func TestGitWorkflowCommitTicketNoGenerateFunc(t *testing.T) {
	reg := DefaultRegistry()

	tool, ok := reg.ByName("commit-ticket")
	if !ok {
		t.Fatal("commit-ticket not found in registry")
	}
	if tool.GenerateFunc != nil {
		t.Error("commit-ticket should not have GenerateFunc (uses SharedContent only)")
	}
}
