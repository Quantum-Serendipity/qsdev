package ecosystem

import "github.com/Quantum-Serendipity/qsdev/pkg/types"

// MockModule is a configurable implementation of EcosystemModule for testing.
// Each field corresponds to the return value of the matching interface method.
type MockModule struct {
	NameVal        string
	DisplayNameVal string
	TierVal        int

	// DetectFn, when set, is called by Detect. If nil, DetectResult is returned.
	DetectFn     func(projectRoot string) DetectionResult
	DetectResult DetectionResult

	DevenvNixFragmentVal   string
	DevenvNixFragmentErr   error
	DevenvYamlInputsVal    []DevenvInput
	SecurityConfigsVal     []types.GeneratedFile
	PreCommitHooksVal      []HookConfig
	DenyRulesVal           []string
	CICommandsVal          []CICommand
	PackageManagersVal         []PackageManagerInfo
	WizardFieldsVal            []WizardField
	VerificationCommandsVal    VerificationCommands
	ManifestFilesVal           []ManifestFileInfo
}

func (m *MockModule) Name() string        { return m.NameVal }
func (m *MockModule) DisplayName() string  { return m.DisplayNameVal }
func (m *MockModule) Tier() int            { return m.TierVal }

func (m *MockModule) Detect(projectRoot string) DetectionResult {
	if m.DetectFn != nil {
		return m.DetectFn(projectRoot)
	}
	return m.DetectResult
}

func (m *MockModule) DevenvNixFragment(config ModuleConfig) (string, error) {
	return m.DevenvNixFragmentVal, m.DevenvNixFragmentErr
}

func (m *MockModule) DevenvYamlInputs(config ModuleConfig) []DevenvInput {
	return m.DevenvYamlInputsVal
}

func (m *MockModule) SecurityConfigs(config ModuleConfig) []types.GeneratedFile {
	return m.SecurityConfigsVal
}

func (m *MockModule) PreCommitHooks(config ModuleConfig) []HookConfig {
	return m.PreCommitHooksVal
}

func (m *MockModule) DenyRules(config ModuleConfig) []string {
	return m.DenyRulesVal
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

func (m *MockModule) VerificationCommands(_ ModuleConfig) VerificationCommands {
	return m.VerificationCommandsVal
}

func (m *MockModule) ManifestFiles(_ ModuleConfig) []ManifestFileInfo {
	return m.ManifestFilesVal
}
