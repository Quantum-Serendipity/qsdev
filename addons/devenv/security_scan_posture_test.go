package devenv_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestGenerateDevenvNix_SecurityScanCreditsSAST pins the contract between the
// devenv.nix the addon renders and posture's SAST layer (F202): posture credits
// SAST only when the security-scan task script runs semgrep, so the rendered
// script must be recognised exactly when semgrep is enabled.
func TestGenerateDevenvNix_SecurityScanCreditsSAST(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		tools map[string]bool
		want  posture.LayerStatus
	}{
		{name: "semgrep enabled", tools: map[string]bool{"semgrep": true, "gitleaks": true}, want: posture.LayerEnabled},
		{name: "semgrep disabled", tools: map[string]bool{"gitleaks": true}, want: posture.LayerDisabled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			answers := types.WizardAnswers{
				ProjectName:  "demo",
				Languages:    []types.LanguageChoice{{Name: "go"}},
				EnabledTools: tt.tools,
			}
			got, err := devenv.GenerateDevenvNix(answers, ecosystem.DefaultRegistry())
			if err != nil {
				t.Fatalf("GenerateDevenvNix: %v", err)
			}
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, got.Path), got.Content, 0o644); err != nil {
				t.Fatal(err)
			}
			state := types.GeneratedState{Files: map[string]types.FileState{got.Path: {}}}

			cov := posture.AssessDefenseLayers(dir, tt.tools, types.DetectedProject{}, state, 3)
			l := posture.FindLayerByName(cov.Layers, "sast")
			if l == nil {
				t.Fatal("sast layer not found")
			}
			if l.Status != tt.want {
				t.Errorf("sast status = %q (%s), want %q", l.Status, l.Reason, tt.want)
			}
		})
	}
}

// TestGenerateDevenvNix_SecurityScanCreditsLicenseCompliance pins the same
// contract for the license-compliance layer (F218): enabling the tool must
// install ScanCode and render a security-scan task that runs it with the
// license policy, which posture recognises; without the tool neither appears.
func TestGenerateDevenvNix_SecurityScanCreditsLicenseCompliance(t *testing.T) {
	t.Parallel()
	const scancodePkg = "python3Packages.scancode-toolkit"
	tests := []struct {
		name    string
		tools   map[string]bool
		want    posture.LayerStatus
		wantPkg bool
	}{
		{name: "enabled", tools: map[string]bool{"license-compliance": true, "gitleaks": true}, want: posture.LayerEnabled, wantPkg: true},
		{name: "disabled", tools: map[string]bool{"gitleaks": true}, want: posture.LayerDisabled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			answers := types.WizardAnswers{
				ProjectName:  "demo",
				Languages:    []types.LanguageChoice{{Name: "go"}},
				EnabledTools: tt.tools,
			}
			got, err := devenv.GenerateDevenvNix(answers, ecosystem.DefaultRegistry())
			if err != nil {
				t.Fatalf("GenerateDevenvNix: %v", err)
			}
			if has := strings.Contains(string(got.Content), scancodePkg); has != tt.wantPkg {
				t.Errorf("devenv.nix contains %s = %v, want %v", scancodePkg, has, tt.wantPkg)
			}
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, got.Path), got.Content, 0o644); err != nil {
				t.Fatal(err)
			}
			state := types.GeneratedState{Files: map[string]types.FileState{got.Path: {}}}

			cov := posture.AssessDefenseLayers(dir, tt.tools, types.DetectedProject{}, state, 3)
			l := posture.FindLayerByName(cov.Layers, "license-compliance")
			if l == nil {
				t.Fatal("license-compliance layer not found")
			}
			if l.Status != tt.want {
				t.Errorf("license-compliance status = %q (%s), want %q", l.Status, l.Reason, tt.want)
			}
		})
	}
}
