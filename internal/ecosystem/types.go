package ecosystem

import "fmt"

// Confidence indicates how certain the detection logic is that an ecosystem
// is present in a project directory.
type Confidence int

const (
	ConfidenceAbsent   Confidence = iota // No indicators found.
	ConfidenceProbable                   // Some indicators found (e.g. file extensions).
	ConfidenceCertain                    // Definitive marker found (e.g. go.mod, package.json).
)

var confidenceNames = [...]string{
	ConfidenceAbsent:   "absent",
	ConfidenceProbable: "probable",
	ConfidenceCertain:  "certain",
}

func (c Confidence) String() string {
	if int(c) < len(confidenceNames) {
		return confidenceNames[c]
	}
	return "unknown"
}

func (c Confidence) MarshalText() ([]byte, error) {
	s := c.String()
	if s == "unknown" {
		return nil, fmt.Errorf("cannot marshal unknown Confidence value %d", int(c))
	}
	return []byte(s), nil
}

func (c *Confidence) UnmarshalText(text []byte) error {
	for i, name := range confidenceNames {
		if name == string(text) {
			*c = Confidence(i)
			return nil
		}
	}
	return fmt.Errorf("unknown confidence level: %q", string(text))
}

// DetectionResult is returned by an EcosystemModule's Detect method.
type DetectionResult struct {
	Detected        bool         `yaml:"detected"         json:"detected"`
	Confidence      Confidence   `yaml:"confidence"       json:"confidence"`
	Evidence        []string     `yaml:"evidence"         json:"evidence"`
	SuggestedConfig ModuleConfig `yaml:"suggested_config" json:"suggested_config"`
}

// ModuleConfig holds the user-facing configuration for an ecosystem module,
// typically populated from detection or wizard answers.
type ModuleConfig struct {
	Version        string            `yaml:"version"         json:"version"`
	PackageManager string            `yaml:"package_manager" json:"package_manager"`
	Extras         map[string]string `yaml:"extras"          json:"extras"`
}

// DevenvInput represents a devenv.sh input (flake reference) to be added
// to devenv.yaml.
type DevenvInput struct {
	URL     string            `yaml:"url"              json:"url"`
	Follows map[string]string `yaml:"follows,omitempty" json:"follows,omitempty"`
}

// HookConfig represents a pre-commit hook configuration entry.
type HookConfig struct {
	ID                     string   `yaml:"id"                        json:"id"`
	Name                   string   `yaml:"name"                      json:"name"`
	Description            string   `yaml:"description"               json:"description"`
	Entry                  string   `yaml:"entry"                     json:"entry"`
	Language               string   `yaml:"language"                  json:"language"`
	Types                  []string `yaml:"types"                     json:"types"`
	Stages                 []string `yaml:"stages"                    json:"stages"`
	PassFilenames          bool     `yaml:"pass_filenames"            json:"pass_filenames"`
	Files                  string   `yaml:"files"                     json:"files"`
	AdditionalDependencies []string `yaml:"additional_dependencies"   json:"additional_dependencies"`
	BuiltIn                bool     `yaml:"built_in"                  json:"built_in"`
}

// CIPhase categorizes a CI command into a build pipeline phase.
type CIPhase int

const (
	CIPhaseInstall CIPhase = iota
	CIPhaseTest
	CIPhaseScan
)

var ciPhaseNames = [...]string{
	CIPhaseInstall: "install",
	CIPhaseTest:    "test",
	CIPhaseScan:    "scan",
}

func (p CIPhase) String() string {
	if int(p) < len(ciPhaseNames) {
		return ciPhaseNames[p]
	}
	return "unknown"
}

func (p CIPhase) MarshalText() ([]byte, error) {
	s := p.String()
	if s == "unknown" {
		return nil, fmt.Errorf("cannot marshal unknown CIPhase value %d", int(p))
	}
	return []byte(s), nil
}

func (p *CIPhase) UnmarshalText(text []byte) error {
	for i, name := range ciPhaseNames {
		if name == string(text) {
			*p = CIPhase(i)
			return nil
		}
	}
	return fmt.Errorf("unknown CI phase: %q", string(text))
}

// CICommand represents a command to include in CI pipeline configuration.
type CICommand struct {
	Name        string  `yaml:"name"        json:"name"`
	Command     string  `yaml:"command"     json:"command"`
	Description string  `yaml:"description" json:"description"`
	Phase       CIPhase `yaml:"phase"       json:"phase"`
}

// PackageManagerInfo describes a package manager's capabilities and commands,
// used for security policy generation and CI integration.
type PackageManagerInfo struct {
	Name                 string `yaml:"name"                   json:"name"`
	LockFile             string `yaml:"lock_file"              json:"lock_file"`
	InstallCommand       string `yaml:"install_command"        json:"install_command"`
	FrozenInstallCommand string `yaml:"frozen_install_command" json:"frozen_install_command"`
	AuditCommand         string `yaml:"audit_command"          json:"audit_command"`
	AgeGatingSupport     bool   `yaml:"age_gating_support"     json:"age_gating_support"`
}

