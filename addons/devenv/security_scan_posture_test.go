package devenv_test

import (
	"os"
	"path/filepath"
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
