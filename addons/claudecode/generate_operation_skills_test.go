package claudecode_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
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
