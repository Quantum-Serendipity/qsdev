package claudecode_test

import (
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/claudecode"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestGenerateGdevReference_NotNil(t *testing.T) {
	answers := types.WizardAnswers{ProjectName: "test"}
	reg := ecosystem.NewRegistry()

	got, err := claudecode.GenerateGdevReference(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if got.Path != ".claude/gdev-reference.md" {
		t.Errorf("Path = %q, want %q", got.Path, ".claude/gdev-reference.md")
	}
	if got.Mode != 0o644 {
		t.Errorf("Mode = %#o, want %#o", got.Mode, 0o644)
	}
	if got.Strategy != types.LibraryManaged {
		t.Errorf("Strategy = %v, want LibraryManaged", got.Strategy)
	}
}

func TestGenerateGdevReference_ContainsCommands(t *testing.T) {
	answers := types.WizardAnswers{ProjectName: "test"}
	reg := ecosystem.NewRegistry()

	got, err := claudecode.GenerateGdevReference(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	// Check for key CLI commands.
	for _, cmd := range []string{
		"gdev init",
		"gdev init --update",
		"gdev init --mode join",
		"gdev devenv doctor",
		"gdev devenv setup",
		"gdev enable <tool>",
		"gdev disable <tool>",
		"gdev status",
		"gdev list",
		"gdev check",
		"gdev check --format json",
		"gdev check --audit-level medium",
		"gdev config migrate",
	} {
		requireContains(t, content, cmd)
	}
}

func TestGenerateGdevReference_ContainsTroubleshooting(t *testing.T) {
	answers := types.WizardAnswers{ProjectName: "test"}
	reg := ecosystem.NewRegistry()

	got, err := claudecode.GenerateGdevReference(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	requireContains(t, content, "Troubleshooting")
	requireContains(t, content, "gdev commands not found")
	requireContains(t, content, "devenv not activated")
	requireContains(t, content, "Permission denied")
}

func TestGenerateGdevReference_ContainsSecurityPolicy(t *testing.T) {
	answers := types.WizardAnswers{ProjectName: "test"}
	reg := ecosystem.NewRegistry()

	got, err := claudecode.GenerateGdevReference(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	requireContains(t, content, "Security Policy")
	requireContains(t, content, "deny rules")
	requireContains(t, content, "gdev enable")
	requireContains(t, content, "gdev devenv doctor")
}
