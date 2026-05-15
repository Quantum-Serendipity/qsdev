package toolreg

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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
		Tool{Name: "standalone", Category: CategorySecurity},
	)
	enabled := map[string]bool{"standalone": true}

	err := ValidateDisable(reg, "standalone", enabled)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateDisable_NoDependents(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "base", Category: CategorySecurity},
		Tool{Name: "unrelated", Category: CategoryDevEx},
	)
	enabled := map[string]bool{"base": true, "unrelated": true}

	err := ValidateDisable(reg, "base", enabled)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateDisable_DependentEnabled(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "base", Category: CategorySecurity},
		Tool{Name: "addon", Category: CategorySecurity, Prerequisites: []string{"base"}},
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
		Tool{Name: "base", Category: CategorySecurity},
		Tool{Name: "addon", Category: CategorySecurity, Prerequisites: []string{"base"}},
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
		Tool{Name: "base", Category: CategorySecurity},
		Tool{Name: "child-a", Category: CategorySecurity, Prerequisites: []string{"base"}},
		Tool{Name: "child-b", Category: CategoryDevEx, Prerequisites: []string{"base"}},
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

// --- ComputeDefaults tests ---

func TestComputeDefaults_AlwaysOn(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "always-tool", Category: CategorySecurity, Default: AlwaysOn},
	)

	defaults := ComputeDefaults(reg, types.DetectedProject{})
	if !defaults["always-tool"] {
		t.Fatal("expected always-on tool to be enabled")
	}
}

func TestComputeDefaults_OnWhenDetected_Detected(t *testing.T) {
	reg := newTestRegistry(
		Tool{
			Name:     "go-tool",
			Category: CategoryDevEx,
			Default:  OnWhenDetected,
			DetectFunc: func(d types.DetectedProject) bool {
				return d.HasGoMod
			},
		},
	)

	detected := types.DetectedProject{HasGoMod: true}
	defaults := ComputeDefaults(reg, detected)
	if !defaults["go-tool"] {
		t.Fatal("expected detected tool to be enabled")
	}
}

func TestComputeDefaults_OnWhenDetected_NotDetected(t *testing.T) {
	reg := newTestRegistry(
		Tool{
			Name:     "go-tool",
			Category: CategoryDevEx,
			Default:  OnWhenDetected,
			DetectFunc: func(d types.DetectedProject) bool {
				return d.HasGoMod
			},
		},
	)

	detected := types.DetectedProject{HasGoMod: false}
	defaults := ComputeDefaults(reg, detected)
	if defaults["go-tool"] {
		t.Fatal("expected non-detected tool to not be enabled")
	}
}

func TestComputeDefaults_OnWhenDetected_NilDetectFunc(t *testing.T) {
	reg := newTestRegistry(
		Tool{
			Name:       "missing-detect",
			Category:   CategoryDevEx,
			Default:    OnWhenDetected,
			DetectFunc: nil,
		},
	)

	defaults := ComputeDefaults(reg, types.DetectedProject{HasGoMod: true})
	if defaults["missing-detect"] {
		t.Fatal("expected tool with nil DetectFunc to not be enabled")
	}
}

func TestComputeDefaults_OptIn(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "optional-tool", Category: CategoryDevEx, Default: OptIn},
	)

	defaults := ComputeDefaults(reg, types.DetectedProject{})
	if defaults["optional-tool"] {
		t.Fatal("expected opt-in tool to not be enabled by default")
	}
}

func TestComputeDefaults_AlwaysOff(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "deprecated-tool", Category: CategoryDevEx, Default: AlwaysOff},
	)

	defaults := ComputeDefaults(reg, types.DetectedProject{})
	if defaults["deprecated-tool"] {
		t.Fatal("expected always-off tool to not be enabled")
	}
}

func TestComputeDefaults_MixedPolicies(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "always", Category: CategorySecurity, Default: AlwaysOn},
		Tool{
			Name: "detected", Category: CategoryDevEx, Default: OnWhenDetected,
			DetectFunc: func(d types.DetectedProject) bool { return d.HasPackageJSON },
		},
		Tool{
			Name: "not-detected", Category: CategoryDevEx, Default: OnWhenDetected,
			DetectFunc: func(d types.DetectedProject) bool { return d.HasCargoToml },
		},
		Tool{Name: "opt-in", Category: CategoryInfrastructure, Default: OptIn},
		Tool{Name: "off", Category: CategoryInfrastructure, Default: AlwaysOff},
	)

	detected := types.DetectedProject{HasPackageJSON: true, HasCargoToml: false}
	defaults := ComputeDefaults(reg, detected)

	if !defaults["always"] {
		t.Error("expected 'always' to be enabled")
	}
	if !defaults["detected"] {
		t.Error("expected 'detected' to be enabled")
	}
	if defaults["not-detected"] {
		t.Error("expected 'not-detected' to not be enabled")
	}
	if defaults["opt-in"] {
		t.Error("expected 'opt-in' to not be enabled")
	}
	if defaults["off"] {
		t.Error("expected 'off' to not be enabled")
	}

	// Exactly 2 tools should be enabled.
	count := 0
	for _, v := range defaults {
		if v {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected 2 enabled tools, got %d", count)
	}
}
