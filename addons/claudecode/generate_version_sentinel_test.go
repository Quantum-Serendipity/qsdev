package claudecode_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
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

// elixirMock declares mix.exs, a manifest Version-Sentinel has no check for.
func elixirMock() *ecosystem.MockModule {
	return &ecosystem.MockModule{
		NameVal:        "elixir",
		DisplayNameVal: "Elixir",
		TierVal:        2,
		ManifestFilesVal: []ecosystem.ManifestFileInfo{
			{Path: "mix.exs", Ecosystem: "elixir", LockFile: "mix.lock"},
		},
	}
}

// pnpmMock declares package.json locked by pnpm-lock.yaml, which
// Version-Sentinel checks only for presence.
func pnpmMock() *ecosystem.MockModule {
	return &ecosystem.MockModule{
		NameVal:        "pnpm",
		DisplayNameVal: "pnpm",
		TierVal:        2,
		ManifestFilesVal: []ecosystem.ManifestFileInfo{
			{Path: "package.json", Ecosystem: "npm", LockFile: "pnpm-lock.yaml"},
		},
	}
}

// versionSentinelIgnore returns the generated .version-sentinel/ignore, or
// nil when none is generated. The file comes from the registered ecosystem
// modules, so langs name real ones.
func versionSentinelIgnore(t *testing.T, langs ...string) *types.GeneratedFile {
	t.Helper()
	reg := newTestRegistry(t, goMock(), jsMock(), elixirMock())
	answers := types.WizardAnswers{
		Tier:       "full",
		AgentTools: types.AgentToolsAnswers{VersionSentinel: true},
	}
	for _, l := range langs {
		answers.Languages = append(answers.Languages, types.LanguageChoice{Name: l})
	}
	files, err := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{}).Generate(answers)
	if err != nil {
		t.Fatal(err)
	}
	for i, f := range files {
		if f.Path == ".version-sentinel/ignore" {
			return &files[i]
		}
	}
	return nil
}

// TestVersionSentinelIgnore lists exactly the manifests whose versions
// Version-Sentinel never compares (manifest_coverage's presence_only and
// uncovered), and is absent when every manifest is version-diffed.
func TestVersionSentinelIgnore(t *testing.T) {
	t.Parallel()
	tests := []struct {
		langs []string
		want  []string // nil: no ignore file
	}{
		{langs: []string{"go"}},
		{langs: []string{"go", "javascript"}},
		{langs: []string{"go", "elixir"}, want: []string{"mix.exs"}},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.langs, "+"), func(t *testing.T) {
			t.Parallel()
			f := versionSentinelIgnore(t, tt.langs...)
			if tt.want == nil {
				if f != nil {
					t.Fatalf("unexpected ignore file:\n%s", f.Content)
				}
				return
			}
			if f == nil {
				t.Fatal("no ignore file")
			}
			var got []string
			for _, line := range strings.Split(strings.TrimSpace(string(f.Content)), "\n") {
				if !strings.HasPrefix(line, "#") {
					got = append(got, line)
				}
			}
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("ignore lists %q, want %q", got, tt.want)
			}
		})
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

// TestVersionSentinelClaudeMdSection verifies the CLAUDE.md section states
// the coverage manifest_coverage reports (version-diffed, lockfile presence
// only, not covered) and describes it as advisory, never "guards dependency
// changes".
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
				"- **Version-Sentinel** MCP tools give advisory (non-blocking) dependency version checks for: go.mod. Use them when changing dependencies.\n" +
				"<!-- /qsdev:version-sentinel -->",
		},
		{
			name:  "go and javascript",
			langs: []string{"go", "javascript"},
			want: "<!-- qsdev:version-sentinel -->\n" +
				"- **Version-Sentinel** MCP tools give advisory (non-blocking) dependency version checks for: go.mod, package.json. Use them when changing dependencies.\n" +
				"<!-- /qsdev:version-sentinel -->",
		},
		{
			name:  "every coverage class",
			langs: []string{"go", "pnpm", "elixir"},
			want: "<!-- qsdev:version-sentinel -->\n" +
				"- **Version-Sentinel** MCP tools give advisory (non-blocking) dependency version checks for: go.mod. Use them when changing dependencies.\n" +
				"- Version-Sentinel only checks that a lockfile exists for: package.json. Their versions are not compared; review them manually.\n" +
				"- Version-Sentinel does NOT cover: mix.exs. Review these manually.\n" +
				"<!-- /qsdev:version-sentinel -->",
		},
		{
			name:  "uncovered only",
			langs: []string{"elixir"},
			want: "<!-- qsdev:version-sentinel -->\n" +
				"- Version-Sentinel does NOT cover: mix.exs. Review these manually.\n" +
				"<!-- /qsdev:version-sentinel -->",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			reg := newTestRegistry(t, goMock(), jsMock(), elixirMock(), pnpmMock())
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
