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

// TestStandardTier_MCPGeneratedNoFullOnlyArtifacts verifies DEFECT-5: at the
// standard tier, configured MCP servers DO materialize .mcp.json, while
// Full-only artifacts (consulting agents, operation skills such as the
// postmortem agent) remain gated out.
func TestStandardTier_MCPGeneratedNoFullOnlyArtifacts(t *testing.T) {
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

	var sawMCP bool
	for _, f := range files {
		if f.Path == ".mcp.json" {
			sawMCP = true
		}
		if strings.Contains(f.Path, "agent-postmortem") {
			t.Error("postmortem skill should not be generated at Standard tier")
		}
	}
	if !sawMCP {
		t.Error("MCP config should be generated at Standard tier when MCP servers are configured (DEFECT-5)")
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

// TestSuppressedConfigWarnings covers DEFECT-5/7 messaging: configured skills or
// MCP servers only warn when the resolved tier is below Standard (where they are
// suppressed); at Standard+ they are emitted, so there is nothing to warn about.
func TestSuppressedConfigWarnings(t *testing.T) {
	t.Run("below standard warns", func(t *testing.T) {
		w := claudecode.SuppressedConfigWarnings(types.WizardAnswers{
			Tier:       "supply-chain-only",
			Skills:     []string{"deploy"},
			MCPServers: []string{"github"},
		})
		if len(w) != 2 {
			t.Fatalf("expected 2 warnings below standard, got %d: %v", len(w), w)
		}
	})
	t.Run("standard does not warn", func(t *testing.T) {
		w := claudecode.SuppressedConfigWarnings(types.WizardAnswers{
			Tier:       "standard",
			Skills:     []string{"deploy"},
			MCPServers: []string{"github"},
		})
		if len(w) != 0 {
			t.Errorf("expected no warnings at standard, got: %v", w)
		}
	})
}
