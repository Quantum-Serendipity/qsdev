package versionsentinel

import (
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules" // every ecosystem module
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestManifestCoverage_AgreesWithClaudeMdSection pins one meaning of
// "covered" across the Version-Sentinel surfaces: for every ecosystem, the
// generated CLAUDE.md section lists as covered, lockfile-presence-only and
// not covered exactly the manifests manifest_coverage reports as diffed,
// presence_only and uncovered (go.mod was once "NOT covered" in one and
// "diffed" in the other).
func TestManifestCoverage_AgreesWithClaudeMdSection(t *testing.T) {
	tool, ok := toolreg.DefaultRegistry().ByName(toolreg.ToolVersionSentinel)
	if !ok || tool.SectionDataFunc == nil {
		t.Fatalf("catalog tool %q has no CLAUDE.md section data", toolreg.ToolVersionSentinel)
	}
	modules := ecosystem.DefaultRegistry().All()
	if len(modules) < 10 {
		t.Fatalf("only %d ecosystem modules registered", len(modules))
	}
	for _, mod := range modules {
		t.Run(mod.Name(), func(t *testing.T) {
			languages := []types.LanguageChoice{{Name: mod.Name()}}
			report := manifestCoverageFor(t, languages)
			section := tool.SectionDataFunc(types.WizardAnswers{Languages: languages}, ecosystem.DefaultRegistry())
			for _, tc := range []struct {
				key    string
				report []manifestCoverageEntry
			}{
				{"Covered", report.Diffed},
				{"PresenceOnly", report.PresenceOnly},
				{"Uncovered", report.Uncovered},
			} {
				got, _ := section[tc.key].([]string)
				if want := manifestPaths(tc.report); !slices.Equal(got, want) && len(got)+len(want) > 0 {
					t.Errorf("CLAUDE.md %s = %q, manifest_coverage reports %q", tc.key, got, want)
				}
			}
		})
	}
}
