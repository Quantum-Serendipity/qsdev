package types

import "testing"

// TestFillDefaults_NoDefaultTierFallsBackToStandardPreset verifies that only
// a provider declaring no default tier leaves the tier unset, and that the
// standard permission preset then still applies to Claude Code.
func TestFillDefaults_NoDefaultTierFallsBackToStandardPreset(t *testing.T) {
	t.Parallel()
	a := WizardAnswers{ClaudeCode: true}
	a.FillDefaults(DetectedProject{}, zeroDefaults{})
	if a.Tier != "" {
		t.Errorf("Tier = %q, want empty", a.Tier)
	}
	if a.PermissionLevel != "standard" {
		t.Errorf("PermissionLevel = %q, want standard", a.PermissionLevel)
	}
}
