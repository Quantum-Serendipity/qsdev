package devenv

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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

	// Verify qsdev env vars are present.
	checks := map[string]string{
		"QSDEV_PROJECT_NAME":     "myproject",
		"QSDEV_SECURITY_PROFILE": "enhanced",
		"QSDEV_ECOSYSTEMS":       "go,python",
		"QSDEV_TOOL_COUNT":       "2",
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

	// QSDEV_VERSION should be present and non-empty.
	if v, ok := data.EnvVars["QSDEV_VERSION"]; !ok || v == "" {
		t.Error("EnvVars missing or empty QSDEV_VERSION")
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

	if v := data.EnvVars["QSDEV_PROJECT_NAME"]; v != "unknown" {
		t.Errorf("default QSDEV_PROJECT_NAME = %q, want %q", v, "unknown")
	}
	if v := data.EnvVars["QSDEV_SECURITY_PROFILE"]; v != "standard" {
		t.Errorf("default QSDEV_SECURITY_PROFILE = %q, want %q", v, "standard")
	}
	if v := data.EnvVars["QSDEV_ECOSYSTEMS"]; v != "none" {
		t.Errorf("default QSDEV_ECOSYSTEMS = %q, want %q", v, "none")
	}
	if v := data.EnvVars["QSDEV_TOOL_COUNT"]; v != "0" {
		t.Errorf("default QSDEV_TOOL_COUNT = %q, want %q", v, "0")
	}
}

func TestBuildDevenvNixData_HookPackagesBareName(t *testing.T) {
	t.Parallel()
	reg := ecosystem.NewRegistry()
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "go",
		DisplayNameVal: "Go",
		TierVal:        1,
		PreCommitHooksVal: []ecosystem.HookConfig{
			{
				ID:         "staticcheck",
				Name:       "staticcheck",
				Entry:      "staticcheck ./...",
				NixPackage: "go-tools",
				Language:   "system",
				Types:      []string{"go"},
				Stages:     []string{"pre-commit"},
			},
		},
	})

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{{Name: "go"}},
	}

	data, err := BuildDevenvNixData(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, pkg := range data.Packages {
		if pkg == "pkgs.go-tools" {
			t.Error("data.Packages contains \"pkgs.go-tools\"; want bare name \"go-tools\"")
		}
	}

	found := false
	for _, pkg := range data.Packages {
		if pkg == "go-tools" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("data.Packages does not contain \"go-tools\"; got %v", data.Packages)
	}
}

func TestBuildDevenvNixData_NoUvInBasePackages(t *testing.T) {
	t.Parallel()
	reg := ecosystem.NewRegistry()
	answers := types.WizardAnswers{}

	data, err := BuildDevenvNixData(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, pkg := range data.Packages {
		if pkg == "uv" {
			t.Error("data.Packages contains \"uv\"; Python tool should not be in base packages")
		}
	}
}

func TestBuildDevenvNixData_ModulePackagesCollected(t *testing.T) {
	t.Parallel()
	reg := ecosystem.NewRegistry()
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:           "go",
		DisplayNameVal:    "Go",
		TierVal:           1,
		DevenvPackagesVal: []string{"gopls", "delve"},
	})

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{{Name: "go"}},
	}

	data, err := BuildDevenvNixData(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pkgSet := make(map[string]bool, len(data.Packages))
	for _, p := range data.Packages {
		pkgSet[p] = true
	}
	for _, want := range []string{"gopls", "delve"} {
		if !pkgSet[want] {
			t.Errorf("data.Packages missing %q from module PackageProvider; got %v", want, data.Packages)
		}
	}
}

func TestBuildEnterShellScript_ContainsGdevVars(t *testing.T) {
	script := buildEnterShellScript()

	for _, want := range []string{"QSDEV_PROJECT_NAME", "QSDEV_SECURITY_PROFILE", "QSDEV_TOOL_COUNT"} {
		if !strings.Contains(script, want) {
			t.Errorf("enterShell script does not contain %q", want)
		}
	}
}
