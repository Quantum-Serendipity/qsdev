package posture

import (
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestEvaluateConformance_BaselinePass(t *testing.T) {
	defense := DefenseCoverage{
		Layers: []DefenseLayer{
			{Name: "pretooluse-hooks", Weight: WeightCritical, Status: LayerEnabled},
			{Name: "age-gating", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "install-script-blocking", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "lock-file-enforcement", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "vulnerability-scanning", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "nix-hardening", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "sast", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "secrets-scanning", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "container-security", Weight: WeightMedium, Status: LayerNotApplicable},
			{Name: "license-compliance", Weight: WeightLow, Status: LayerEnabled},
		},
		Score: 100.0,
	}
	deps := DependencyHealth{
		Ecosystems: []EcosystemStatus{
			{Name: "go", Detected: true, LockFile: "valid"},
		},
		Totals: VulnSeverityCounts{},
		Score:  100.0,
	}
	enabledTools := map[string]bool{
		"attach-guard":       true,
		"semgrep":            true,
		"gitleaks":           true,
		"license-compliance": true,
	}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"CLAUDE.md":                   {},
			".claude/settings.json":       {},
			".pre-commit-config.yaml":     {},
		},
	}

	result := EvaluateConformance(defense, deps, enabledTools, genState)

	if !result.Baseline.Pass {
		t.Error("expected baseline to pass")
		for _, c := range result.Baseline.Checks {
			if !c.Pass {
				t.Errorf("  failed: %s — %s", c.Name, c.Reason)
			}
		}
	}
}

func TestEvaluateConformance_BaselineFail_NoCLAUDEMD(t *testing.T) {
	defense := DefenseCoverage{
		Layers: []DefenseLayer{
			{Name: "pretooluse-hooks", Weight: WeightCritical, Status: LayerEnabled},
			{Name: "age-gating", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "install-script-blocking", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "lock-file-enforcement", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "vulnerability-scanning", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "nix-hardening", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "sast", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "secrets-scanning", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "container-security", Weight: WeightMedium, Status: LayerNotApplicable},
			{Name: "license-compliance", Weight: WeightLow, Status: LayerEnabled},
		},
	}
	deps := DependencyHealth{
		Ecosystems: []EcosystemStatus{{Name: "go", Detected: true, LockFile: "valid"}},
	}
	enabledTools := map[string]bool{}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			// No CLAUDE.md
			".claude/settings.json":   {},
			".pre-commit-config.yaml": {},
		},
	}

	result := EvaluateConformance(defense, deps, enabledTools, genState)

	if result.Baseline.Pass {
		t.Error("expected baseline to fail without CLAUDE.md")
	}

	found := false
	for _, c := range result.Baseline.Checks {
		if c.Name == "claude-md-present" {
			found = true
			if c.Pass {
				t.Error("claude-md-present should have failed")
			}
		}
	}
	if !found {
		t.Error("claude-md-present check not found")
	}
}

func TestEvaluateConformance_BaselineFail_CriticalVulns(t *testing.T) {
	defense := DefenseCoverage{
		Layers: []DefenseLayer{
			{Name: "pretooluse-hooks", Weight: WeightCritical, Status: LayerEnabled},
			{Name: "age-gating", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "install-script-blocking", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "lock-file-enforcement", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "vulnerability-scanning", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "nix-hardening", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "sast", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "secrets-scanning", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "container-security", Weight: WeightMedium, Status: LayerNotApplicable},
			{Name: "license-compliance", Weight: WeightLow, Status: LayerEnabled},
		},
	}
	deps := DependencyHealth{
		Ecosystems: []EcosystemStatus{{Name: "go", Detected: true, LockFile: "valid"}},
		Totals:     VulnSeverityCounts{Critical: 2},
	}
	enabledTools := map[string]bool{}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"CLAUDE.md":                {},
			".claude/settings.json":    {},
			".pre-commit-config.yaml":  {},
		},
	}

	result := EvaluateConformance(defense, deps, enabledTools, genState)

	if result.Baseline.Pass {
		t.Error("expected baseline to fail with critical vulns")
	}
}

func TestEvaluateConformance_BaselineFail_MissingLockFile(t *testing.T) {
	defense := DefenseCoverage{
		Layers: []DefenseLayer{
			{Name: "pretooluse-hooks", Weight: WeightCritical, Status: LayerEnabled},
			{Name: "age-gating", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "install-script-blocking", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "lock-file-enforcement", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "vulnerability-scanning", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "nix-hardening", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "sast", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "secrets-scanning", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "container-security", Weight: WeightMedium, Status: LayerNotApplicable},
			{Name: "license-compliance", Weight: WeightLow, Status: LayerEnabled},
		},
	}
	deps := DependencyHealth{
		Ecosystems: []EcosystemStatus{
			{Name: "go", Detected: true, LockFile: "missing"},
		},
	}
	enabledTools := map[string]bool{}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"CLAUDE.md":                {},
			".claude/settings.json":    {},
			".pre-commit-config.yaml":  {},
		},
	}

	result := EvaluateConformance(defense, deps, enabledTools, genState)

	if result.Baseline.Pass {
		t.Error("expected baseline to fail without lock files")
	}
}

