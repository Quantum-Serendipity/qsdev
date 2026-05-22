package posture

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestAssessDefenseLayers_AllEnabled(t *testing.T) {
	enabledTools := map[string]bool{
		"attach-guard":       true,
		"semgrep":            true,
		"gitleaks":           true,
		"ripsecrets":         true,
		"container-security": true,
		"license-compliance": true,
		"socket-dev-mcp":     true,
	}
	detected := types.DetectedProject{
		HasDockerfile: true,
	}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			".claude/hooks/package-guard.py": {},
			"age-gate-npm.yaml":              {},
			".pre-commit-config.yaml":        {},
			".grype.yaml":                    {},
			"devenv.nix":                     {},
			".semgrep.yml":                   {},
		},
	}

	result := AssessDefenseLayers(enabledTools, detected, genState, 3)

	if result.Score != 100.0 {
		t.Errorf("all enabled: score = %f, want 100.0", result.Score)
	}
	if len(result.Layers) != 10 {
		t.Errorf("layer count: got %d, want 10", len(result.Layers))
	}
}

func TestAssessDefenseLayers_ContainerSecurityNA(t *testing.T) {
	enabledTools := map[string]bool{
		"container-security": true,
	}
	detected := types.DetectedProject{
		HasDockerfile: false,
	}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{},
	}

	result := AssessDefenseLayers(enabledTools, detected, genState, 3)

	found := false
	for _, l := range result.Layers {
		if l.Name == "container-security" {
			found = true
			if l.Status != LayerNotApplicable {
				t.Errorf("container-security without Dockerfile: status = %q, want %q", l.Status, LayerNotApplicable)
			}
		}
	}
	if !found {
		t.Error("container-security layer not found")
	}
}

func TestAssessDefenseLayers_ContainerSecurityDisabledWithDockerfile(t *testing.T) {
	enabledTools := map[string]bool{}
	detected := types.DetectedProject{
		HasDockerfile: true,
	}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{},
	}

	result := AssessDefenseLayers(enabledTools, detected, genState, 3)

	for _, l := range result.Layers {
		if l.Name == "container-security" {
			if l.Status != LayerDisabled {
				t.Errorf("container-security with Dockerfile but not enabled: status = %q, want %q", l.Status, LayerDisabled)
			}
			return
		}
	}
	t.Error("container-security layer not found")
}

func TestAssessDefenseLayers_SecretsPartial(t *testing.T) {
	enabledTools := map[string]bool{
		"gitleaks": true,
		// ripsecrets NOT enabled
	}
	detected := types.DetectedProject{}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{},
	}

	result := AssessDefenseLayers(enabledTools, detected, genState, 3)

	for _, l := range result.Layers {
		if l.Name == "secrets-scanning" {
			if l.Status != LayerPartial {
				t.Errorf("secrets-scanning (gitleaks only): status = %q, want %q", l.Status, LayerPartial)
			}
			if l.Score != 5 {
				t.Errorf("secrets-scanning (gitleaks only): score = %d, want 5", l.Score)
			}
			return
		}
	}
	t.Error("secrets-scanning layer not found")
}

func TestAssessDefenseLayers_SecretsFull(t *testing.T) {
	enabledTools := map[string]bool{
		"gitleaks":   true,
		"ripsecrets": true,
	}
	detected := types.DetectedProject{}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{},
	}

	result := AssessDefenseLayers(enabledTools, detected, genState, 3)

	for _, l := range result.Layers {
		if l.Name == "secrets-scanning" {
			if l.Status != LayerEnabled {
				t.Errorf("secrets-scanning (both): status = %q, want %q", l.Status, LayerEnabled)
			}
			if l.Score != 10 {
				t.Errorf("secrets-scanning (both): score = %d, want 10", l.Score)
			}
			return
		}
	}
	t.Error("secrets-scanning layer not found")
}

func TestAssessDefenseLayers_PreToolUsePartial(t *testing.T) {
	// attach-guard enabled but no package-guard.py
	enabledTools := map[string]bool{
		"attach-guard": true,
	}
	detected := types.DetectedProject{}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{},
	}

	result := AssessDefenseLayers(enabledTools, detected, genState, 3)

	for _, l := range result.Layers {
		if l.Name == "pretooluse-hooks" {
			if l.Status != LayerPartial {
				t.Errorf("pretooluse-hooks partial: status = %q, want %q", l.Status, LayerPartial)
			}
			if l.Score != 5 {
				t.Errorf("pretooluse-hooks partial: score = %d, want 5", l.Score)
			}
			return
		}
	}
	t.Error("pretooluse-hooks layer not found")
}

