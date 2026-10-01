package check

import (
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestCheckRequiredTools_NoneDisabled(t *testing.T) {
	ctx := CheckContext{
		QsdevConfig: &types.QsdevConfig{
			Tools: types.ToolsConfig{Enabled: []string{"safety-block", "pre-commit"}},
		},
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
				Enabled:  []string{"pre-commit"},
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
		enabled  []string
		disabled []string
		wantFail []string
	}{
		{name: "opt-in tool disabled", enabled: []string{"safety-block"}, disabled: []string{"semgrep"}},
		{name: "always-on tool disabled", disabled: []string{"safety-block"}, wantFail: []string{"tool_not_disabled_safety-block"}},
		{name: "mixed", disabled: []string{"semgrep", "safety-block"}, wantFail: []string{"tool_not_disabled_safety-block"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := CheckContext{
				QsdevConfig:       &types.QsdevConfig{Tools: types.ToolsConfig{Enabled: tt.enabled, Disabled: tt.disabled}},
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

// TestCheckRequiredTools_FailsWhenAlwaysOnAbsent is the U28-01 regression: an
// always-on tool recorded in neither tools.enabled nor tools.disabled was
// dropped without the explicit `disable --force` opt-out, so the check fails.
func TestCheckRequiredTools_FailsWhenAlwaysOnAbsent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tools    types.ToolsConfig
		wantFail []string
	}{
		{name: "recorded as enabled", tools: types.ToolsConfig{Enabled: []string{"attach-guard", "branch-naming"}}},
		{name: "one absent", tools: types.ToolsConfig{Enabled: []string{"branch-naming"}}, wantFail: []string{"tool_missing_attach-guard"}},
		{name: "no tools block", wantFail: []string{"tool_missing_attach-guard", "tool_missing_branch-naming"}},
		{
			name:     "explicitly disabled is not missing",
			tools:    types.ToolsConfig{Enabled: []string{"branch-naming"}, Disabled: []string{"attach-guard"}},
			wantFail: []string{"tool_not_disabled_attach-guard"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := CheckContext{
				QsdevConfig:       &types.QsdevConfig{Tools: tt.tools},
				AlwaysOnToolNames: []string{"attach-guard", "branch-naming"},
			}

			var failed []string
			for _, r := range CheckRequiredTools(ctx) {
				if r.Status != StatusFail {
					continue
				}
				failed = append(failed, r.Name)
				if r.Severity != SeverityHigh {
					t.Errorf("%s severity = %s, want %s", r.Name, r.Severity, SeverityHigh)
				}
				if strings.HasPrefix(r.Name, "tool_missing_") {
					for _, want := range []string{"qsdev update", "disable " + strings.TrimPrefix(r.Name, "tool_missing_") + " --force"} {
						if !strings.Contains(r.Remediation, want) {
							t.Errorf("%s remediation %q lacks %q", r.Name, r.Remediation, want)
						}
					}
				}
			}
			if !slices.Equal(failed, tt.wantFail) {
				t.Fatalf("failed = %v, want %v", failed, tt.wantFail)
			}
		})
	}
}
