package claudecode_test

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ---------------------------------------------------------------------------
// loadQsdevOpsManifest tests
// ---------------------------------------------------------------------------

func TestLoadQsdevOpsManifest_Valid(t *testing.T) {
	manifest, err := claudecode.ExportLoadQsdevOpsManifest()
	if err != nil {
		t.Fatalf("loadQsdevOpsManifest returned error: %v", err)
	}

	if len(manifest.Skills) != 11 {
		t.Errorf("expected 11 skills in manifest, got %d", len(manifest.Skills))
	}

	for i, s := range manifest.Skills {
		if s.Name == "" {
			t.Errorf("skill %d has empty name", i)
		}
		if s.Description == "" {
			t.Errorf("skill %d (%s) has empty description", i, s.Name)
		}
		if len(s.Tags) == 0 {
			t.Errorf("skill %d (%s) has no tags", i, s.Name)
		}
	}
}

func TestLoadQsdevOpsManifest_AllFilesExist(t *testing.T) {
	manifest, err := claudecode.ExportLoadQsdevOpsManifest()
	if err != nil {
		t.Fatalf("loadQsdevOpsManifest returned error: %v", err)
	}

	// Deploy all skills (nil EnabledTools = deploy all) to verify embedded files exist.
	answers := types.WizardAnswers{}
	files, err := claudecode.ExportDeployOperationSkills(answers)
	if err != nil {
		t.Fatalf("deployOperationSkills returned error: %v", err)
	}

	if len(files) != len(manifest.Skills) {
		t.Errorf("expected %d files (one per manifest entry), got %d", len(manifest.Skills), len(files))
	}

	// Verify each manifest entry has a corresponding file.
	fileSet := make(map[string]bool)
	for _, f := range files {
		fileSet[f.Path] = true
	}
	for _, s := range manifest.Skills {
		expected := ".claude/skills/" + s.Name + "/SKILL.md"
		if !fileSet[expected] {
			t.Errorf("manifest entry %q has no corresponding file at %q", s.Name, expected)
		}
	}
}

// ---------------------------------------------------------------------------
// deployOperationSkills tests
// ---------------------------------------------------------------------------

func TestDeployOperationSkills_DeploysAll(t *testing.T) {
	// When EnabledTools is nil, all skills should be deployed.
	answers := types.WizardAnswers{}
	files, err := claudecode.ExportDeployOperationSkills(answers)
	if err != nil {
		t.Fatalf("deployOperationSkills returned error: %v", err)
	}

	if len(files) != 11 {
		t.Errorf("expected 11 files when EnabledTools is nil, got %d", len(files))
	}
}

func TestDeployOperationSkills_RespectsEnabledTools(t *testing.T) {
	answers := types.WizardAnswers{
		EnabledTools: map[string]bool{
			"qsdev-init":   true,
			"qsdev-doctor": true,
			// All others implicitly false or absent.
		},
	}

	files, err := claudecode.ExportDeployOperationSkills(answers)
	if err != nil {
		t.Fatalf("deployOperationSkills returned error: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files when only 2 enabled, got %d", len(files))
	}

	paths := make(map[string]bool)
	for _, f := range files {
		paths[f.Path] = true
	}
	if !paths[".claude/skills/qsdev-init/SKILL.md"] {
		t.Error("missing qsdev-init skill")
	}
	if !paths[".claude/skills/qsdev-doctor/SKILL.md"] {
		t.Error("missing qsdev-doctor skill")
	}
}

func TestDeployOperationSkills_ContentNotEmpty(t *testing.T) {
	answers := types.WizardAnswers{}
	files, err := claudecode.ExportDeployOperationSkills(answers)
	if err != nil {
		t.Fatalf("deployOperationSkills returned error: %v", err)
	}

	for _, f := range files {
		if len(f.Content) == 0 {
			t.Errorf("skill file %q has empty content", f.Path)
		}
	}
}

func TestDeployOperationSkills_UserOnlySkills(t *testing.T) {
	manifest, err := claudecode.ExportLoadQsdevOpsManifest()
	if err != nil {
		t.Fatalf("loadQsdevOpsManifest returned error: %v", err)
	}

	answers := types.WizardAnswers{}
	files, err := claudecode.ExportDeployOperationSkills(answers)
	if err != nil {
		t.Fatalf("deployOperationSkills returned error: %v", err)
	}

	// Build name -> content map.
	contentMap := make(map[string]string)
	for _, f := range files {
		// Extract skill name from path: .claude/skills/{name}/SKILL.md
		parts := strings.Split(f.Path, "/")
		if len(parts) >= 3 {
			contentMap[parts[2]] = string(f.Content)
		}
	}

	for _, entry := range manifest.Skills {
		content, ok := contentMap[entry.Name]
		if !ok {
			t.Errorf("skill %q not found in deployed files", entry.Name)
			continue
		}

		hasDisableModelInvocation := strings.Contains(content, "disable-model-invocation: true")

		if entry.UserOnly && !hasDisableModelInvocation {
			t.Errorf("user-only skill %q should have 'disable-model-invocation: true' in frontmatter", entry.Name)
		}
		if !entry.UserOnly && hasDisableModelInvocation {
			t.Errorf("claude-invocable skill %q should NOT have 'disable-model-invocation: true' in frontmatter", entry.Name)
		}
	}
}

func TestDeployOperationSkills_AllowedTools(t *testing.T) {
	answers := types.WizardAnswers{}
	files, err := claudecode.ExportDeployOperationSkills(answers)
	if err != nil {
		t.Fatalf("deployOperationSkills returned error: %v", err)
	}

	for _, f := range files {
		content := string(f.Content)
		if !strings.Contains(content, "Bash(qsdev *)") {
			t.Errorf("skill file %q should have 'allowed-tools' containing 'Bash(qsdev *)'", f.Path)
		}
	}
}

