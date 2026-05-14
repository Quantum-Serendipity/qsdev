package claudecode

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// GdevOpsManifest holds the list of gdev operation skills parsed from
// gdev-ops-manifest.yaml.
type GdevOpsManifest struct {
	Skills []GdevOpsEntry `yaml:"skills"`
}

// GdevOpsEntry describes a single gdev operation skill in the manifest.
type GdevOpsEntry struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
	UserOnly    bool     `yaml:"user_only"`
}

// loadGdevOpsManifest reads and parses the gdev-ops skill manifest from the
// embedded filesystem.
func loadGdevOpsManifest() (*GdevOpsManifest, error) {
	data, err := templateFS.ReadFile("templates/skills/gdev-ops-manifest.yaml")
	if err != nil {
		return nil, fmt.Errorf("reading gdev-ops manifest: %w", err)
	}

	var manifest GdevOpsManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing gdev-ops manifest: %w", err)
	}

	return &manifest, nil
}

// deployOperationSkills reads the gdev operation skill files from the embedded
// filesystem and returns GeneratedFile entries for each. When EnabledTools is
// non-nil, only skills whose name is present and true in EnabledTools are
// deployed. When EnabledTools is nil (legacy/first-run), all skills are
// deployed.
func deployOperationSkills(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
	manifest, err := loadGdevOpsManifest()
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
			return nil, fmt.Errorf("reading gdev-ops skill %q: %w", entry.Name, err)
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

// AvailableGdevOpsSkillNames returns the names of all gdev operation skills
// from the embedded manifest.
func AvailableGdevOpsSkillNames() []string {
	manifest, err := loadGdevOpsManifest()
	if err != nil {
		return nil
	}
	names := make([]string, len(manifest.Skills))
	for i, s := range manifest.Skills {
		names[i] = s.Name
	}
	return names
}