// WizardFieldType categorizes the kind of TUI form widget to render.
type WizardFieldType int

const (
	FieldTypeSelect      WizardFieldType = iota // Single-choice dropdown.
	FieldTypeMultiSelect                        // Multi-choice checkboxes.
	FieldTypeInput                              // Free-text input.
	FieldTypeConfirm                            // Yes/no confirmation.
)

var wizardFieldTypeNames = [...]string{
	FieldTypeSelect:      "select",
	FieldTypeMultiSelect: "multi_select",
	FieldTypeInput:       "input",
	FieldTypeConfirm:     "confirm",
}

func (f WizardFieldType) String() string {
	if int(f) < len(wizardFieldTypeNames) {
		return wizardFieldTypeNames[f]
	}
	return "unknown"
}

func (f WizardFieldType) MarshalText() ([]byte, error) {
	s := f.String()
	if s == "unknown" {
		return nil, fmt.Errorf("cannot marshal unknown WizardFieldType value %d", int(f))
	}
	return []byte(s), nil
}

func (f *WizardFieldType) UnmarshalText(text []byte) error {
	for i, name := range wizardFieldTypeNames {
		if name == string(text) {
			*f = WizardFieldType(i)
			return nil
		}
	}
	return fmt.Errorf("unknown wizard field type: %q", string(text))
}

// WizardField describes a single form field that an ecosystem module
// contributes to the init wizard.
type WizardField struct {
	Key         string          `yaml:"key"         json:"key"`
	Label       string          `yaml:"label"       json:"label"`
	Description string          `yaml:"description" json:"description"`
	Type        WizardFieldType `yaml:"type"        json:"type"`
	Options     []WizardOption  `yaml:"options"     json:"options"`
	Default     string          `yaml:"default"     json:"default"`
	Required    bool            `yaml:"required"    json:"required"`
	Condition   string          `yaml:"condition"   json:"condition"`
}

// WizardOption is a single selectable option within a WizardField.
type WizardOption struct {
	Label string `yaml:"label" json:"label"`
	Value string `yaml:"value" json:"value"`
}

// VerificationCommands holds the project verification commands for an ecosystem,
// organized by category.
type VerificationCommands struct {
	Build     []string `yaml:"build,omitempty"      json:"build,omitempty"`
	Test      []string `yaml:"test,omitempty"       json:"test,omitempty"`
	Lint      []string `yaml:"lint,omitempty"       json:"lint,omitempty"`
	TypeCheck []string `yaml:"type_check,omitempty" json:"type_check,omitempty"`
	Format    []string `yaml:"format,omitempty"     json:"format,omitempty"`
}

// IsEmpty returns true when all command categories are empty.
func (v VerificationCommands) IsEmpty() bool {
	return len(v.Build) == 0 && len(v.Test) == 0 && len(v.Lint) == 0 &&
		len(v.TypeCheck) == 0 && len(v.Format) == 0
}

// All returns a flattened slice of all verification commands across categories.
func (v VerificationCommands) All() []string {
	var all []string
	all = append(all, v.Build...)
	all = append(all, v.Test...)
	all = append(all, v.Lint...)
	all = append(all, v.TypeCheck...)
	all = append(all, v.Format...)
	return all
}

// ManifestFileInfo describes a dependency manifest file and its properties.
type ManifestFileInfo struct {
	Path           string         `yaml:"path"              json:"path"`
	Ecosystem      string         `yaml:"ecosystem"         json:"ecosystem"`
	VSSupported    bool           `yaml:"vs_supported"      json:"vs_supported"`
	LockFile       string         `yaml:"lock_file"         json:"lock_file"`
	LockFilePolicy LockFilePolicy `yaml:"lock_file_policy"  json:"lock_file_policy"`
}

// LockFilePolicy categorizes the lock file enforcement level for an ecosystem.
type LockFilePolicy int

const (
	LockFilePolicyRequired    LockFilePolicy = iota // Lock file must be committed
	LockFilePolicyRecommended                       // Lock file should be committed
	LockFilePolicyNone                              // No lock file mechanism
)

var lockFilePolicyNames = [...]string{
	LockFilePolicyRequired:    "required",
	LockFilePolicyRecommended: "recommended",
	LockFilePolicyNone:        "none",
}

func (p LockFilePolicy) String() string {
	if int(p) < len(lockFilePolicyNames) {
		return lockFilePolicyNames[p]
	}
	return "unknown"
}

func (p LockFilePolicy) MarshalText() ([]byte, error) {
	s := p.String()
	if s == "unknown" {
		return nil, fmt.Errorf("cannot marshal unknown LockFilePolicy value %d", int(p))
	}
	return []byte(s), nil
}

func (p *LockFilePolicy) UnmarshalText(text []byte) error {
	for i, name := range lockFilePolicyNames {
		if name == string(text) {
			*p = LockFilePolicy(i)
			return nil
		}
	}
	return fmt.Errorf("unknown lock file policy: %q", string(text))
}
