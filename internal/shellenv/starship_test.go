package shellenv

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestGenerateStarshipToml(t *testing.T) {
	answers := types.WizardAnswers{
		ProjectName: "myapp",
	}

	got, err := GenerateStarshipToml(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Path != ".starship.toml" {
		t.Errorf("Path = %q, want %q", got.Path, ".starship.toml")
	}
	if got.Mode != 0o644 {
		t.Errorf("Mode = %#o, want %#o", got.Mode, 0o644)
	}
	if got.Strategy != types.Overwrite {
		t.Errorf("Strategy = %v, want Overwrite", got.Strategy)
	}

	content := string(got.Content)

	if len(content) == 0 {
		t.Fatal("generated content is empty")
	}

	// Verify custom.gdev sections are present.
	for _, section := range []string{"[custom.gdev]", "[custom.gdev_security]", "[custom.gdev_tools]"} {
		if !strings.Contains(content, section) {
			t.Errorf("content does not contain section %q", section)
		}
	}

	// Verify references to gdev environment variables.
	for _, envVar := range []string{"GDEV_PROJECT_NAME", "GDEV_SECURITY_PROFILE", "GDEV_TOOL_COUNT"} {
		if !strings.Contains(content, envVar) {
			t.Errorf("content does not reference %q", envVar)
		}
	}
}

func TestGenerateStarshipToml_ContainsDescriptions(t *testing.T) {
	answers := types.WizardAnswers{}

	got, err := GenerateStarshipToml(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	// Each section should have a description.
	descriptions := []string{
		"Active gdev project",
		"gdev security profile",
		"Active tool count",
	}
	for _, desc := range descriptions {
		if !strings.Contains(content, desc) {
			t.Errorf("content does not contain description %q", desc)
		}
	}
}