func TestEvaluateConformance_BaselineFail_HighLayerDisabled(t *testing.T) {
	defense := DefenseCoverage{
		Layers: []DefenseLayer{
			{Name: "pretooluse-hooks", Weight: WeightCritical, Status: LayerDisabled},
			{Name: "age-gating", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "install-script-blocking", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "lock-file-enforcement", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "vulnerability-scanning", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "nix-hardening", Weight: WeightMedium, Status: LayerDisabled},
			{Name: "sast", Weight: WeightMedium, Status: LayerDisabled},
			{Name: "secrets-scanning", Weight: WeightMedium, Status: LayerDisabled},
			{Name: "container-security", Weight: WeightMedium, Status: LayerNotApplicable},
			{Name: "license-compliance", Weight: WeightLow, Status: LayerDisabled},
		},
	}
	deps := DependencyHealth{
		Ecosystems: []EcosystemStatus{{Name: "go", Detected: true, LockFile: "valid"}},
	}
	enabledTools := map[string]bool{}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"CLAUDE.md":                {},
			".claude/settings.json":    {},
			".pre-commit-config.yaml":  {},
		},
	}

	result := EvaluateConformance(defense, deps, enabledTools, genState)

	if result.Baseline.Pass {
		t.Error("expected baseline to fail with critical layer disabled")
	}
}

func TestEvaluateConformance_EnhancedPass(t *testing.T) {
	defense := DefenseCoverage{
		Layers: []DefenseLayer{
			{Name: "pretooluse-hooks", Weight: WeightCritical, Status: LayerEnabled},
			{Name: "age-gating", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "install-script-blocking", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "lock-file-enforcement", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "vulnerability-scanning", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "nix-hardening", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "sast", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "secrets-scanning", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "container-security", Weight: WeightMedium, Status: LayerNotApplicable},
			{Name: "license-compliance", Weight: WeightLow, Status: LayerEnabled},
		},
	}
	deps := DependencyHealth{
		Ecosystems: []EcosystemStatus{{Name: "go", Detected: true, LockFile: "valid"}},
		Totals:     VulnSeverityCounts{},
	}
	enabledTools := map[string]bool{
		"semgrep":            true,
		"gitleaks":           true,
		"license-compliance": true,
	}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"CLAUDE.md":                {},
			".claude/settings.json":    {},
			".pre-commit-config.yaml":  {},
		},
	}

	result := EvaluateConformance(defense, deps, enabledTools, genState)

	if !result.Baseline.Pass {
		t.Error("expected baseline to pass")
		for _, c := range result.Baseline.Checks {
			if !c.Pass {
				t.Errorf("  baseline failed: %s — %s", c.Name, c.Reason)
			}
		}
	}
	if !result.Enhanced.Pass {
		t.Error("expected enhanced to pass")
		for _, c := range result.Enhanced.Checks {
			if !c.Pass {
				t.Errorf("  enhanced failed: %s — %s", c.Name, c.Reason)
			}
		}
	}
}

func TestEvaluateConformance_EnhancedFail_HighVulns(t *testing.T) {
	defense := DefenseCoverage{
		Layers: []DefenseLayer{
			{Name: "pretooluse-hooks", Weight: WeightCritical, Status: LayerEnabled},
			{Name: "age-gating", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "install-script-blocking", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "lock-file-enforcement", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "vulnerability-scanning", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "nix-hardening", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "sast", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "secrets-scanning", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "container-security", Weight: WeightMedium, Status: LayerNotApplicable},
			{Name: "license-compliance", Weight: WeightLow, Status: LayerEnabled},
		},
	}
	deps := DependencyHealth{
		Ecosystems: []EcosystemStatus{{Name: "go", Detected: true, LockFile: "valid"}},
		Totals:     VulnSeverityCounts{High: 3},
	}
	enabledTools := map[string]bool{
		"semgrep":            true,
		"gitleaks":           true,
		"license-compliance": true,
	}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"CLAUDE.md":                {},
			".claude/settings.json":    {},
			".pre-commit-config.yaml":  {},
		},
	}

	result := EvaluateConformance(defense, deps, enabledTools, genState)

	if !result.Baseline.Pass {
		t.Error("expected baseline to pass (high vulns don't fail baseline)")
	}
	if result.Enhanced.Pass {
		t.Error("expected enhanced to fail with high vulns")
	}
}

