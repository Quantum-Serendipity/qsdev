package claudecode

import (
	"fmt"
	"sort"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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
	return loadYAMLManifest[ConsultingSkillManifest]("templates/skills/consulting-manifest.yaml")
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

		f, err := consultingWorkflowFile(skill.Name)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}

	return files, nil
}

// consultingWorkflowFile builds the SKILL.md file for one consulting workflow.
// It is the single builder for both init/update (deployWorkflowSkills) and
// `enable` (the consulting-workflow-* tool GenerateFunc).
func consultingWorkflowFile(name string) (types.GeneratedFile, error) {
	content, err := templateFS.ReadFile("templates/skills/" + name + "/SKILL.md")
	if err != nil {
		return types.GeneratedFile{}, fmt.Errorf("reading consulting skill file %q: %w", name, err)
	}
	return types.GeneratedFile{
		Path:     ".claude/skills/" + name + "/SKILL.md",
		Content:  content,
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.LibraryManaged,
		Owner:    "consulting-workflow-" + name,
	}, nil
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
