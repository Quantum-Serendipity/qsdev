package posture

import (
	"strings"
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
	dir, genState := writeProjectFiles(t, map[string]string{
		".claude/hooks/package-guard.py": "",
		".claude/settings.json":          settingsWithPackageGuard,
		".pre-commit-config.yaml":        preCommitWithLockAudit,
		".grype.yaml":                    "",
		"devenv.nix":                     hardenedDevenvNix,
		".semgrepignore":                 "",
	})

	result := AssessDefenseLayers(dir, enabledTools, detected, genState, 3)

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

	result := AssessDefenseLayers("", enabledTools, detected, genState, 3)

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

	result := AssessDefenseLayers("", enabledTools, detected, genState, 3)

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

	result := AssessDefenseLayers("", enabledTools, detected, genState, 3)

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

	result := AssessDefenseLayers("", enabledTools, detected, genState, 3)

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

	result := AssessDefenseLayers("", enabledTools, detected, genState, 3)

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

// TestAssessDefenseLayers_PreToolUseFull checks the layer is judged from the
// hook registered in settings.json, not from package-guard.py existing: with
// the registration removed the script guards nothing.
func TestAssessDefenseLayers_PreToolUseFull(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		settings string
		want     LayerStatus
	}{
		{"registered", settingsWithPackageGuard, LayerEnabled},
		{"hooks stripped", `{"permissions": {"defaultMode": "bypassPermissions"}}`, LayerPartial},
		{"guard registered under another event", strings.Replace(settingsWithPackageGuard, "PreToolUse", "PostToolUse", 1), LayerPartial},
		{"all hooks disabled", strings.Replace(settingsWithPackageGuard, "{", `{"disableAllHooks": true, `, 1), LayerPartial},
		{"guard only under a decoy-cased key", strings.Replace(settingsWithPackageGuard, `"hooks"`, `"Hooks"`, 1), LayerPartial},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir, genState := writeProjectFiles(t, map[string]string{
				".claude/hooks/package-guard.py": "",
				".claude/settings.json":          tt.settings,
			})
			result := AssessDefenseLayers(dir, map[string]bool{"attach-guard": true}, types.DetectedProject{}, genState, 3)
			if got := layerByName(t, result, "pretooluse-hooks"); got.Status != tt.want {
				t.Errorf("pretooluse-hooks: status = %q (%s), want %q", got.Status, got.Reason, tt.want)
			}
		})
	}
}