func TestEvaluateConformance_EnhancedFail_NoSemgrep(t *testing.T) {
	defense := DefenseCoverage{
		Layers: []DefenseLayer{
			{Name: "pretooluse-hooks", Weight: WeightCritical, Status: LayerEnabled},
			{Name: "age-gating", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "install-script-blocking", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "lock-file-enforcement", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "vulnerability-scanning", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "nix-hardening", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "sast", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "secrets-scanning", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "container-security", Weight: WeightMedium, Status: LayerNotApplicable},
			{Name: "license-compliance", Weight: WeightLow, Status: LayerEnabled},
		},
	}
	deps := DependencyHealth{
		Ecosystems: []EcosystemStatus{{Name: "go", Detected: true, LockFile: "valid"}},
	}
	enabledTools := map[string]bool{
		// No semgrep
		"gitleaks":           true,
		"license-compliance": true,
	}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"CLAUDE.md":                {},
			".claude/settings.json":    {},
			".pre-commit-config.yaml":  {},
		},
	}

	result := EvaluateConformance(defense, deps, enabledTools, genState)

	if result.Enhanced.Pass {
		t.Error("expected enhanced to fail without semgrep")
	}
}

func TestEvaluateConformance_EnhancedRequiresBaseline(t *testing.T) {
	// If baseline fails, enhanced should also fail even if all enhanced checks pass
	defense := DefenseCoverage{
		Layers: []DefenseLayer{
			{Name: "pretooluse-hooks", Weight: WeightCritical, Status: LayerDisabled},
			{Name: "age-gating", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "install-script-blocking", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "lock-file-enforcement", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "vulnerability-scanning", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "nix-hardening", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "sast", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "secrets-scanning", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "container-security", Weight: WeightMedium, Status: LayerNotApplicable},
			{Name: "license-compliance", Weight: WeightLow, Status: LayerEnabled},
		},
	}
	deps := DependencyHealth{
		Ecosystems: []EcosystemStatus{{Name: "go", Detected: true, LockFile: "valid"}},
	}
	enabledTools := map[string]bool{
		"semgrep":            true,
		"gitleaks":           true,
		"license-compliance": true,
	}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"CLAUDE.md":                {},
			".claude/settings.json":    {},
			".pre-commit-config.yaml":  {},
		},
	}

	result := EvaluateConformance(defense, deps, enabledTools, genState)

	if result.Baseline.Pass {
		t.Error("expected baseline to fail (critical layer disabled)")
	}
	if result.Enhanced.Pass {
		t.Error("expected enhanced to fail when baseline fails")
	}
}

func TestEvaluateConformance_BaselineCheckCount(t *testing.T) {
	defense := DefenseCoverage{
		Layers: []DefenseLayer{
			{Name: "pretooluse-hooks", Weight: WeightCritical, Status: LayerEnabled},
		},
	}
	deps := DependencyHealth{}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{},
	}

	result := EvaluateConformance(defense, deps, map[string]bool{}, genState)

	if len(result.Baseline.Checks) != 6 {
		t.Errorf("baseline check count: got %d, want 6", len(result.Baseline.Checks))
	}
	if len(result.Enhanced.Checks) != 5 {
		t.Errorf("enhanced check count: got %d, want 5", len(result.Enhanced.Checks))
	}
}

func TestEvaluateConformance_BaselineFail_NoPreCommit(t *testing.T) {
	defense := DefenseCoverage{
		Layers: []DefenseLayer{
			{Name: "pretooluse-hooks", Weight: WeightCritical, Status: LayerEnabled},
			{Name: "age-gating", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "install-script-blocking", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "lock-file-enforcement", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "vulnerability-scanning", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "nix-hardening", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "sast", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "secrets-scanning", Weight: WeightMedium, Status: LayerEnabled},
			{Name: "container-security", Weight: WeightMedium, Status: LayerNotApplicable},
			{Name: "license-compliance", Weight: WeightLow, Status: LayerEnabled},
		},
	}
	deps := DependencyHealth{
		Ecosystems: []EcosystemStatus{{Name: "go", Detected: true, LockFile: "valid"}},
	}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"CLAUDE.md":             {},
			".claude/settings.json": {},
			// No pre-commit config
		},
	}

	result := EvaluateConformance(defense, deps, map[string]bool{}, genState)

	if result.Baseline.Pass {
		t.Error("expected baseline to fail without pre-commit hooks")
	}
}

func TestEvaluateConformance_BaselineFail_NoSettingsJSON(t *testing.T) {
	defense := DefenseCoverage{
		Layers: []DefenseLayer{
			{Name: "pretooluse-hooks", Weight: WeightCritical, Status: LayerEnabled},
			{Name: "age-gating", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "install-script-blocking", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "lock-file-enforcement", Weight: WeightHigh, Status: LayerEnabled},
			{Name: "vulnerability-scanning", Weight: WeightHigh, Status: LayerEnabled},
		},
	}
	deps := DependencyHealth{
		Ecosystems: []EcosystemStatus{{Name: "go", Detected: true, LockFile: "valid"}},
	}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"CLAUDE.md":               {},
			".pre-commit-config.yaml": {},
			// No .claude/settings.json
		},
	}

	result := EvaluateConformance(defense, deps, map[string]bool{}, genState)

	if result.Baseline.Pass {
		t.Error("expected baseline to fail without settings.json")
	}
}
