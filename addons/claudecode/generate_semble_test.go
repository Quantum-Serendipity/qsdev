package claudecode_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/claudecode"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
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
