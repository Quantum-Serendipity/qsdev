package claudecode_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
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
// genuinely Full-only artifacts (consulting agents, operation skills, the qsdev
// reference doc) remain gated out. The always-on agent-postmortem skill is NOT
// a Full-only artifact: it is advertised in CLAUDE.md at Standard, so its
// SKILL.md must be present too (BL-P1-9).
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

	var sawMCP, sawPostmortem bool
	for _, f := range files {
		if f.Path == ".mcp.json" {
			sawMCP = true
		}
		if f.Path == ".claude/skills/agent-postmortem/SKILL.md" {
			sawPostmortem = true
		}
		// Full-only surfaces must remain gated out at Standard.
		if f.Path == ".claude/qsdev-reference.md" {
			t.Error("qsdev reference doc should not be generated at Standard tier")
		}
		if strings.HasPrefix(f.Path, ".claude/agents/") {
			t.Errorf("consulting agent %q should not be generated at Standard tier", f.Path)
		}
	}
	if !sawMCP {
		t.Error("MCP config should be generated at Standard tier when MCP servers are configured (DEFECT-5)")
	}
	if !sawPostmortem {
		t.Error("always-on agent-postmortem SKILL.md should be generated at Standard tier (BL-P1-9)")
	}
}

// TestAlwaysOnGenerateFuncs_NonNilBelowFull asserts that the catalog-derived
// always-on agent-tool generators (agent-postmortem, version-sentinel) produce
// files below the Full tier when their toggle is enabled. Regression guard for
// BL-P1-9: previously these GenerateFuncs self-gated on Full and returned
// (nil, nil) at Standard, so the enable path wrote only the CLAUDE.md section.
func TestAlwaysOnGenerateFuncs_NonNilBelowFull(t *testing.T) {
	reg := toolreg.DefaultRegistry()
	cases := []struct {
		tool    string
		answers types.WizardAnswers
	}{
		{
			tool: "agent-postmortem",
			answers: types.WizardAnswers{
				Tier:       "standard",
				AgentTools: types.AgentToolsAnswers{PostmortemEnabled: true},
			},
		},
		{
			tool: "version-sentinel",
			answers: types.WizardAnswers{
				Tier:       "standard",
				AgentTools: types.AgentToolsAnswers{VersionSentinel: true},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			tool, ok := reg.ByName(tc.tool)
			if !ok {
				t.Fatalf("tool %q not registered", tc.tool)
			}
			// "always-on" is derived from the catalog policy, not a hardcoded set.
			if tool.Default != toolreg.AlwaysOn {
				t.Fatalf("expected %q to be catalog default_policy always-on, got %v", tc.tool, tool.Default)
			}
			if tool.GenerateFunc == nil {
				t.Fatalf("tool %q has no GenerateFunc attached", tc.tool)
			}
			gotFiles, err := tool.GenerateFunc(tc.answers)
			if err != nil {
				t.Fatalf("GenerateFunc(%q) error: %v", tc.tool, err)
			}
			if len(gotFiles) == 0 {
				t.Errorf("always-on tool %q generated no files below Full; expected non-nil", tc.tool)
			}
		})
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
