package claudecode_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestAlwaysOnTools_StandardTier(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Tier:      "standard",
		Languages: []types.LanguageChoice{{Name: "go"}},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	var foundGitleaks, foundSemgrep bool
	for _, f := range files {
		if f.Path == ".gitleaks.toml" {
			foundGitleaks = true
		}
		if f.Path == ".semgrep.yml" {
			foundSemgrep = true
		}
	}
	if !foundGitleaks {
		t.Error("expected .gitleaks.toml at Standard tier")
	}
	if !foundSemgrep {
		t.Error("expected .semgrep.yml at Standard tier")
	}
}

func TestAlwaysOnTools_StandardTier_NoMCPOrAgents(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Tier:       "standard",
		Languages:  []types.LanguageChoice{{Name: "go"}},
		MCPServers: []string{"semble"},
		AgentTools: types.AgentToolsAnswers{PostmortemEnabled: true},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if f.Path == ".mcp.json" {
			t.Error("MCP config should not be generated at Standard tier")
		}
		if strings.Contains(f.Path, "agent-postmortem") {
			t.Error("postmortem skill should not be generated at Standard tier")
		}
	}
}

func TestAlwaysOnTools_FullTier_NoDuplication(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Tier:      "full",
		Languages: []types.LanguageChoice{{Name: "go"}},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	counts := map[string]int{}
	for _, f := range files {
		counts[f.Path]++
	}
	if counts[".gitleaks.toml"] != 1 {
		t.Errorf(".gitleaks.toml generated %d times, want 1", counts[".gitleaks.toml"])
	}
	if counts[".semgrep.yml"] != 1 {
		t.Errorf(".semgrep.yml generated %d times, want 1", counts[".semgrep.yml"])
	}
}

func TestAlwaysOnTools_SupplyChainTier_NoToolFiles(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Tier:      "supply-chain-only",
		Languages: []types.LanguageChoice{{Name: "go"}},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if f.Path == ".gitleaks.toml" || f.Path == ".semgrep.yml" {
			t.Errorf("AlwaysOn tool file %q should not be generated at supply-chain-only tier", f.Path)
		}
	}
}
