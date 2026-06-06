package claudecode

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
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
		Mode:     0o644,
		Strategy: types.LibraryManaged,
	})

	// Ignore file for unsupported ecosystems.
	report := collectManifestCoverage(answers, registry)
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
			Mode:     0o644,
			Strategy: types.LibraryManaged,
		})
	}

	// Seed events.jsonl for MCP server history tool.
	files = append(files, types.GeneratedFile{
		Path:     ".version-sentinel/events.jsonl",
		Content:  []byte{},
		Mode:     0o644,
		Strategy: types.LibraryManaged,
	})

	return files, nil
}

func collectManifestCoverage(answers types.WizardAnswers, registry *ecosystem.Registry) ecosystem.ManifestCoverageReport {
	modules, configFor := resolveLanguageModules(answers, registry)
	return ecosystem.AggregateManifestCoverage(modules, configFor)
}
