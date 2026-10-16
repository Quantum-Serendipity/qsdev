package claudecode_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestSemble_Disabled(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Languages:  []types.LanguageChoice{{Name: "go"}},
		AgentTools: types.AgentToolsAnswers{SembleEnabled: false},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if strings.Contains(f.Path, "semble") {
			t.Errorf("semble file %q should not be generated when disabled", f.Path)
		}
	}
}

func TestSemble_MCPMode(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Tier:       "full",
		Languages:  []types.LanguageChoice{{Name: "go"}},
		MCPServers: []string{"semble"},
		AgentTools: types.AgentToolsAnswers{
			SembleEnabled: true,
			SembleMode:    "mcp",
		},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	var hasMCP bool
	for _, f := range files {
		if f.Path == ".mcp.json" {
			hasMCP = true
			var mcp struct {
				MCPServers map[string]struct {
					Command string   `json:"command"`
					Args    []string `json:"args"`
				} `json:"mcpServers"`
			}
			if err := json.Unmarshal(f.Content, &mcp); err != nil {
				t.Fatalf("parsing .mcp.json: %v", err)
			}
			entry, ok := mcp.MCPServers["semble"]
			if !ok {
				t.Error("expected semble entry in .mcp.json")
			} else {
				if entry.Command != "uvx" {
					t.Errorf("semble command = %q, want uvx", entry.Command)
				}
			}
		}
		if f.Path == ".claude/agents/semble-search.md" {
			t.Error("MCP-only mode should NOT generate sub-agent file")
		}
	}
	if !hasMCP {
		t.Error("expected .mcp.json to be generated")
	}
}

func TestSemble_SubAgentMode(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Tier:      "full",
		Languages: []types.LanguageChoice{{Name: "go"}},
		AgentTools: types.AgentToolsAnswers{
			SembleEnabled: true,
			SembleMode:    "subagent",
		},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	var hasSubAgent bool
	for _, f := range files {
		if f.Path == ".claude/agents/semble-search.md" {
			hasSubAgent = true
			if len(f.Content) == 0 {
				t.Error("sub-agent file should not be empty")
			}
		}
	}
	if !hasSubAgent {
		t.Error("expected sub-agent file in subagent mode")
	}
}

func TestSemble_BothMode(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Tier:       "full",
		Languages:  []types.LanguageChoice{{Name: "go"}},
		MCPServers: []string{"semble"},
		AgentTools: types.AgentToolsAnswers{
			SembleEnabled: true,
			SembleMode:    "both",
		},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	var hasMCP, hasSubAgent bool
	for _, f := range files {
		if f.Path == ".mcp.json" {
			hasMCP = true
		}
		if f.Path == ".claude/agents/semble-search.md" {
			hasSubAgent = true
		}
	}
	if !hasMCP {
		t.Error("both mode should generate .mcp.json")
	}
	if !hasSubAgent {
		t.Error("both mode should generate sub-agent")
	}
}

func TestSemble_TextFilesFlag(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Tier:       "full",
		Languages:  []types.LanguageChoice{{Name: "go"}},
		MCPServers: []string{"semble"},
		AgentTools: types.AgentToolsAnswers{
			SembleEnabled:   true,
			SembleMode:      "mcp",
			SembleTextFiles: true,
		},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if f.Path == ".mcp.json" {
			content := string(f.Content)
			if !strings.Contains(content, "--include-text-files") {
				t.Errorf(".mcp.json should contain --include-text-files flag:\n%s", content)
			}
			return
		}
	}
	t.Error("expected .mcp.json")
}

// TestSemble_EmptyModeUsesCatalogDefault pins that an empty semble mode
// generates the catalog default mode's file set, the semble-search sub-agent
// included, rather than a default kept in the generator.
func TestSemble_EmptyModeUsesCatalogDefault(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	paths := func(mode string) []string {
		t.Helper()
		answers := types.WizardAnswers{
			Tier:       "full",
			Languages:  []types.LanguageChoice{{Name: "go"}},
			MCPServers: []string{"semble"},
			AgentTools: types.AgentToolsAnswers{SembleEnabled: true, SembleMode: mode},
		}
		files, err := claudecode.NewClaudeCodeGenerator(newTestRegistry(t, goMock()), claudecode.Config{}).Generate(answers)
		if err != nil {
			t.Fatal(err)
		}
		out := make([]string, 0, len(files))
		for _, f := range files {
			out = append(out, f.Path)
		}
		slices.Sort(out)
		return out
	}
	got, want := paths(""), paths(cat.DefaultSembleMode())
	if !slices.Equal(got, want) {
		t.Errorf("empty mode files = %v, want the catalog default %q's %v", got, cat.DefaultSembleMode(), want)
	}
	if !slices.Contains(got, ".claude/agents/semble-search.md") {
		t.Errorf("empty mode (catalog default %q) generated no semble-search sub-agent: %v", cat.DefaultSembleMode(), got)
	}
}
