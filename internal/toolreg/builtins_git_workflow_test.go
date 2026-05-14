package toolreg

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
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

func TestGitWorkflowToolEnableDisable(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range gitWorkflowToolNames {
		t.Run(name, func(t *testing.T) {
			tool, ok := reg.ByName(name)
			if !ok {
				t.Fatalf("tool %q not found", name)
			}

			if tool.EnableFunc == nil {
				t.Fatalf("tool %q has nil EnableFunc", name)
			}
			if tool.DisableFunc == nil {
				t.Fatalf("tool %q has nil DisableFunc", name)
			}

			// Test enable.
			answers := &types.WizardAnswers{
				EnabledTools: make(map[string]bool),
			}
			tool.EnableFunc(answers)
			if !answers.EnabledTools[name] {
				t.Errorf("after EnableFunc, EnabledTools[%q] should be true", name)
			}

			// Test disable.
			tool.DisableFunc(answers)
			if answers.EnabledTools[name] {
				t.Errorf("after DisableFunc, EnabledTools[%q] should be false", name)
			}
		})
	}
}

func TestGitWorkflowToolEnableFunc_NilMap(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range gitWorkflowToolNames {
		t.Run(name, func(t *testing.T) {
			tool, ok := reg.ByName(name)
			if !ok {
				t.Fatalf("tool %q not found", name)
			}

			answers := &types.WizardAnswers{}
			tool.EnableFunc(answers)
			if answers.EnabledTools == nil {
				t.Fatal("EnableFunc should initialize EnabledTools map when nil")
			}
			if !answers.EnabledTools[name] {
				t.Errorf("EnableFunc should set %q to true", name)
			}
		})
	}
}

func TestGitWorkflowToolDisableFunc_NilMap(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range gitWorkflowToolNames {
		t.Run(name, func(t *testing.T) {
			tool, ok := reg.ByName(name)
			if !ok {
				t.Fatalf("tool %q not found", name)
			}

			answers := &types.WizardAnswers{}
			tool.DisableFunc(answers)
			if answers.EnabledTools == nil {
				t.Fatal("DisableFunc should initialize EnabledTools map when nil")
			}
			if answers.EnabledTools[name] {
				t.Errorf("DisableFunc should set %q to false", name)
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

	fn, ok := tool.SharedContent["branch-naming"]
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

	fn, ok := tool.SharedContent["commit-ticket"]
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

