package ecosystem_test

import (
	"testing"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*ecosystem.MockModule)(nil)

func TestMockModuleReturnsConfiguredValues(t *testing.T) {
	m := &ecosystem.MockModule{
		NameVal:        "go",
		DisplayNameVal: "Go",
		TierVal:        1,
		DetectResult: ecosystem.DetectionResult{
			Detected:   true,
			Confidence: ecosystem.ConfidenceCertain,
			Evidence:   []string{"go.mod found"},
		},
		DevenvNixFragmentVal: "languages.go.enable = true;",
		DenyRulesVal:         []string{"go install --insecure"},
		WizardFieldsVal: []ecosystem.WizardField{
			{Key: "go_version", Label: "Go Version", Type: ecosystem.FieldTypeSelect},
		},
	}

	if m.Name() != "go" {
		t.Errorf("Name() = %q, want %q", m.Name(), "go")
	}
	if m.DisplayName() != "Go" {
		t.Errorf("DisplayName() = %q, want %q", m.DisplayName(), "Go")
	}
	if m.Tier() != 1 {
		t.Errorf("Tier() = %d, want %d", m.Tier(), 1)
	}

	dr := m.Detect("/tmp/project")
	if !dr.Detected {
		t.Error("Detect().Detected = false, want true")
	}
	if dr.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Detect().Confidence = %v, want %v", dr.Confidence, ecosystem.ConfidenceCertain)
	}
	if len(dr.Evidence) != 1 || dr.Evidence[0] != "go.mod found" {
		t.Errorf("Detect().Evidence = %v, want [go.mod found]", dr.Evidence)
	}

	nix, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment error: %v", err)
	}
	if nix != "languages.go.enable = true;" {
		t.Errorf("DevenvNixFragment() = %q, want %q", nix, "languages.go.enable = true;")
	}

	rules := m.DenyRules(ecosystem.ModuleConfig{})
	if len(rules) != 1 || rules[0] != "go install --insecure" {
		t.Errorf("DenyRules() = %v, want [go install --insecure]", rules)
	}

	fields := m.WizardFields()
	if len(fields) != 1 || fields[0].Key != "go_version" {
		t.Errorf("WizardFields() = %v, want [{Key:go_version ...}]", fields)
	}
}

func TestMockModuleDetectFnOverridesResult(t *testing.T) {
	m := &ecosystem.MockModule{
		NameVal: "python",
		DetectResult: ecosystem.DetectionResult{
			Detected: false,
		},
		DetectFn: func(projectRoot string) ecosystem.DetectionResult {
			return ecosystem.DetectionResult{
				Detected:   true,
				Confidence: ecosystem.ConfidenceProbable,
				Evidence:   []string{"detected via DetectFn for " + projectRoot},
			}
		},
	}

	dr := m.Detect("/my/project")
	if !dr.Detected {
		t.Error("DetectFn should override DetectResult; Detected = false, want true")
	}
	if dr.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want %v", dr.Confidence, ecosystem.ConfidenceProbable)
	}
	if len(dr.Evidence) != 1 {
		t.Fatalf("Evidence length = %d, want 1", len(dr.Evidence))
	}
	want := "detected via DetectFn for /my/project"
	if dr.Evidence[0] != want {
		t.Errorf("Evidence[0] = %q, want %q", dr.Evidence[0], want)
	}
}

func TestMockModuleNilReturnValues(t *testing.T) {
	m := &ecosystem.MockModule{NameVal: "empty"}

	if inputs := m.DevenvYamlInputs(ecosystem.ModuleConfig{}); inputs != nil {
		t.Errorf("DevenvYamlInputs() = %v, want nil", inputs)
	}
	if configs := m.SecurityConfigs(ecosystem.ModuleConfig{}); configs != nil {
		t.Errorf("SecurityConfigs() = %v, want nil", configs)
	}
	if hooks := m.PreCommitHooks(ecosystem.ModuleConfig{}); hooks != nil {
		t.Errorf("PreCommitHooks() = %v, want nil", hooks)
	}
	if rules := m.DenyRules(ecosystem.ModuleConfig{}); rules != nil {
		t.Errorf("DenyRules() = %v, want nil", rules)
	}
	if cmds := m.CICommands(ecosystem.ModuleConfig{}); cmds != nil {
		t.Errorf("CICommands() = %v, want nil", cmds)
	}
	if pms := m.PackageManagers(); pms != nil {
		t.Errorf("PackageManagers() = %v, want nil", pms)
	}
	if fields := m.WizardFields(); fields != nil {
		t.Errorf("WizardFields() = %v, want nil", fields)
	}
}
