package claudecode

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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
	return loadYAMLManifest[AgentManifest]("templates/agents/manifest.yaml")
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

		f, err := consultingAgentFile(a.Name)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}

	return files, nil
}

// consultingAgentFile builds the .claude/agents file for one consulting agent.
// It is the single builder for both init/update (deployAgents) and `enable`
// (the consulting-agent-* tool GenerateFunc), so both paths agree on the
// file's content, strategy and owner.
func consultingAgentFile(name string) (types.GeneratedFile, error) {
	content, err := templateFS.ReadFile("templates/agents/" + name + ".md")
	if err != nil {
		return types.GeneratedFile{}, fmt.Errorf("reading agent file %q: %w", name, err)
	}
	return types.GeneratedFile{
		Path:     ".claude/agents/" + name + ".md",
		Content:  content,
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.LibraryManaged,
		Owner:    "consulting-agent-" + name,
	}, nil
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
