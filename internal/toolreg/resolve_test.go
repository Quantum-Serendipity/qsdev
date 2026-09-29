package toolreg

import (
	"errors"
	"strings"
	"testing"
)

func newTestRegistry(tools ...Tool) *Registry {
	reg := NewRegistry()
	for _, t := range tools {
		if err := reg.Register(t); err != nil {
			panic("test setup: " + err.Error())
		}
	}
	return reg
}

// --- ValidateEnable tests ---

func TestValidateEnable_Success(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "base", Category: CategorySecurity},
		Tool{Name: "addon", Category: CategorySecurity, Prerequisites: []string{"base"}},
	)
	enabled := map[string]bool{"base": true}

	err := ValidateEnable(reg, "addon", enabled)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateEnable_NoPrerequisites(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "standalone", Category: CategoryDevEx},
	)

	err := ValidateEnable(reg, "standalone", map[string]bool{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateEnable_MissingPrerequisite(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "base", Category: CategorySecurity},
		Tool{Name: "addon", Category: CategorySecurity, Prerequisites: []string{"base"}},
	)
	enabled := map[string]bool{} // base is not enabled

	err := ValidateEnable(reg, "addon", enabled)
	if err == nil {
		t.Fatal("expected error for missing prerequisite, got nil")
	}
	if !strings.Contains(err.Error(), "prerequisite") {
		t.Fatalf("expected error mentioning prerequisite, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), `"base"`) {
		t.Fatalf("expected error mentioning base, got %q", err.Error())
	}
}

func TestValidateEnable_MultiplePrerequisites_OneMissing(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "dep-a", Category: CategorySecurity},
		Tool{Name: "dep-b", Category: CategorySecurity},
		Tool{Name: "child", Category: CategorySecurity, Prerequisites: []string{"dep-a", "dep-b"}},
	)
	enabled := map[string]bool{"dep-a": true} // dep-b missing

	err := ValidateEnable(reg, "child", enabled)
	if err == nil {
		t.Fatal("expected error for missing prerequisite, got nil")
	}
	if !strings.Contains(err.Error(), `"dep-b"`) {
		t.Fatalf("expected error mentioning dep-b, got %q", err.Error())
	}
}

func TestValidateEnable_Conflict(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "tool-a", Category: CategorySecurity},
		Tool{Name: "tool-b", Category: CategorySecurity, Conflicts: []string{"tool-a"}},
	)
	enabled := map[string]bool{"tool-a": true}

	err := ValidateEnable(reg, "tool-b", enabled)
	if err == nil {
		t.Fatal("expected error for conflict, got nil")
	}
	if !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("expected error mentioning conflicts, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), `"tool-a"`) {
		t.Fatalf("expected error mentioning tool-a, got %q", err.Error())
	}
}

// TestValidateEnable_ConflictDeclaredByEnabledTool pins F480: a conflict
// declared only on the already-enabled tool must still block the enable.
func TestValidateEnable_ConflictDeclaredByEnabledTool(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry(
		Tool{Name: "semgrep", Category: CategorySecurity, Conflicts: []string{"opengrep"}},
		Tool{Name: "opengrep", Category: CategorySecurity},
	)

	tests := []struct {
		name    string
		enabled map[string]bool
		wantErr bool
	}{
		{name: "declaring tool enabled", enabled: map[string]bool{"semgrep": true}, wantErr: true},
		{name: "declaring tool disabled", enabled: map[string]bool{"semgrep": false}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateEnable(reg, "opengrep", tt.enabled)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateEnable() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), `conflicts with enabled tool "semgrep"`) {
				t.Errorf("error = %q, want it to name semgrep", err.Error())
			}
		})
	}
}

func TestValidateEnable_ConflictNotEnabled(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "tool-a", Category: CategorySecurity},
		Tool{Name: "tool-b", Category: CategorySecurity, Conflicts: []string{"tool-a"}},
	)
	enabled := map[string]bool{} // tool-a not enabled, so no conflict

	err := ValidateEnable(reg, "tool-b", enabled)
	if err != nil {
		t.Fatalf("expected no error (conflict not enabled), got %v", err)
	}
}

