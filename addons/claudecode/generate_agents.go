package claudecode

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// AgentManifest holds the list of available consulting workflow agents
// parsed from manifest.yaml.
type AgentManifest struct {
	Agents []AgentEntry `yaml:"agents"`
}

// AgentEntry describes a single agent in the manifest.
type AgentEntry struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
	ReadOnly    bool     `yaml:"read_only"`
}

// loadAgentManifest reads and parses the agent manifest from the embedded filesystem.
func loadAgentManifest() (*AgentManifest, error) {
	data, err := templateFS.ReadFile("templates/agents/manifest.yaml")
	if err != nil {
		return nil, fmt.Errorf("reading agent manifest: %w", err)
	}

	var manifest AgentManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing agent manifest: %w", err)
	}

	return &manifest, nil
}

// deployAgents reads the selected agent files from the embedded filesystem and
// returns GeneratedFile entries for each. Agents are opt-in: they are only
// deployed when explicitly enabled in WizardAnswers.EnabledTools.
func deployAgents(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
	manifest, err := loadAgentManifest()
	if err != nil {
		return nil, err
	}

	var files []types.GeneratedFile
	for _, a := range manifest.Agents {
		toolKey := "consulting-agent-" + a.Name

		// Agents are OptIn — if EnabledTools is nil (legacy), skip all agents.
		if answers.EnabledTools == nil {
			continue
		}
		if !answers.EnabledTools[toolKey] {
			continue
		}

		content, err := templateFS.ReadFile("templates/agents/" + a.Name + ".md")
		if err != nil {
			return nil, fmt.Errorf("reading agent file %q: %w", a.Name, err)
		}

		files = append(files, types.GeneratedFile{
			Path:     ".claude/agents/" + a.Name + ".md",
			Content:  content,
			Mode:     0o644,
			Strategy: types.LibraryManaged,
			Owner:    toolKey,
		})
	}

	return files, nil
}

// AvailableAgentNames returns the names of all agents from the embedded manifest.
func AvailableAgentNames() []string {
	manifest, err := loadAgentManifest()
	if err != nil {
		return nil
	}
	names := make([]string, len(manifest.Agents))
	for i, a := range manifest.Agents {
		names[i] = a.Name
	}
	return names
}
