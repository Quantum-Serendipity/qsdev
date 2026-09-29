package claudecode

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func generateVersionSentinelFiles(answers types.WizardAnswers, registry *ecosystem.Registry) ([]types.GeneratedFile, error) {
	if !answers.AgentTools.VersionSentinel {
		return nil, nil
	}

	var files []types.GeneratedFile

	// Recovery workflow skill.
	skillContent, err := templateFS.ReadFile("templates/skills/version-sentinel-recovery.md")
	if err != nil {
		return nil, fmt.Errorf("reading version-sentinel skill: %w", err)
	}
	files = append(files, types.GeneratedFile{
		Path:     ".claude/skills/version-sentinel/SKILL.md",
		Content:  skillContent,
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.LibraryManaged,
	})

	// Ignore file for unsupported ecosystems.
	report := ecosystem.LanguageManifestCoverage(answers.Languages, registry)
	if report.HasUncovered() {
		var lines []string
		lines = append(lines, "# Ecosystems not covered by Version-Sentinel — verify versions manually")
		for _, m := range report.Uncovered {
			lines = append(lines, m.Path)
		}
		lines = append(lines, "")

		files = append(files, types.GeneratedFile{
			Path:     ".version-sentinel/ignore",
			Content:  []byte(strings.Join(lines, "\n")),
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.LibraryManaged,
		})
	}

	// .version-sentinel/events.jsonl is deliberately NOT generated: it is an
	// append-only history that vsentinel.LogVersionEvent creates on first
	// write (O_CREATE) and ReadVersionHistory treats as empty when absent.
	// Emitting it as a managed file would let every regeneration replace the
	// recorded history with the empty seed.

	return files, nil
}
