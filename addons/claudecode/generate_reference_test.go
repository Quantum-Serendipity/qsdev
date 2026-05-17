package claudecode_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestGenerateQsdevReference_NotNil(t *testing.T) {
	answers := types.WizardAnswers{ProjectName: "test"}
	reg := ecosystem.NewRegistry()

	got, err := claudecode.GenerateQsdevReference(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if got.Path != ".claude/qsdev-reference.md" {
		t.Errorf("Path = %q, want %q", got.Path, ".claude/qsdev-reference.md")
	}
	if got.Mode != 0o644 {
		t.Errorf("Mode = %#o, want %#o", got.Mode, 0o644)
	}
	if got.Strategy != types.LibraryManaged {
		t.Errorf("Strategy = %v, want LibraryManaged", got.Strategy)
	}
}

func TestGenerateQsdevReference_ContainsCommands(t *testing.T) {
	answers := types.WizardAnswers{ProjectName: "test"}
	reg := ecosystem.NewRegistry()

	got, err := claudecode.GenerateQsdevReference(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	// Check for key CLI commands.
	for _, cmd := range []string{
		"qsdev init",
		"qsdev init --update",
		"qsdev init --mode join",
		"qsdev devenv doctor",
		"qsdev devenv setup",
		"qsdev enable <tool>",
		"qsdev disable <tool>",
		"qsdev status",
		"qsdev list",
		"qsdev check",
		"qsdev check --format json",
		"qsdev check --audit-level medium",
		"qsdev config migrate",
	} {
		requireContains(t, content, cmd)
	}
}

func TestGenerateQsdevReference_ContainsTroubleshooting(t *testing.T) {
	answers := types.WizardAnswers{ProjectName: "test"}
	reg := ecosystem.NewRegistry()

	got, err := claudecode.GenerateQsdevReference(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	requireContains(t, content, "Troubleshooting")
	requireContains(t, content, "qsdev commands not found")
	requireContains(t, content, "devenv not activated")
	requireContains(t, content, "Permission denied")
}

func TestGenerateQsdevReference_ContainsSecurityPolicy(t *testing.T) {
	answers := types.WizardAnswers{ProjectName: "test"}
	reg := ecosystem.NewRegistry()

	got, err := claudecode.GenerateQsdevReference(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	requireContains(t, content, "Security Policy")
	requireContains(t, content, "deny rules")
	requireContains(t, content, "qsdev enable")
	requireContains(t, content, "qsdev devenv doctor")
}