func TestDeployOperationSkills_FileMetadata(t *testing.T) {
	answers := types.WizardAnswers{}
	files, err := claudecode.ExportDeployOperationSkills(answers)
	if err != nil {
		t.Fatalf("deployOperationSkills returned error: %v", err)
	}

	for _, f := range files {
		if !strings.HasPrefix(f.Path, ".claude/skills/qsdev-") {
			t.Errorf("path %q does not start with .claude/skills/qsdev-", f.Path)
		}
		if !strings.HasSuffix(f.Path, "/SKILL.md") {
			t.Errorf("path %q does not end with /SKILL.md", f.Path)
		}
		if f.Mode != 0o644 {
			t.Errorf("file %q has mode %o, want 0o644", f.Path, f.Mode)
		}
		if f.Strategy != types.LibraryManaged {
			t.Errorf("file %q has strategy %v, want LibraryManaged", f.Path, f.Strategy)
		}
		if f.Owner == "" {
			t.Errorf("file %q has empty Owner", f.Path)
		}
	}
}

func TestAvailableQsdevOpsSkillNames(t *testing.T) {
	names := claudecode.AvailableQsdevOpsSkillNames()
	if len(names) != 11 {
		t.Errorf("expected 11 qsdev-ops skill names, got %d", len(names))
	}

	expected := map[string]bool{
		"qsdev-init":    true,
		"qsdev-onboard": true,
		"qsdev-setup":   true,
		"qsdev-enable":  true,
		"qsdev-disable": true,
		"qsdev-update":  true,
		"qsdev-doctor":  true,
		"qsdev-status":  true,
		"qsdev-tools":   true,
		"qsdev-detect":  true,
		"qsdev-add-dep": true,
	}

	for _, name := range names {
		if !expected[name] {
			t.Errorf("unexpected skill name: %q", name)
		}
	}
}

// TestQsdevOpsManifest_EveryEntryRegisteredAlwaysOn verifies every operation
// skill in the manifest has an always-on tool in the registry. Without one,
// MergeInferredTools never sets EnabledTools[name], so deployOperationSkills
// silently skips the skill (qsdev-add-dep was missing this way).
func TestQsdevOpsManifest_EveryEntryRegisteredAlwaysOn(t *testing.T) {
	t.Parallel()
	reg := toolreg.DefaultRegistry()
	for _, name := range claudecode.AvailableQsdevOpsSkillNames() {
		tool, ok := reg.ByName(name)
		if !ok {
			t.Errorf("ops skill %q has no registered tool", name)
			continue
		}
		if tool.Default != toolreg.AlwaysOn {
			t.Errorf("ops skill tool %q default = %v, want AlwaysOn", name, tool.Default)
		}
	}

	answers := types.WizardAnswers{Tier: "full"}
	toolreg.MergeInferredTools(&answers, reg)
	files, err := claudecode.ExportDeployOperationSkills(answers)
	if err != nil {
		t.Fatalf("deployOperationSkills: %v", err)
	}
	if got, want := len(files), len(claudecode.AvailableQsdevOpsSkillNames()); got != want {
		t.Errorf("deployed %d ops skills after MergeInferredTools, want %d", got, want)
	}
}

// skillInjectionRE matches a Claude Code skill shell injection: !`command`.
var skillInjectionRE = regexp.MustCompile("!`([^`]*)`")

// TestSkillInjections_NoFabricatedFallbacks verifies that no skill's shell
// injection masks a failed command by printing made-up data (e.g.
// `|| echo '{"available": []}'`), which the agent would present as real,
// empty state. Failures must surface as an explicit marker instead.
func TestSkillInjections_NoFabricatedFallbacks(t *testing.T) {
	t.Parallel()
	fabricated := regexp.MustCompile(`\|\|\s*echo\s+'[\[{]`)
	var checked int
	err := fs.WalkDir(claudecode.ExportTemplateFS, "templates/skills", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		content, err := claudecode.ExportTemplateFS.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range skillInjectionRE.FindAllStringSubmatch(string(content), -1) {
			checked++
			if fabricated.MatchString(m[1]) {
				t.Errorf("%s: injection %q fabricates data on failure; emit an explicit error marker", path, m[1])
			}
			if strings.HasSuffix(m[1], "\\") {
				t.Errorf("%s: injection %q ends in a backslash (escaped backtick splits the injection)", path, m[1])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked == 0 {
		t.Fatal("no skill shell injections found; the pattern is stale")
	}
}

// TestQsdevOpsManifest_UserOnlyMatchesTemplates verifies each operation
// skill's manifest user_only flag matches its SKILL.md: user-only skills must
// set disable-model-invocation so the model cannot invoke them on its own.
func TestQsdevOpsManifest_UserOnlyMatchesTemplates(t *testing.T) {
	t.Parallel()
	manifest, err := claudecode.ExportLoadQsdevOpsManifest()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range manifest.Skills {
		content, err := claudecode.ExportTemplateFS.ReadFile("templates/skills/" + e.Name + "/SKILL.md")
		if err != nil {
			t.Fatalf("reading skill %q: %v", e.Name, err)
		}
		disabled := strings.Contains(string(content), "\ndisable-model-invocation: true\n")
		if e.UserOnly != disabled {
			t.Errorf("skill %q: manifest user_only=%v but disable-model-invocation=%v", e.Name, e.UserOnly, disabled)
		}
	}
}
