package ecosystem

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// AssertModuleIdentity checks the three identity properties (Name, DisplayName,
// Tier) of an EcosystemModule. It is intended for cross-package use in module
// test files to eliminate boilerplate.
func AssertModuleIdentity(t *testing.T, m EcosystemModule, wantName, wantDisplay string, wantTier int) {
	t.Helper()

	if got := m.Name(); got != wantName {
		t.Errorf("Name() = %q, want %q", got, wantName)
	}
	if got := m.DisplayName(); got != wantDisplay {
		t.Errorf("DisplayName() = %q, want %q", got, wantDisplay)
	}
	if got := m.Tier(); got != wantTier {
		t.Errorf("Tier() = %d, want %d", got, wantTier)
	}
}

// Compile-time interface compliance checks.
var _ EcosystemModule = (*MockModule)(nil)
var _ DevenvYamlInputProvider = (*MockModule)(nil)
var _ DenyRuleProvider = (*MockModule)(nil)
var _ WizardFieldProvider = (*MockModule)(nil)
var _ ManifestFileProvider = (*MockModule)(nil)
var _ PackageProvider = (*MockModule)(nil)
var _ ReadDenyRuleProvider = (*MockModule)(nil)

// MockModule is a configurable implementation of EcosystemModule for testing.
// Each field corresponds to the return value of the matching interface method.
type MockModule struct {
	NameVal        string
	DisplayNameVal string
	TierVal        int

	// DetectFn, when set, is called by Detect. If nil, DetectResult is returned.
	DetectFn     func(projectRoot string) DetectionResult
	DetectResult DetectionResult

	DevenvNixFragmentVal    string
	DevenvNixFragmentErr    error
	DevenvYamlInputsVal     []DevenvInput
	SecurityConfigsVal      []types.GeneratedFile
	PreCommitHooksVal       []HookConfig
	DenyRulesVal            []string
	ReadDenyRulesVal        []string
	CICommandsVal           []CICommand
	PackageManagersVal      []PackageManagerInfo
	WizardFieldsVal         []WizardField
	VerificationCommandsVal VerificationCommands
	ManifestFilesVal        []ManifestFileInfo
	DevenvPackagesVal       []string
}

func (m *MockModule) Name() string        { return m.NameVal }
func (m *MockModule) DisplayName() string { return m.DisplayNameVal }
func (m *MockModule) Tier() int           { return m.TierVal }

func (m *MockModule) Detect(projectRoot string) DetectionResult {
	if m.DetectFn != nil {
		return m.DetectFn(projectRoot)
	}
	return m.DetectResult
}

func (m *MockModule) DevenvNixFragment(config ModuleConfig) (string, error) {
	return m.DevenvNixFragmentVal, m.DevenvNixFragmentErr
}

func (m *MockModule) DevenvYamlInputs(_ ModuleConfig) []DevenvInput {
	return m.DevenvYamlInputsVal
}

func (m *MockModule) SecurityConfigs(config ModuleConfig) []types.GeneratedFile {
	return m.SecurityConfigsVal
}

func (m *MockModule) PreCommitHooks(config ModuleConfig) []HookConfig {
	return m.PreCommitHooksVal
}

func (m *MockModule) DenyRules(_ ModuleConfig) []string {
	return m.DenyRulesVal
}

func (m *MockModule) ReadDenyRules(_ ModuleConfig) []string {
	return m.ReadDenyRulesVal
}

func (m *MockModule) CICommands(config ModuleConfig) []CICommand {
	return m.CICommandsVal
}

func (m *MockModule) PackageManagers() []PackageManagerInfo {
	return m.PackageManagersVal
}

func (m *MockModule) WizardFields() []WizardField {
	return m.WizardFieldsVal
}

func (m *MockModule) ManifestFiles(_ ModuleConfig) []ManifestFileInfo {
	return m.ManifestFilesVal
}

func (m *MockModule) VerificationCommands(_ ModuleConfig) VerificationCommands {
	return m.VerificationCommandsVal
}

func (m *MockModule) DevenvPackages(_ ModuleConfig) []string {
	return m.DevenvPackagesVal
}
