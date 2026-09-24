package check

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestCheckRequiredTools_NoneDisabled(t *testing.T) {
	ctx := CheckContext{
		QsdevConfig:       &types.QsdevConfig{},
		ToolNames:         []string{"safety-block", "pre-commit"},
		AlwaysOnToolNames: []string{"safety-block", "pre-commit"},
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
		QsdevConfig: &types.QsdevConfig{
			Tools: types.ToolsConfig{
				Disabled: []string{"safety-block"},
			},
		},
		ToolNames:         []string{"safety-block", "pre-commit"},
		AlwaysOnToolNames: []string{"safety-block", "pre-commit"},
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

// TestCheckRequiredTools_OnlyAlwaysOnToolsAreRequired verifies that disabling
// an opt-in or detected tool is legitimate and does not fail the check, while
// disabling an always-on tool still does.
func TestCheckRequiredTools_OnlyAlwaysOnToolsAreRequired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		disabled []string
		wantFail []string
	}{
		{name: "opt-in tool disabled", disabled: []string{"semgrep"}},
		{name: "always-on tool disabled", disabled: []string{"safety-block"}, wantFail: []string{"tool_not_disabled_safety-block"}},
		{name: "mixed", disabled: []string{"semgrep", "safety-block"}, wantFail: []string{"tool_not_disabled_safety-block"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := CheckContext{
				QsdevConfig:       &types.QsdevConfig{Tools: types.ToolsConfig{Disabled: tt.disabled}},
				ToolNames:         []string{"safety-block", "semgrep"},
				AlwaysOnToolNames: []string{"safety-block"},
			}

			var failed []string
			for _, r := range CheckRequiredTools(ctx) {
				if r.Status == StatusFail {
					failed = append(failed, r.Name)
				}
			}
			if len(failed) != len(tt.wantFail) {
				t.Fatalf("failed = %v, want %v", failed, tt.wantFail)
			}
			for i := range failed {
				if failed[i] != tt.wantFail[i] {
					t.Fatalf("failed = %v, want %v", failed, tt.wantFail)
				}
			}
		})
	}
}

func TestCheckRequiredTools_NoConfig(t *testing.T) {
	ctx := CheckContext{
		AlwaysOnToolNames: []string{"safety-block"},
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
		QsdevConfig:       &types.QsdevConfig{},
		AlwaysOnToolNames: nil,
	}

	results := CheckRequiredTools(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusSkip {
		t.Errorf("Status = %s, want %s", results[0].Status, StatusSkip)
	}
}
