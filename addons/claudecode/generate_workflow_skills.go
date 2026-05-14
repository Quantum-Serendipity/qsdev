package claudecode

import (
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// ConsultingSkillManifest holds the list of consulting workflow skills
// parsed from consulting-manifest.yaml.
type ConsultingSkillManifest struct {
	Skills []ConsultingSkillEntry `yaml:"skills"`
}

// ConsultingSkillEntry describes a single consulting workflow skill in the manifest.
type ConsultingSkillEntry struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
}

// loadConsultingSkillManifest reads and parses the consulting skill manifest
// from the embedded filesystem.
func loadConsultingSkillManifest() (*ConsultingSkillManifest, error) {
	data, err := templateFS.ReadFile("templates/skills/consulting-manifest.yaml")
	if err != nil {
		return nil, fmt.Errorf("reading consulting skill manifest: %w", err)
	}

	var manifest ConsultingSkillManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing consulting skill manifest: %w", err)
	}

	return &manifest, nil
}

// deployWorkflowSkills deploys consulting workflow skills based on the enabled
// tools in the wizard answers. Skills are only deployed when explicitly enabled
// via EnabledTools["consulting-workflow-{name}"].
func deployWorkflowSkills(answers types.WizardAnswers, _ *ecosystem.Registry) ([]types.GeneratedFile, error) {
	manifest, err := loadConsultingSkillManifest()
	if err != nil {
		return nil, err
	}

	// If EnabledTools is nil, no consulting workflow skills are deployed
	// (OptIn default — user must explicitly enable).
	if answers.EnabledTools == nil {
		return nil, nil
	}

	var files []types.GeneratedFile
	for _, skill := range manifest.Skills {
		toolKey := "consulting-workflow-" + skill.Name
		if !answers.EnabledTools[toolKey] {
			continue
		}

		content, err := templateFS.ReadFile("templates/skills/" + skill.Name + "/SKILL.md")
		if err != nil {
			return nil, fmt.Errorf("reading consulting skill file %q: %w", skill.Name, err)
		}

		files = append(files, types.GeneratedFile{
			Path:     ".claude/skills/" + skill.Name + "/SKILL.md",
			Content:  content,
			Mode:     0o644,
			Strategy: types.LibraryManaged,
			Owner:    "consulting-workflow-" + skill.Name,
		})
	}

	return files, nil
}

// AvailableConsultingSkillNames returns the names of all consulting workflow
// skills from the embedded manifest.
func AvailableConsultingSkillNames() []string {
	manifest, err := loadConsultingSkillManifest()
	if err != nil {
		return nil
	}
	names := make([]string, len(manifest.Skills))
	for i, s := range manifest.Skills {
		names[i] = s.Name
	}
	sort.Strings(names)
	return names
}
