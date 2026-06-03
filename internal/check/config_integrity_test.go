package check

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestCheckConfigIntegrity_ValidConfig(t *testing.T) {
	ctx := CheckContext{
		QsdevConfig: &types.QsdevConfig{
			Version: 1,
			Languages: []types.LanguageConfig{
				{Name: "go", Version: "1.24"},
			},
			Services: []types.ServiceConfig{
				{Name: "postgres"},
			},
			Profile: "go-web",
		},
		ProfileNames: []string{"go-web", "ts-fullstack"},
		ToolNames:    []string{"safety-block", "pre-commit"},
	}

	results := CheckConfigIntegrity(ctx)

	// Should have a passing config_exists check.
	var configExists *CheckResult
	for i := range results {
		if results[i].Name == "config_exists" {
			configExists = &results[i]
			break
		}
	}

	if configExists == nil {
		t.Fatal("expected config_exists result")
		return
	}
	if configExists.Status != StatusPass {
		t.Errorf("config_exists.Status = %s, want %s", configExists.Status, StatusPass)
	}

	// No validation failures expected.
	for _, r := range results {
		if r.Status == StatusFail {
			t.Errorf("unexpected failure: %s: %s", r.Name, r.Message)
		}
	}
}

func TestCheckConfigIntegrity_InvalidLanguage(t *testing.T) {
	ctx := CheckContext{
		QsdevConfig: &types.QsdevConfig{
			Version: 1,
			Languages: []types.LanguageConfig{
				{Name: "cobol"},
			},
		},
	}

	results := CheckConfigIntegrity(ctx)

	hasFail := false
	for _, r := range results {
		if r.Status == StatusFail && r.Name == "config_validation" {
			hasFail = true
			break
		}
	}

	if !hasFail {
		t.Error("expected a config_validation failure for unknown language")
	}
}

func TestCheckConfigIntegrity_NoConfig(t *testing.T) {
	ctx := CheckContext{}

	results := CheckConfigIntegrity(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusFail {
		t.Errorf("Status = %s, want %s", results[0].Status, StatusFail)
	}
	if results[0].Severity != SeverityCritical {
		t.Errorf("Severity = %s, want %s", results[0].Severity, SeverityCritical)
	}
}

func TestCheckConfigIntegrity_InvalidProfile(t *testing.T) {
	ctx := CheckContext{
		QsdevConfig: &types.QsdevConfig{
			Version: 1,
			Profile: "unknown-profile",
		},
		ProfileNames: []string{"go-web", "ts-fullstack"},
	}

	results := CheckConfigIntegrity(ctx)

	hasFail := false
	for _, r := range results {
		if r.Status == StatusFail {
			hasFail = true
			break
		}
	}

	if !hasFail {
		t.Error("expected a validation failure for unknown profile")
	}
}

func TestCheckConfigIntegrity_InvalidService(t *testing.T) {
	ctx := CheckContext{
		QsdevConfig: &types.QsdevConfig{
			Version: 1,
			Services: []types.ServiceConfig{
				{Name: "oracle"},
			},
		},
	}

	results := CheckConfigIntegrity(ctx)

	hasFail := false
	for _, r := range results {
		if r.Status == StatusFail {
			hasFail = true
			break
		}
	}

	if !hasFail {
		t.Error("expected a validation failure for unknown service")
	}
}