func TestAssessDefenseLayers_NixHardening(t *testing.T) {
	enabledTools := map[string]bool{}
	detected := types.DetectedProject{}

	t.Run("enabled when devenv.nix carries hardening", func(t *testing.T) {
		dir, genState := writeProjectFiles(t, map[string]string{"devenv.nix": hardenedDevenvNix})
		result := AssessDefenseLayers(dir, enabledTools, detected, genState, 3)
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
		result := AssessDefenseLayers("", enabledTools, detected, genState, 3)
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

// TestAssessDefenseLayers_SAST is the F202 regression: SAST is credited only
// when the security-scan task script in devenv.nix actually runs semgrep, not
// for a config file's presence (the old .semgrep.yml was invalid and unused).
func TestAssessDefenseLayers_SAST(t *testing.T) {
	t.Parallel()

	scanScript := func(exec string) string {
		return "{ pkgs, ... }:\n{\n  scripts.\"qsdev-security-scan\" = {\n    description = \"Run security scanners\";\n    exec = ''\n" +
			exec + "\n    '';\n  };\n}\n"
	}
	const semgrepLine = "      semgrep --config p/golang --metrics=off --error ."

	tests := []struct {
		name  string
		tools map[string]bool
		files map[string]string
		want  LayerStatus
	}{
		{
			name:  "enabled and wired",
			tools: map[string]bool{"semgrep": true},
			files: map[string]string{"devenv.nix": scanScript("      set -euo pipefail\n" + semgrepLine)},
			want:  LayerEnabled,
		},
		{
			name:  "grouped scripts attrset",
			tools: map[string]bool{"semgrep": true},
			files: map[string]string{"devenv.nix": "{\n  scripts = {\n    \"qsdev-build\" = {\n      exec = ''\n        go build\n      '';\n    };\n" +
				"    \"qsdev-security-scan\" = {\n      description = \"Run security scanners\";\n      exec = ''\n  " + semgrepLine + "\n      '';\n    };\n  };\n}\n"},
			want: LayerEnabled,
		},
		{
			name:  "similarly named script",
			tools: map[string]bool{"semgrep": true},
			files: map[string]string{"devenv.nix": "{\n  scripts.\"my-qsdev-security-scan\".exec = ''\n" + semgrepLine + "\n  '';\n}\n"},
			want:  LayerPartial,
		},
		{
			name:  "dotted exec form",
			tools: map[string]bool{"semgrep": true},
			files: map[string]string{"devenv.nix": "{\n  scripts.qsdev-security-scan.exec = ''\n" + semgrepLine + "\n  '';\n}\n"},
			want:  LayerEnabled,
		},
		{
			name:  "escaped quotes before semgrep line",
			tools: map[string]bool{"semgrep": true},
			files: map[string]string{"devenv.nix": scanScript("      echo '''quoted''' ''${HOME}\n" + semgrepLine)},
			want:  LayerEnabled,
		},
		{
			name:  "stale ignore file alone is not SAST",
			tools: map[string]bool{"semgrep": true},
			files: map[string]string{".semgrepignore": "vendor/\n", ".semgrep.yml": "rules:\n  - p/golang\n"},
			want:  LayerPartial,
		},
		{
			name:  "task without semgrep",
			tools: map[string]bool{"semgrep": true},
			files: map[string]string{"devenv.nix": scanScript("      gitleaks detect --no-banner")},
			want:  LayerPartial,
		},
		{
			name:  "semgrep commented out",
			tools: map[string]bool{"semgrep": true},
			files: map[string]string{"devenv.nix": scanScript("      # " + semgrepLine[6:])},
			want:  LayerPartial,
		},
		{
			name:  "semgrep in another script",
			tools: map[string]bool{"semgrep": true},
			files: map[string]string{"devenv.nix": "{\n  scripts.\"qsdev-lint\" = {\n    exec = ''\n" + semgrepLine +
				"\n    '';\n  };\n  scripts.\"qsdev-security-scan\" = {\n    exec = ''\n      gitleaks detect\n    '';\n  };\n}\n"},
			want: LayerPartial,
		},
		{
			name:  "unterminated script",
			tools: map[string]bool{"semgrep": true},
			files: map[string]string{"devenv.nix": "{\n  scripts.\"qsdev-security-scan\" = {\n    exec = ''\n" + semgrepLine + "\n"},
			want:  LayerPartial,
		},
		{
			name:  "wired but tool not enabled",
			tools: map[string]bool{},
			files: map[string]string{"devenv.nix": scanScript(semgrepLine)},
			want:  LayerPartial,
		},
		{
			name:  "not enabled",
			tools: map[string]bool{},
			files: map[string]string{},
			want:  LayerDisabled,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir, genState := writeProjectFiles(t, tt.files)
			l := layerByName(t, AssessDefenseLayers(dir, tt.tools, types.DetectedProject{}, genState, 3), "sast")
			if l.Status != tt.want {
				t.Errorf("sast: status = %q (%s), want %q", l.Status, l.Reason, tt.want)
			}
		})
	}
}

// TestAssessDefenseLayers_LicenseCompliance is the F218 regression: the layer
// is credited only when the security-scan task script in devenv.nix runs
// ScanCode with the license policy, not merely because the tool is enabled
// (the policy file on its own enforces nothing).
func TestAssessDefenseLayers_LicenseCompliance(t *testing.T) {
	t.Parallel()

	scanScript := func(exec string) string {
		return "{ pkgs, ... }:\n{\n  scripts.\"qsdev-security-scan\" = {\n    description = \"Run security scanners\";\n    exec = ''\n" +
			exec + "\n    '';\n  };\n}\n"
	}
	const scanLine = "      scancode --quiet --license --license-policy .scancode.yml --ignore '.git' --json - . | jq -r '.files'"

	tests := []struct {
		name  string
		tools map[string]bool
		files map[string]string
		want  LayerStatus
	}{
		{
			name:  "enabled and run by the task",
			tools: map[string]bool{"license-compliance": true},
			files: map[string]string{"devenv.nix": scanScript("      set -euo pipefail\n" + scanLine)},
			want:  LayerEnabled,
		},
		{
			name:  "enabled with only the policy file",
			tools: map[string]bool{"license-compliance": true},
			files: map[string]string{".scancode.yml": "license_policies: []\n"},
			want:  LayerPartial,
		},
		{
			name:  "task scans licenses without a policy",
			tools: map[string]bool{"license-compliance": true},
			files: map[string]string{"devenv.nix": scanScript("      scancode --license --json - .")},
			want:  LayerPartial,
		},
		{
			name:  "scan commented out",
			tools: map[string]bool{"license-compliance": true},
			files: map[string]string{"devenv.nix": scanScript("      # " + scanLine[6:])},
			want:  LayerPartial,
		},
		{
			name:  "run by the task but not enabled",
			tools: map[string]bool{},
			files: map[string]string{"devenv.nix": scanScript(scanLine)},
			want:  LayerPartial,
		},
		{
			name:  "disabled",
			tools: map[string]bool{},
			want:  LayerDisabled,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir, genState := writeProjectFiles(t, tt.files)
			l := layerByName(t, AssessDefenseLayers(dir, tt.tools, types.DetectedProject{}, genState, 3), "license-compliance")
			if l.Status != tt.want {
				t.Errorf("license-compliance: status = %q (%s), want %q", l.Status, l.Reason, tt.want)
			}
		})
	}
}

func TestAssessDefenseLayers_LayerCount(t *testing.T) {
	result := AssessDefenseLayers(
		"",
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
		result := AssessDefenseLayers("", enabledTools, detected, genState, 3)
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

	// W027: the guard has no age check for Java (or .NET, ...), so a project
	// using one is only partly age-gated, and the report says which.
	t.Run("partial for ecosystems the guard does not age-check", func(t *testing.T) {
		enabledTools := map[string]bool{"attach-guard": true}
		genState := types.GeneratedState{
			Files: map[string]types.FileState{
				".claude/hooks/package-guard.py": {},
			},
		}
		tests := []struct {
			name       string
			detected   types.DetectedProject
			wantStatus LayerStatus
			wantInText string
		}{
			{"go", types.DetectedProject{HasGoMod: true}, LayerEnabled, ""},
			{"java and javascript", types.DetectedProject{HasPomXML: true, HasPackageJSON: true}, LayerPartial, "not for: java"},
			{"java", types.DetectedProject{HasPomXML: true}, LayerPartial, "java"},
			{"javascript only", types.DetectedProject{HasPackageJSON: true}, LayerEnabled, ""},
			{"python and rust", types.DetectedProject{HasPyProject: true, HasCargoToml: true}, LayerEnabled, ""},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := AssessDefenseLayers("", enabledTools, tt.detected, genState, 3)
				for _, l := range result.Layers {
					if l.Name != "age-gating" {
						continue
					}
					if l.Status != tt.wantStatus || !strings.Contains(l.Reason, tt.wantInText) {
						t.Errorf("age-gating = %q (%q), want %q containing %q", l.Status, l.Reason, tt.wantStatus, tt.wantInText)
					}
					return
				}
				t.Error("age-gating layer not found")
			})
		}
	})

	t.Run("disabled without attach-guard", func(t *testing.T) {
		enabledTools := map[string]bool{}
		genState := types.GeneratedState{
			Files: map[string]types.FileState{
				".claude/hooks/package-guard.py": {},
			},
		}
		result := AssessDefenseLayers("", enabledTools, detected, genState, 3)
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
		"",
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
	result := AssessDefenseLayers("", enabledTools, detected, genState, 1)

	if result.Score == 0 {
		t.Error("T1 score should not be 0 when T1 layers are enabled")
	}

	// Now test at tier 3 with same tools — score should be lower because
	// higher-tier layers are included but disabled.
	resultT3 := AssessDefenseLayers("", enabledTools, detected, genState, 3)

	if resultT3.Score >= result.Score {
		t.Errorf("T3 score (%f) should be lower than T1 score (%f) with same tools, because more layers are in scope but disabled",
			resultT3.Score, result.Score)
	}
}
