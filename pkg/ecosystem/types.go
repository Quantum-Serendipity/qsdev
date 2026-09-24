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
	if int(c) >= 0 && int(c) < len(confidenceNames) {
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
	RegistryProxy  string            `yaml:"registry_proxy"  json:"registry_proxy"`
	// RepositoryAllowlist is .qsdev.yaml java.repository_allowlist: the ids
	// of project-declared Maven repositories that a module's generated
	// registry mirror must leave alone (resolved from their own URL).
	RepositoryAllowlist []string `yaml:"repository_allowlist,omitempty" json:"repository_allowlist,omitempty"`
	// IsolateCLIConfig is .qsdev.yaml cloud.isolate_cli_config: a cloud CLI
	// module points its CLI's configuration directory into the project.
	IsolateCLIConfig bool `yaml:"isolate_cli_config,omitempty" json:"isolate_cli_config,omitempty"`
}

// PM returns the configured PackageManager, falling back to defaultPM if empty.
func (c ModuleConfig) PM(defaultPM string) string {
	if c.PackageManager != "" {
		return c.PackageManager
	}
	return defaultPM
}

// Extra returns the value of the given Extras key, falling back to defaultVal
// if the key is absent, empty, or Extras is nil.
func (c ModuleConfig) Extra(key, defaultVal string) string {
	if c.Extras != nil {
		if v, ok := c.Extras[key]; ok && v != "" {
			return v
		}
	}
	return defaultVal
}

// DevenvInput represents a devenv.sh input (flake reference) to be added
// to devenv.yaml.
type DevenvInput struct {
	URL     string `yaml:"url"              json:"url"`
	Follows string `yaml:"follows,omitempty" json:"follows,omitempty"`
}

// HookConfig represents a pre-commit hook configuration entry.
//
// Types is an AND filter (a file must carry every listed identify tag), so a
// custom hook lists at most one tag there; alternatives go in TypesOr (files
// matching ANY listed tag). For BuiltIn hooks only TypesOr, ExcludeTypes and
// Settings are rendered, replacing the git-hooks.nix defaults (its other
// fields come from upstream).
type HookConfig struct {
	ID                     string   `yaml:"id"                        json:"id"`
	Name                   string   `yaml:"name"                      json:"name"`
	Description            string   `yaml:"description"               json:"description"`
	Entry                  string   `yaml:"entry"                     json:"entry"`
	Language               string   `yaml:"language"                  json:"language"`
	Types                  []string `yaml:"types"                     json:"types"`
	TypesOr                []string `yaml:"types_or,omitempty"        json:"types_or,omitempty"`
	ExcludeTypes           []string `yaml:"exclude_types,omitempty"   json:"exclude_types,omitempty"`
	Stages                 []string `yaml:"stages"                    json:"stages"`
	PassFilenames          bool     `yaml:"pass_filenames"            json:"pass_filenames"`
	Files                  string   `yaml:"files"                     json:"files"`
	AdditionalDependencies []string `yaml:"additional_dependencies"   json:"additional_dependencies"`
	BuiltIn                bool     `yaml:"built_in"                  json:"built_in"`
	NixPackage             string   `yaml:"nix_package,omitempty"     json:"nix_package,omitempty"`
	Excludes               []string `yaml:"excludes,omitempty"        json:"excludes,omitempty"` // Path regexes the hook skips (git-hooks.nix excludes); honored for built-in hooks too.
	// Settings sets git-hooks.nix `settings.<key>` string options of a
	// BuiltIn hook (e.g. binPath).
	Settings map[string]string `yaml:"settings,omitempty" json:"settings,omitempty"`
	// Script, when set, is a bash script run as the hook instead of Entry,
	// for checks that need logic around the tool (preconditions, clear
	// failure messages). Staged files arrive as "$@" when PassFilenames is
	// set, and NixPackage's bin directory is first on PATH.
	Script string `yaml:"script,omitempty" json:"script,omitempty"`
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
	if int(p) >= 0 && int(p) < len(ciPhaseNames) {
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

// PackageManagerInfo describes one of an ecosystem's package managers. CI
// lock-file enforcement and audit commands are not package manager metadata:
// they come from EcosystemModule.CICommands, which the generated CI workflow
// runs.
type PackageManagerInfo struct {
	Name           string `yaml:"name"            json:"name"`
	LockFile       string `yaml:"lock_file"       json:"lock_file"`
	InstallCommand string `yaml:"install_command" json:"install_command"`
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
	if int(f) >= 0 && int(f) < len(wizardFieldTypeNames) {
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
// contributes to the init wizard. The wizard renders a module's fields on
// their own screen when the module's language is selected, seeds each field
// from the language's current configuration (else Default) and records the
// answer back into it.
//
// Key names the ModuleConfig setting the answer is stored in and must be the
// key the module itself reads: types.SettingVersion ("version") for
// ModuleConfig.Version, types.SettingPackageManager ("package_manager") for
// ModuleConfig.PackageManager, and any other key for ModuleConfig.Extras[Key].
// Confirm answers are stored as "true" or "false", multi-select answers as a
// comma-separated list. Placeholder is the example shown in an empty
// FieldTypeInput.
type WizardField struct {
	Key         string          `yaml:"key"         json:"key"`
	Label       string          `yaml:"label"       json:"label"`
	Description string          `yaml:"description" json:"description"`
	Type        WizardFieldType `yaml:"type"        json:"type"`
	Options     []WizardOption  `yaml:"options"     json:"options"`
	Default     string          `yaml:"default"     json:"default"`
	Placeholder string          `yaml:"placeholder" json:"placeholder"`
	Required    bool            `yaml:"required"    json:"required"`
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
	if int(p) >= 0 && int(p) < len(lockFilePolicyNames) {
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

// DoctorCheck describes a single health check contributed by an ecosystem
// module. EnvCheck names an environment variable that must hold a real value;
// an empty or placeholder value counts as unset. Command is a live
// verification the user can run (for example a cloud CLI auth probe); the
// doctor never executes it, so Timeout only bounds a caller that opts to.
type DoctorCheck struct {
	Name        string `yaml:"name"        json:"name"`
	Description string `yaml:"description" json:"description"`
	Command     string `yaml:"command"     json:"command"`
	EnvCheck    string `yaml:"env_check"   json:"env_check"`
	Timeout     int    `yaml:"timeout"     json:"timeout"`
	Provider    string `yaml:"provider"    json:"provider"`
}

// ToolchainRequirement is a minimum tool version that a module's generated
// security setting depends on (see ToolchainRequirementProvider).
type ToolchainRequirement struct {
	// Binary is the executable looked up on PATH, e.g. "npm".
	Binary string
	// VersionArg is the argument that makes Binary print its version.
	VersionArg string
	// MinVersion is the oldest version that honours Setting, e.g. "11.10.0".
	MinVersion string
	// Setting names the generated setting that needs MinVersion, e.g.
	// ".npmrc min-release-age".
	Setting string
}
