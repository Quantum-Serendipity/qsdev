package devenv

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestBuildEcosystemsList_Empty(t *testing.T) {
	answers := types.WizardAnswers{}
	got := buildEcosystemsList(answers)
	if got != "none" {
		t.Errorf("buildEcosystemsList({}) = %q, want %q", got, "none")
	}
}

func TestBuildEcosystemsList_Single(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go"},
		},
	}
	got := buildEcosystemsList(answers)
	if got != "go" {
		t.Errorf("buildEcosystemsList({go}) = %q, want %q", got, "go")
	}
}

func TestBuildEcosystemsList_Multiple_Sorted(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "python"},
			{Name: "go"},
			{Name: "rust"},
		},
	}
	got := buildEcosystemsList(answers)
	if got != "go,python,rust" {
		t.Errorf("buildEcosystemsList({python, go, rust}) = %q, want %q", got, "go,python,rust")
	}
}

func TestCountEnabledTools_Empty(t *testing.T) {
	answers := types.WizardAnswers{}
	got := countEnabledTools(answers)
	if got != 0 {
		t.Errorf("countEnabledTools({}) = %d, want 0", got)
	}
}

func TestCountEnabledTools_NilMap(t *testing.T) {
	answers := types.WizardAnswers{EnabledTools: nil}
	got := countEnabledTools(answers)
	if got != 0 {
		t.Errorf("countEnabledTools(nil map) = %d, want 0", got)
	}
}

func TestCountEnabledTools_Mixed(t *testing.T) {
	answers := types.WizardAnswers{
		EnabledTools: map[string]bool{
			"tool-a": true,
			"tool-b": false,
			"tool-c": true,
			"tool-d": true,
		},
	}
	got := countEnabledTools(answers)
	if got != 3 {
		t.Errorf("countEnabledTools(3 true, 1 false) = %d, want 3", got)
	}
}

func TestBuildDevenvNixData_GdevEnvVars(t *testing.T) {
	reg := ecosystem.NewRegistry()

	answers := types.WizardAnswers{
		ProjectName:     "myproject",
		ComplianceLevel: "enhanced",
		EnabledTools: map[string]bool{
			"tool-a": true,
			"tool-b": true,
		},
		Languages: []types.LanguageChoice{
			{Name: "go"},
			{Name: "python"},
		},
	}

	// Register mock modules so BuildDevenvNixData doesn't error.
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "go",
		DisplayNameVal: "Go",
		TierVal:        1,
	})
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "python",
		DisplayNameVal: "Python",
		TierVal:        1,
	})

	data, err := BuildDevenvNixData(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify gdev env vars are present.
	checks := map[string]string{
		"GDEV_PROJECT_NAME":    "myproject",
		"GDEV_SECURITY_PROFILE": "enhanced",
		"GDEV_ECOSYSTEMS":      "go,python",
		"GDEV_TOOL_COUNT":      "2",
	}
	for key, want := range checks {
		got, ok := data.EnvVars[key]
		if !ok {
			t.Errorf("EnvVars missing key %q", key)
			continue
		}
		if got != want {
			t.Errorf("EnvVars[%q] = %q, want %q", key, got, want)
		}
	}

	// GDEV_VERSION should be present and non-empty.
	if v, ok := data.EnvVars["GDEV_VERSION"]; !ok || v == "" {
		t.Error("EnvVars missing or empty GDEV_VERSION")
	}
}

func TestBuildDevenvNixData_GdevEnvVars_Defaults(t *testing.T) {
	reg := ecosystem.NewRegistry()

	// Empty answers should produce safe defaults.
	answers := types.WizardAnswers{}

	data, err := BuildDevenvNixData(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v := data.EnvVars["GDEV_PROJECT_NAME"]; v != "unknown" {
		t.Errorf("default GDEV_PROJECT_NAME = %q, want %q", v, "unknown")
	}
	if v := data.EnvVars["GDEV_SECURITY_PROFILE"]; v != "standard" {
		t.Errorf("default GDEV_SECURITY_PROFILE = %q, want %q", v, "standard")
	}
	if v := data.EnvVars["GDEV_ECOSYSTEMS"]; v != "none" {
		t.Errorf("default GDEV_ECOSYSTEMS = %q, want %q", v, "none")
	}
	if v := data.EnvVars["GDEV_TOOL_COUNT"]; v != "0" {
		t.Errorf("default GDEV_TOOL_COUNT = %q, want %q", v, "0")
	}
}

func TestBuildEnterShellScript_ContainsGdevVars(t *testing.T) {
	script := buildEnterShellScript()

	for _, want := range []string{"GDEV_PROJECT_NAME", "GDEV_SECURITY_PROFILE", "GDEV_TOOL_COUNT"} {
		if !strings.Contains(script, want) {
			t.Errorf("enterShell script does not contain %q", want)
		}
	}
}