func TestAssessDefenseLayers_PreToolUseFull(t *testing.T) {
	enabledTools := map[string]bool{
		"attach-guard": true,
	}
	detected := types.DetectedProject{}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			".claude/hooks/package-guard.py": {},
		},
	}

	result := AssessDefenseLayers(enabledTools, detected, genState, 3)

	for _, l := range result.Layers {
		if l.Name == "pretooluse-hooks" {
			if l.Status != LayerEnabled {
				t.Errorf("pretooluse-hooks full: status = %q, want %q", l.Status, LayerEnabled)
			}
			return
		}
	}
	t.Error("pretooluse-hooks layer not found")
}

func TestAssessDefenseLayers_NixHardening(t *testing.T) {
	enabledTools := map[string]bool{}
	detected := types.DetectedProject{}

	t.Run("enabled when devenv.nix present", func(t *testing.T) {
		genState := types.GeneratedState{
			Files: map[string]types.FileState{
				"devenv.nix": {},
			},
		}
		result := AssessDefenseLayers(enabledTools, detected, genState, 3)
		for _, l := range result.Layers {
			if l.Name == "nix-hardening" {
				if l.Status != LayerEnabled {
					t.Errorf("nix-hardening: status = %q, want %q", l.Status, LayerEnabled)
				}
				return
			}
		}
		t.Error("nix-hardening layer not found")
	})

	t.Run("disabled when devenv.nix absent", func(t *testing.T) {
		genState := types.GeneratedState{
			Files: map[string]types.FileState{},
		}
		result := AssessDefenseLayers(enabledTools, detected, genState, 3)
		for _, l := range result.Layers {
			if l.Name == "nix-hardening" {
				if l.Status != LayerDisabled {
					t.Errorf("nix-hardening: status = %q, want %q", l.Status, LayerDisabled)
				}
				return
			}
		}
		t.Error("nix-hardening layer not found")
	})
}

func TestAssessDefenseLayers_SAST(t *testing.T) {
	detected := types.DetectedProject{}

	t.Run("fully enabled", func(t *testing.T) {
		enabledTools := map[string]bool{"semgrep": true}
		genState := types.GeneratedState{
			Files: map[string]types.FileState{".semgrep.yml": {}},
		}
		result := AssessDefenseLayers(enabledTools, detected, genState, 3)
		for _, l := range result.Layers {
			if l.Name == "sast" {
				if l.Status != LayerEnabled {
					t.Errorf("sast: status = %q, want %q", l.Status, LayerEnabled)
				}
				return
			}
		}
		t.Error("sast layer not found")
	})

	t.Run("partial - tool only", func(t *testing.T) {
		enabledTools := map[string]bool{"semgrep": true}
		genState := types.GeneratedState{
			Files: map[string]types.FileState{},
		}
		result := AssessDefenseLayers(enabledTools, detected, genState, 3)
		for _, l := range result.Layers {
			if l.Name == "sast" {
				if l.Status != LayerPartial {
					t.Errorf("sast partial: status = %q, want %q", l.Status, LayerPartial)
				}
				return
			}
		}
		t.Error("sast layer not found")
	})
}

func TestAssessDefenseLayers_LicenseCompliance(t *testing.T) {
	detected := types.DetectedProject{}
	genState := types.GeneratedState{Files: map[string]types.FileState{}}

	t.Run("enabled", func(t *testing.T) {
		enabledTools := map[string]bool{"license-compliance": true}
		result := AssessDefenseLayers(enabledTools, detected, genState, 3)
		for _, l := range result.Layers {
			if l.Name == "license-compliance" {
				if l.Status != LayerEnabled {
					t.Errorf("license-compliance: status = %q, want %q", l.Status, LayerEnabled)
				}
				return
			}
		}
		t.Error("license-compliance layer not found")
	})

	t.Run("disabled", func(t *testing.T) {
		enabledTools := map[string]bool{}
		result := AssessDefenseLayers(enabledTools, detected, genState, 3)
		for _, l := range result.Layers {
			if l.Name == "license-compliance" {
				if l.Status != LayerDisabled {
					t.Errorf("license-compliance: status = %q, want %q", l.Status, LayerDisabled)
				}
				return
			}
		}
		t.Error("license-compliance layer not found")
	})
}

