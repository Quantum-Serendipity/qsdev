package claudecode_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestVersionSentinel_Disabled(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Languages:  []types.LanguageChoice{{Name: "go"}},
		AgentTools: types.AgentToolsAnswers{VersionSentinel: false},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if strings.Contains(f.Path, "version-sentinel") {
			t.Errorf("version-sentinel file %q should not be generated when disabled", f.Path)
		}
	}
}

func TestVersionSentinelIgnore_GoProject(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Tier:       "full",
		Languages:  []types.LanguageChoice{{Name: "go"}},
		AgentTools: types.AgentToolsAnswers{VersionSentinel: true},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	var ignoreFile *types.GeneratedFile
	for i, f := range files {
		if f.Path == ".version-sentinel/ignore" {
			ignoreFile = &files[i]
			break
		}
	}

	if ignoreFile == nil {
		t.Fatal("expected .version-sentinel/ignore file for Go project (VS doesn't cover go.mod)")
		return
	}

	content := string(ignoreFile.Content)
	if !strings.Contains(content, "go.mod") {
		t.Errorf("ignore file should contain go.mod, got:\n%s", content)
	}
}

func TestVersionSentinelIgnore_NpmProject(t *testing.T) {
	reg := newTestRegistry(t, jsMock())
	answers := types.WizardAnswers{
		Tier:       "full",
		Languages:  []types.LanguageChoice{{Name: "javascript"}},
		AgentTools: types.AgentToolsAnswers{VersionSentinel: true},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if f.Path == ".version-sentinel/ignore" {
			t.Error("npm project should NOT have .version-sentinel/ignore (npm is covered)")
		}
	}
}

func TestVersionSentinelRecoverySkill(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Tier:       "full",
		Languages:  []types.LanguageChoice{{Name: "go"}},
		AgentTools: types.AgentToolsAnswers{VersionSentinel: true},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	var skillFile *types.GeneratedFile
	for i, f := range files {
		if f.Path == ".claude/skills/version-sentinel/SKILL.md" {
			skillFile = &files[i]
			break
		}
	}

	if skillFile == nil {
		t.Fatal("expected version-sentinel recovery skill")
		return
	}

	// The skill must describe the MCP tools qsdev actually ships, not a
	// blocking hook or slash commands that are never installed.
	content := string(skillFile.Content)
	for _, tool := range []string{"check_versions", "detect_drift", "manifest_coverage", "version_history"} {
		if !strings.Contains(content, tool) {
			t.Errorf("skill should describe the %s MCP tool", tool)
		}
	}
	for _, absent := range []string{"BLOCKED: version-sentinel", "/vs-record", "/check-versions"} {
		if strings.Contains(content, absent) {
			t.Errorf("skill references %q, which qsdev does not install", absent)
		}
	}
	for _, f := range files {
		if f.Path == ".version-sentinel/events.jsonl" {
			t.Error("events.jsonl is an append-only log and must not be generated (regeneration would wipe history)")
		}
	}
}

// TestVersionSentinelClaudeMdSection verifies the CLAUDE.md section only claims
// coverage for covered manifests and describes it as advisory: a Go-only
// project (no covered manifests) must not read "guards dependency changes in: .".
func TestVersionSentinelClaudeMdSection(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		langs []string
		want  string
	}{
		{
			name:  "go only",
			langs: []string{"go"},
			want: "<!-- qsdev:version-sentinel -->\n" +
				"- Version-Sentinel does NOT cover: go.mod. Review these manually.\n" +
				"<!-- /qsdev:version-sentinel -->",
		},
		{
			name:  "go and javascript",
			langs: []string{"go", "javascript"},
			want: "<!-- qsdev:version-sentinel -->\n" +
				"- **Version-Sentinel** MCP tools give advisory (non-blocking) dependency version checks for: package.json. Use them when changing dependencies.\n" +
				"- Version-Sentinel does NOT cover: go.mod. Review these manually.\n" +
				"<!-- /qsdev:version-sentinel -->",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			reg := newTestRegistry(t, goMock(), jsMock())
			answers := types.WizardAnswers{
				Tier:         "standard",
				AgentTools:   types.AgentToolsAnswers{VersionSentinel: true},
				EnabledTools: map[string]bool{"version-sentinel": true},
			}
			for _, l := range tc.langs {
				answers.Languages = append(answers.Languages, types.LanguageChoice{Name: l})
			}
			got, err := claudecode.GenerateClaudeMd(answers, reg)
			if err != nil {
				t.Fatal(err)
			}
			content := string(got.Content)
			if !strings.Contains(content, tc.want) {
				t.Errorf("CLAUDE.md version-sentinel section mismatch; want:\n%s\n\ngot:\n%s", tc.want, content)
			}
			if strings.Contains(content, "guards dependency changes") {
				t.Error("CLAUDE.md claims Version-Sentinel guards changes; it is advisory")
			}
		})
	}
}

func TestVersionSentinelIgnore_GoAndJsProject(t *testing.T) {
	reg := newTestRegistry(t, goMock(), jsMock())
	answers := types.WizardAnswers{
		Tier: "full",
		Languages: []types.LanguageChoice{
			{Name: "go"},
			{Name: "javascript"},
		},
		AgentTools: types.AgentToolsAnswers{VersionSentinel: true},
	}

	gen := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{})
	files, err := gen.Generate(answers)
	if err != nil {
		t.Fatal(err)
	}

	var ignoreFile *types.GeneratedFile
	for i, f := range files {
		if f.Path == ".version-sentinel/ignore" {
			ignoreFile = &files[i]
			break
		}
	}

	if ignoreFile == nil {
		t.Fatal("expected ignore file for Go+JS project (go.mod is uncovered)")
		return
	}

	content := string(ignoreFile.Content)
	if !strings.Contains(content, "go.mod") {
		t.Error("ignore file should contain go.mod")
	}
	if strings.Contains(content, "package.json") {
		t.Error("ignore file should NOT contain package.json (npm is covered)")
	}
}
