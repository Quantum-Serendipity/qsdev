package toolreg

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestShellToolsRegistered(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range []string{"starship-integration", "otel-config"} {
		tool, ok := reg.ByName(name)
		if !ok {
			t.Errorf("tool %q not found in DefaultRegistry", name)
			continue
		}
		if tool.DisplayName == "" {
			t.Errorf("tool %q has empty DisplayName", name)
		}
		if tool.Description == "" {
			t.Errorf("tool %q has empty Description", name)
		}
	}
}

func TestStarshipIntegrationCategory(t *testing.T) {
	reg := DefaultRegistry()
	tool, ok := reg.ByName("starship-integration")
	if !ok {
		t.Fatal("starship-integration not found in registry")
	}
	if tool.Category != CategoryDevEx {
		t.Errorf("category = %v, want %v", tool.Category, CategoryDevEx)
	}
}

func TestOtelConfigCategory(t *testing.T) {
	reg := DefaultRegistry()
	tool, ok := reg.ByName("otel-config")
	if !ok {
		t.Fatal("otel-config not found in registry")
	}
	if tool.Category != CategoryInfrastructure {
		t.Errorf("category = %v, want %v", tool.Category, CategoryInfrastructure)
	}
}

func TestShellToolsOptIn(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range []string{"starship-integration", "otel-config"} {
		tool, ok := reg.ByName(name)
		if !ok {
			t.Errorf("tool %q not found", name)
			continue
		}
		if tool.Default != OptIn {
			t.Errorf("tool %q has Default = %v, want OptIn", name, tool.Default)
		}
	}
}

func TestShellToolEnableDisable(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range []string{"starship-integration", "otel-config"} {
		t.Run(name, func(t *testing.T) {
			tool, ok := reg.ByName(name)
			if !ok {
				t.Fatalf("tool %q not found", name)
			}

			if tool.EnableFunc == nil {
				t.Fatal("EnableFunc is nil")
			}
			if tool.DisableFunc == nil {
				t.Fatal("DisableFunc is nil")
			}

			// Enable with nil EnabledTools map.
			answers := &types.WizardAnswers{}
			tool.EnableFunc(answers)

			if answers.EnabledTools == nil {
				t.Fatal("EnableFunc did not initialize EnabledTools")
			}
			if !answers.EnabledTools[name] {
				t.Errorf("after EnableFunc, EnabledTools[%q] should be true", name)
			}

			// Disable.
			tool.DisableFunc(answers)

			if answers.EnabledTools[name] {
				t.Errorf("after DisableFunc, EnabledTools[%q] should be false", name)
			}
		})
	}
}

func TestStarshipSharedContent(t *testing.T) {
	reg := DefaultRegistry()
	tool, ok := reg.ByName("starship-integration")
	if !ok {
		t.Fatal("starship-integration not found in registry")
	}

	if tool.SharedContent == nil {
		t.Fatal("SharedContent map is nil")
	}

	fn, ok := tool.SharedContent["starship"]
	if !ok {
		t.Fatal("SharedContent missing 'starship' key")
	}

	content, err := fn(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("SharedContent['starship'] returned error: %v", err)
	}

	s := string(content)
	if !strings.Contains(s, "STARSHIP_CONFIG") {
		t.Error("starship shared content does not contain STARSHIP_CONFIG")
	}
	if !strings.Contains(s, ".starship.toml") {
		t.Error("starship shared content does not reference .starship.toml")
	}
}

func TestOtelConfigSharedContent(t *testing.T) {
	reg := DefaultRegistry()
	tool, ok := reg.ByName("otel-config")
	if !ok {
		t.Fatal("otel-config not found in registry")
	}

	if tool.SharedContent == nil {
		t.Fatal("SharedContent map is nil")
	}

	fn, ok := tool.SharedContent["otel-config"]
	if !ok {
		t.Fatal("SharedContent missing 'otel-config' key")
	}

	answers := types.WizardAnswers{
		ProjectName: "testapp",
	}
	content, err := fn(answers)
	if err != nil {
		t.Fatalf("SharedContent['otel-config'] returned error: %v", err)
	}

	s := string(content)
	if !strings.Contains(s, "OTEL_EXPORTER_OTLP_ENDPOINT") {
		t.Error("otel shared content does not contain OTEL_EXPORTER_OTLP_ENDPOINT")
	}
	if !strings.Contains(s, "OTEL_SERVICE_NAME") {
		t.Error("otel shared content does not contain OTEL_SERVICE_NAME")
	}
	if !strings.Contains(s, "testapp") {
		t.Error("otel shared content does not contain project name 'testapp'")
	}
}

func TestOtelConfigSharedContent_DefaultEndpoint(t *testing.T) {
	fn := otelConfigNixContent

	content, err := fn(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := string(content)
	if !strings.Contains(s, "http://localhost:4317") {
		t.Error("default endpoint should be http://localhost:4317")
	}
	if !strings.Contains(s, `"unknown"`) {
		t.Error("empty project name should default to 'unknown'")
	}
}

func TestOtelConfigSharedContent_CustomEndpoint(t *testing.T) {
	fn := otelConfigNixContent

	answers := types.WizardAnswers{
		ProjectName: "myservice",
		EnvVars: map[string]string{
			"OTEL_EXPORTER_OTLP_ENDPOINT": "http://collector:4317",
		},
	}
	content, err := fn(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := string(content)
	if !strings.Contains(s, "http://collector:4317") {
		t.Error("custom endpoint should be used when provided in EnvVars")
	}
	if !strings.Contains(s, `"myservice"`) {
		t.Error("project name should appear in OTEL_SERVICE_NAME")
	}
}

func TestStarshipGenerateFunc(t *testing.T) {
	reg := DefaultRegistry()
	tool, ok := reg.ByName("starship-integration")
	if !ok {
		t.Fatal("starship-integration not found in registry")
	}

	if tool.GenerateFunc == nil {
		t.Fatal("GenerateFunc is nil for starship-integration")
	}

	files, err := tool.GenerateFunc(types.WizardAnswers{ProjectName: "test"})
	if err != nil {
		t.Fatalf("GenerateFunc returned error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 generated file, got %d", len(files))
	}

	if files[0].Path != ".starship.toml" {
		t.Errorf("generated file path = %q, want %q", files[0].Path, ".starship.toml")
	}
	if len(files[0].Content) == 0 {
		t.Error("generated file content is empty")
	}
}

func TestOtelConfigGenerateFunc_Nil(t *testing.T) {
	reg := DefaultRegistry()
	tool, ok := reg.ByName("otel-config")
	if !ok {
		t.Fatal("otel-config not found in registry")
	}

	if tool.GenerateFunc != nil {
		t.Error("otel-config should have nil GenerateFunc (no exclusive files)")
	}
}
