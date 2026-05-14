package claudecode_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/claudecode"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// ---------------------------------------------------------------------------
// loadConsultingSkillManifest tests
// ---------------------------------------------------------------------------

func TestLoadConsultingSkillManifest_Valid(t *testing.T) {
	manifest, err := claudecode.ExportLoadConsultingSkillManifest()
	if err != nil {
		t.Fatalf("loadConsultingSkillManifest returned error: %v", err)
	}

	if len(manifest.Skills) != 8 {
		t.Errorf("expected 8 skills in consulting manifest, got %d", len(manifest.Skills))
	}

	for i, s := range manifest.Skills {
		if s.Name == "" {
			t.Errorf("skill %d has empty name", i)
		}
		if s.Description == "" {
			t.Errorf("skill %d (%s) has empty description", i, s.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// deployWorkflowSkills tests
// ---------------------------------------------------------------------------

func TestDeployWorkflowSkills_SkipsWhenDisabled(t *testing.T) {
	// EnabledTools is nil — all consulting workflow skills should be skipped.
	answers := types.WizardAnswers{
		EnabledTools: nil,
	}

	files, err := claudecode.ExportDeployWorkflowSkills(answers, ecosystem.NewRegistry())
	if err != nil {
		t.Fatalf("deployWorkflowSkills returned error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected 0 files when EnabledTools is nil, got %d", len(files))
	}
}

func TestDeployWorkflowSkills_DeploysEnabled(t *testing.T) {
	answers := types.WizardAnswers{
		EnabledTools: map[string]bool{
			"consulting-workflow-review-pr":  true,
			"consulting-workflow-write-adr":  true,
			"consulting-workflow-add-tests":  false, // explicitly disabled
		},
	}

	files, err := claudecode.ExportDeployWorkflowSkills(answers, ecosystem.NewRegistry())
	if err != nil {
		t.Fatalf("deployWorkflowSkills returned error: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	paths := make(map[string]bool)
	for _, f := range files {
		paths[f.Path] = true
	}
	if !paths[".claude/skills/review-pr/SKILL.md"] {
		t.Error("missing .claude/skills/review-pr/SKILL.md")
	}
	if !paths[".claude/skills/write-adr/SKILL.md"] {
		t.Error("missing .claude/skills/write-adr/SKILL.md")
	}
}

func TestDeployWorkflowSkills_AllHaveDisableModelInvocation(t *testing.T) {
	// Enable all skills and verify each has disable-model-invocation in frontmatter.
	manifest, err := claudecode.ExportLoadConsultingSkillManifest()
	if err != nil {
		t.Fatalf("loadConsultingSkillManifest returned error: %v", err)
	}

	enabledTools := make(map[string]bool, len(manifest.Skills))
	for _, s := range manifest.Skills {
		enabledTools["consulting-workflow-"+s.Name] = true
	}

	answers := types.WizardAnswers{
		EnabledTools: enabledTools,
	}

	files, err := claudecode.ExportDeployWorkflowSkills(answers, ecosystem.NewRegistry())
	if err != nil {
		t.Fatalf("deployWorkflowSkills returned error: %v", err)
	}

	for _, f := range files {
		content := string(f.Content)
		if !strings.Contains(content, "disable-model-invocation: true") {
			t.Errorf("skill file %q does not contain 'disable-model-invocation: true'", f.Path)
		}
	}
}

func TestDeployWorkflowSkills_DirectoryFormat(t *testing.T) {
	// Verify paths are in .claude/skills/{name}/SKILL.md format.
	answers := types.WizardAnswers{
		EnabledTools: map[string]bool{
			"consulting-workflow-incident-debug":  true,
			"consulting-workflow-migration-plan":  true,
		},
	}

	files, err := claudecode.ExportDeployWorkflowSkills(answers, ecosystem.NewRegistry())
	if err != nil {
		t.Fatalf("deployWorkflowSkills returned error: %v", err)
	}

	for _, f := range files {
		if !strings.HasPrefix(f.Path, ".claude/skills/") {
			t.Errorf("path %q does not start with .claude/skills/", f.Path)
		}
		if !strings.HasSuffix(f.Path, "/SKILL.md") {
			t.Errorf("path %q does not end with /SKILL.md", f.Path)
		}
		if f.Mode != 0o644 {
			t.Errorf("file %q has mode %o, want 0644", f.Path, f.Mode)
		}
		if f.Strategy != types.LibraryManaged {
			t.Errorf("file %q has strategy %v, want LibraryManaged", f.Path, f.Strategy)
		}
	}
}

func TestAvailableConsultingSkillNames(t *testing.T) {
	names := claudecode.AvailableConsultingSkillNames()

	if len(names) != 8 {
		t.Fatalf("expected 8 consulting skill names, got %d: %v", len(names), names)
	}

	// Verify all expected names are present (sorted).
	expected := []string{
		"add-tests",
		"handoff-doc",
		"incident-debug",
		"migration-plan",
		"onboard-me",
		"review-pr",
		"upgrade-dep",
		"write-adr",
	}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("name[%d] = %q, want %q", i, names[i], want)
		}
	}
}

func TestDeployWorkflowSkills_OwnerSet(t *testing.T) {
	answers := types.WizardAnswers{
		EnabledTools: map[string]bool{
			"consulting-workflow-handoff-doc": true,
		},
	}

	files, err := claudecode.ExportDeployWorkflowSkills(answers, ecosystem.NewRegistry())
	if err != nil {
		t.Fatalf("deployWorkflowSkills returned error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	if files[0].Owner != "consulting-workflow-handoff-doc" {
		t.Errorf("Owner = %q, want %q", files[0].Owner, "consulting-workflow-handoff-doc")
	}
}
