package claudecode

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// QsdevOpsManifest holds the list of qsdev operation skills parsed from
// qsdev-ops-manifest.yaml.
type QsdevOpsManifest struct {
	Skills []QsdevOpsEntry `yaml:"skills"`
}

// QsdevOpsEntry describes a single qsdev operation skill in the manifest.
type QsdevOpsEntry struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
	UserOnly    bool     `yaml:"user_only"`
}

// loadQsdevOpsManifest reads and parses the qsdev-ops skill manifest from the
// embedded filesystem.
func loadQsdevOpsManifest() (*QsdevOpsManifest, error) {
	return loadYAMLManifest[QsdevOpsManifest]("templates/skills/qsdev-ops-manifest.yaml")
}

// deployOperationSkills reads the qsdev operation skill files from the embedded
// filesystem and returns GeneratedFile entries for each. When EnabledTools is
// non-nil, only skills whose name is present and true in EnabledTools are
// deployed. When EnabledTools is nil (legacy/first-run), all skills are
// deployed.
func deployOperationSkills(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
	manifest, err := loadQsdevOpsManifest()
	if err != nil {
		return nil, err
	}

	var files []types.GeneratedFile
	for _, entry := range manifest.Skills {
		// Gate on EnabledTools if present.
		if answers.EnabledTools != nil {
			if !answers.EnabledTools[entry.Name] {
				continue
			}
		}

		content, err := templateFS.ReadFile("templates/skills/" + entry.Name + "/SKILL.md")
		if err != nil {
			return nil, fmt.Errorf("reading qsdev-ops skill %q: %w", entry.Name, err)
		}

		files = append(files, types.GeneratedFile{
			Path:     ".claude/skills/" + entry.Name + "/SKILL.md",
			Content:  content,
			Mode:     0o644,
			Strategy: types.LibraryManaged,
			Owner:    entry.Name,
		})
	}

	return files, nil
}

// AvailableQsdevOpsSkillNames returns the names of all qsdev operation skills
// from the embedded manifest.
func AvailableQsdevOpsSkillNames() []string {
	manifest, err := loadQsdevOpsManifest()
	if err != nil {
		return nil
	}
	names := make([]string, len(manifest.Skills))
	for i, s := range manifest.Skills {
		names[i] = s.Name
	}
	return names
}