func TestAssessDefenseLayers_LayerCount(t *testing.T) {
	result := AssessDefenseLayers(
		map[string]bool{},
		types.DetectedProject{},
		types.GeneratedState{Files: map[string]types.FileState{}},
		3,
	)
	if len(result.Layers) != 10 {
		t.Errorf("layer count: got %d, want 10", len(result.Layers))
	}
}

func TestAssessDefenseLayers_AgeGating(t *testing.T) {
	detected := types.DetectedProject{}

	t.Run("enabled with package-guard", func(t *testing.T) {
		enabledTools := map[string]bool{"attach-guard": true}
		genState := types.GeneratedState{
			Files: map[string]types.FileState{
				".claude/hooks/package-guard.py": {},
			},
		}
		result := AssessDefenseLayers(enabledTools, detected, genState, 3)
		for _, l := range result.Layers {
			if l.Name == "age-gating" {
				if l.Status != LayerEnabled {
					t.Errorf("age-gating: status = %q, want %q", l.Status, LayerEnabled)
				}
				return
			}
		}
		t.Error("age-gating layer not found")
	})

	t.Run("disabled without attach-guard", func(t *testing.T) {
		enabledTools := map[string]bool{}
		genState := types.GeneratedState{
			Files: map[string]types.FileState{
				".claude/hooks/package-guard.py": {},
			},
		}
		result := AssessDefenseLayers(enabledTools, detected, genState, 3)
		for _, l := range result.Layers {
			if l.Name == "age-gating" {
				if l.Status != LayerDisabled {
					t.Errorf("age-gating without attach-guard: status = %q, want %q", l.Status, LayerDisabled)
				}
				return
			}
		}
		t.Error("age-gating layer not found")
	})
}

func TestAssessDefenseLayers_MinTierValues(t *testing.T) {
	t.Parallel()
	result := AssessDefenseLayers(
		map[string]bool{},
		types.DetectedProject{},
		types.GeneratedState{Files: map[string]types.FileState{}},
		3,
	)

	expectedMinTier := map[string]int{
		"pretooluse-hooks":        1,
		"install-script-blocking": 1,
		"lock-file-enforcement":   1,
		"vulnerability-scanning":  1,
		"age-gating":              2,
		"secrets-scanning":        2,
		"sast":                    3,
		"nix-hardening":           3,
		"container-security":      3,
		"license-compliance":      3,
	}

	for _, l := range result.Layers {
		want, ok := expectedMinTier[l.Name]
		if !ok {
			t.Errorf("unexpected layer name %q", l.Name)
			continue
		}
		if l.MinTier != want {
			t.Errorf("layer %q: MinTier = %d, want %d", l.Name, l.MinTier, want)
		}
	}

	// Verify all expected layers were found.
	layerNames := make(map[string]bool)
	for _, l := range result.Layers {
		layerNames[l.Name] = true
	}
	for name := range expectedMinTier {
		if !layerNames[name] {
			t.Errorf("expected layer %q not found in results", name)
		}
	}
}

func TestAssessDefenseLayers_T1ScoreIgnoresHigherTierLayers(t *testing.T) {
	t.Parallel()
	enabledTools := map[string]bool{
		"attach-guard": true,
	}
	detected := types.DetectedProject{}
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			".claude/hooks/package-guard.py": {},
		},
	}

	// At tier 1, only T1 layers are considered.
	// pretooluse-hooks (T1, critical) should be enabled.
	// Higher-tier layers like secrets-scanning (T2), sast (T3) should be excluded.
	result := AssessDefenseLayers(enabledTools, detected, genState, 1)

	if result.Score == 0 {
		t.Error("T1 score should not be 0 when T1 layers are enabled")
	}

	// Now test at tier 3 with same tools — score should be lower because
	// higher-tier layers are included but disabled.
	resultT3 := AssessDefenseLayers(enabledTools, detected, genState, 3)

	if resultT3.Score >= result.Score {
		t.Errorf("T3 score (%f) should be lower than T1 score (%f) with same tools, because more layers are in scope but disabled",
			resultT3.Score, result.Score)
	}
}
