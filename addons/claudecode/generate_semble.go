package claudecode

import (
	"fmt"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func generateSembleFiles(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
	if !answers.AgentTools.SembleEnabled {
		return nil, nil
	}

	mode := answers.AgentTools.SembleMode
	if mode == "" {
		mode = "mcp"
	}

	var files []types.GeneratedFile

	if mode == "subagent" || mode == "both" {
		content, err := templateFS.ReadFile("templates/agents/semble-search.md")
		if err != nil {
			return nil, fmt.Errorf("reading semble sub-agent: %w", err)
		}
		files = append(files, types.GeneratedFile{
			Path:     ".claude/agents/semble-search.md",
			Content:  content,
			Mode:     0o644,
			Strategy: types.LibraryManaged,
		})
	}

	return files, nil
}