func TestValidateEnable_UnknownTool(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "known", Category: CategorySecurity},
	)

	err := ValidateEnable(reg, "nonexistent", map[string]bool{})
	if err == nil {
		t.Fatal("expected error for unknown tool, got nil")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Fatalf("expected error mentioning unknown tool, got %q", err.Error())
	}
}

// --- ValidateDisable tests ---

func TestValidateDisable_Success(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "standalone", Category: CategorySecurity, Default: OptIn},
	)
	enabled := map[string]bool{"standalone": true}

	err := ValidateDisable(reg, "standalone", enabled)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateDisable_NoDependents(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "base", Category: CategorySecurity, Default: OptIn},
		Tool{Name: "unrelated", Category: CategoryDevEx, Default: OptIn},
	)
	enabled := map[string]bool{"base": true, "unrelated": true}

	err := ValidateDisable(reg, "base", enabled)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateDisable_DependentEnabled(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "base", Category: CategorySecurity, Default: OptIn},
		Tool{Name: "addon", Category: CategorySecurity, Default: OptIn, Prerequisites: []string{"base"}},
	)
	enabled := map[string]bool{"base": true, "addon": true}

	err := ValidateDisable(reg, "base", enabled)
	if err == nil {
		t.Fatal("expected error for dependent tool, got nil")
	}
	if !strings.Contains(err.Error(), "required by") {
		t.Fatalf("expected error mentioning 'required by', got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "addon") {
		t.Fatalf("expected error mentioning addon, got %q", err.Error())
	}
}

func TestValidateDisable_DependentNotEnabled(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "base", Category: CategorySecurity, Default: OptIn},
		Tool{Name: "addon", Category: CategorySecurity, Default: OptIn, Prerequisites: []string{"base"}},
	)
	// addon exists but is not enabled, so disabling base should be fine.
	enabled := map[string]bool{"base": true}

	err := ValidateDisable(reg, "base", enabled)
	if err != nil {
		t.Fatalf("expected no error (dependent not enabled), got %v", err)
	}
}

func TestValidateDisable_MultipleDependents(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "base", Category: CategorySecurity, Default: OptIn},
		Tool{Name: "child-a", Category: CategorySecurity, Default: OptIn, Prerequisites: []string{"base"}},
		Tool{Name: "child-b", Category: CategoryDevEx, Default: OptIn, Prerequisites: []string{"base"}},
	)
	enabled := map[string]bool{"base": true, "child-a": true, "child-b": true}

	err := ValidateDisable(reg, "base", enabled)
	if err == nil {
		t.Fatal("expected error for multiple dependents, got nil")
	}
	if !strings.Contains(err.Error(), "child-a") {
		t.Fatalf("expected error mentioning child-a, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "child-b") {
		t.Fatalf("expected error mentioning child-b, got %q", err.Error())
	}
}

func TestValidateDisable_AlwaysOn(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "always-tool", Category: CategorySecurity, Default: AlwaysOn},
	)
	enabled := map[string]bool{"always-tool": true}

	err := ValidateDisable(reg, "always-tool", enabled)
	if err == nil {
		t.Fatal("expected error for always-on tool, got nil")
	}
	var alwaysOnErr *AlwaysOnError
	if !errors.As(err, &alwaysOnErr) {
		t.Fatalf("expected AlwaysOnError, got %T: %v", err, err)
	}
	if alwaysOnErr.ToolName != "always-tool" {
		t.Fatalf("expected tool name %q, got %q", "always-tool", alwaysOnErr.ToolName)
	}
}

func TestValidateDisable_UnknownTool(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "known", Category: CategorySecurity},
	)

	err := ValidateDisable(reg, "nonexistent", map[string]bool{})
	if err == nil {
		t.Fatal("expected error for unknown tool, got nil")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Fatalf("expected error mentioning unknown tool, got %q", err.Error())
	}
}
