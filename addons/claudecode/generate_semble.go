package claudecode

import (
	"fmt"
	"os/exec"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

type sembleResult struct {
	Files      []types.GeneratedFile
	MCPServers []string
	Override   *MCPServerConfig
}

func generateSembleConfig(answers types.WizardAnswers) (*sembleResult, error) {
	if !answers.AgentTools.SembleEnabled {
		return nil, nil
	}

	mode := answers.AgentTools.SembleMode
	if mode == "" {
		mode = "mcp"
	}

	result := &sembleResult{}

	if mode == "mcp" || mode == "both" {
		result.MCPServers = append(result.MCPServers, "semble")

		if answers.AgentTools.SembleTextFiles {
			entry := knownMCPServers["semble"]
			result.Override = &MCPServerConfig{
				Name:    "semble",
				Command: entry.Command,
				Args:    append(append([]string{}, entry.Args...), "--include-text-files"),
			}
		}
	}

	if mode == "subagent" || mode == "both" {
		content, err := templateFS.ReadFile("templates/agents/semble-search.md")
		if err != nil {
			return nil, fmt.Errorf("reading semble sub-agent: %w", err)
		}
		result.Files = append(result.Files, types.GeneratedFile{
			Path:     ".claude/agents/semble-search.md",
			Content:  content,
			Mode:     0o644,
			Strategy: types.LibraryManaged,
		})
	}

	return result, nil
}

func ValidateSemblePrerequisites(mode string) []string {
	var warnings []string

	if mode == "mcp" || mode == "both" || mode == "" {
		if _, err := exec.LookPath("uvx"); err != nil {
			warnings = append(warnings, "uvx not found on PATH (required for Semble MCP server)")
		}
	}

	return warnings
}
