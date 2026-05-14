package check

import (
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestCheckRequiredTools_NoneDisabled(t *testing.T) {
	ctx := CheckContext{
		GdevConfig: &types.GdevConfig{},
		ToolNames:  []string{"safety-block", "pre-commit"},
	}

	results := CheckRequiredTools(ctx)

	for _, r := range results {
		if r.Status == StatusFail {
			t.Errorf("unexpected failure: %s: %s", r.Name, r.Message)
		}
	}

	hasPass := false
	for _, r := range results {
		if r.Status == StatusPass {
			hasPass = true
			break
		}
	}
	if !hasPass {
		t.Error("expected a passing result when no required tools are disabled")
	}
}

func TestCheckRequiredTools_ToolDisabled(t *testing.T) {
	ctx := CheckContext{
		GdevConfig: &types.GdevConfig{
			Tools: types.ToolsConfig{
				Disabled: []string{"safety-block"},
			},
		},
		ToolNames: []string{"safety-block", "pre-commit"},
	}

	results := CheckRequiredTools(ctx)

	hasFail := false
	for _, r := range results {
		if r.Status == StatusFail && r.Severity == SeverityHigh {
			hasFail = true
			break
		}
	}
	if !hasFail {
		t.Error("expected a high-severity failure when a required tool is disabled")
	}
}

func TestCheckRequiredTools_NoConfig(t *testing.T) {
	ctx := CheckContext{
		ToolNames: []string{"safety-block"},
	}

	results := CheckRequiredTools(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusSkip {
		t.Errorf("Status = %s, want %s", results[0].Status, StatusSkip)
	}
}

func TestCheckRequiredTools_NoTools(t *testing.T) {
	ctx := CheckContext{
		GdevConfig: &types.GdevConfig{},
		ToolNames:  nil,
	}

	results := CheckRequiredTools(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusSkip {
		t.Errorf("Status = %s, want %s", results[0].Status, StatusSkip)
	}
}
