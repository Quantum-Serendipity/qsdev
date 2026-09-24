package claudecode

import (
	"fmt"
	"os/exec"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
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
		result.MCPServers = append(result.MCPServers, sembleServerName)

		if answers.AgentTools.SembleTextFiles {
			cat, err := catalog.Default()
			if err != nil {
				return nil, fmt.Errorf("loading catalog for semble config: %w", err)
			}
			def, ok := cat.MCPServer(sembleServerName)
			if !ok {
				return nil, fmt.Errorf("semble MCP server not found in catalog")
			}
			entry := catalogServerEntry(sembleServerName, def, installedMCPServers(answers.ProjectRoot))
			override := sembleTextFilesServer(entry)
			result.Override = &override
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
			Mode:     fileutil.ModeReadWrite,
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

// sembleServerName is the catalog name of the semble MCP server.
const sembleServerName = types.SembleMCPServer

// sembleTextFilesServer returns the .mcp.json definition written for semble
// when text-file indexing is enabled: the server's entry (installed binary or
// pinned launcher) plus --include-text-files.
func sembleTextFilesServer(entry MCPServerEntry) MCPServerConfig {
	return MCPServerConfig{
		Name:    sembleServerName,
		Command: entry.Command,
		Args:    append(append([]string{}, entry.Args...), "--include-text-files"),
		Env:     entry.Env,
	}
}
