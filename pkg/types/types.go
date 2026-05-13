package types

import (
	"os"
	"time"
)

// WizardAnswers holds all user selections from the init wizard.
// It flows from the wizard to generators and is the single source of truth
// for what the user wants configured.
type WizardAnswers struct {
	ProjectName     string            `yaml:"project_name"     json:"project_name"`
	ProjectRoot     string            `yaml:"project_root"     json:"project_root"`
	Detected        DetectedProject   `yaml:"detected"         json:"detected"`
	Languages       []LanguageChoice  `yaml:"languages"        json:"languages"`
	Services        []ServiceChoice   `yaml:"services"         json:"services"`
	Direnv          bool              `yaml:"direnv"           json:"direnv"`
	GitHooks        []string          `yaml:"git_hooks"        json:"git_hooks"`
	ExtraPackages   []string          `yaml:"extra_packages"   json:"extra_packages"`
	EnvVars         map[string]string `yaml:"env_vars"         json:"env_vars"`
	ClaudeCode         bool              `yaml:"claude_code"         json:"claude_code"`
	NixHardeningGuide  bool              `yaml:"nix_hardening_guide" json:"nix_hardening_guide"`
	ProfileName        string            `yaml:"profile_name"        json:"profile_name"`
	ProjectTypeProfile string            `yaml:"project_type_profile" json:"project_type_profile"`
	PermissionLevel string            `yaml:"permission_level" json:"permission_level"`
	Skills          []string          `yaml:"skills"           json:"skills"`
	Hooks           HookChoices       `yaml:"hooks"            json:"hooks"`
	MCPServers      []string          `yaml:"mcp_servers"      json:"mcp_servers"`
	QuickChoice     string            `yaml:"quick_choice"     json:"quick_choice"`
	Confirmed       bool              `yaml:"confirmed"        json:"confirmed"`
	MergeMode       string            `yaml:"merge_mode"       json:"merge_mode"`
	AgentTools      AgentToolsAnswers `yaml:"agent_tools"      json:"agent_tools"`
	EnabledTools    map[string]bool   `yaml:"enabled_tools,omitempty" json:"enabled_tools,omitempty"`
	CIPlatform      string            `yaml:"ci_platform,omitempty"   json:"ci_platform,omitempty"`
	HookTier        string            `yaml:"hook_tier,omitempty"     json:"hook_tier,omitempty"`
}

// AgentToolsAnswers holds AI agent tool selections from the wizard.
type AgentToolsAnswers struct {
	PostmortemEnabled    bool   `yaml:"postmortem_enabled"     json:"postmortem_enabled"`
	VersionSentinel     bool   `yaml:"version_sentinel"       json:"version_sentinel"`
	VersionSentinelHours int   `yaml:"version_sentinel_hours" json:"version_sentinel_hours"`
	SembleEnabled       bool   `yaml:"semble_enabled"         json:"semble_enabled"`
	SembleMode          string `yaml:"semble_mode"            json:"semble_mode"`
	SembleTextFiles     bool   `yaml:"semble_text_files"      json:"semble_text_files"`
}

// LanguageChoice represents a user's selection of a programming language
// and its configuration for the development environment.
type LanguageChoice struct {
	Name           string   `yaml:"name"            json:"name"`
	Version        string   `yaml:"version"         json:"version"`
	PackageManager string   `yaml:"package_manager" json:"package_manager"`
	Extras         []string `yaml:"extras"          json:"extras"`
}

// ServiceChoice represents a user's selection of a development service
// (database, cache, queue, etc.) with its configuration.
type ServiceChoice struct {
	Name     string            `yaml:"name"     json:"name"`
	Version  string            `yaml:"version"  json:"version"`
	Settings map[string]string `yaml:"settings" json:"settings"`
}

// DetectedProject holds the results of scanning a project directory
// for language markers, existing config files, and git state.
type DetectedProject struct {
	HasGoMod       bool   `yaml:"has_go_mod"       json:"has_go_mod"`
	GoVersion      string `yaml:"go_version"       json:"go_version"`
	HasPackageJSON bool   `yaml:"has_package_json" json:"has_package_json"`
	NodeVersion    string `yaml:"node_version"     json:"node_version"`
	PackageManager string `yaml:"package_manager"  json:"package_manager"`
	HasCargoToml   bool   `yaml:"has_cargo_toml"   json:"has_cargo_toml"`
	HasPyProject   bool   `yaml:"has_py_project"   json:"has_py_project"`
	PythonVersion  string `yaml:"python_version"   json:"python_version"`
	HasPomXML      bool   `yaml:"has_pom_xml"      json:"has_pom_xml"`
	HasBuildGradle bool   `yaml:"has_build_gradle" json:"has_build_gradle"`
	HasCsproj      bool   `yaml:"has_csproj"       json:"has_csproj"`
	HasDockerfile  bool   `yaml:"has_dockerfile"   json:"has_dockerfile"`
	HasTerraform   bool   `yaml:"has_terraform"    json:"has_terraform"`

	// Forward-compatible extensibility: new ecosystem modules can register
	// presence here without requiring struct changes.
	Ecosystems map[string]bool `yaml:"ecosystems" json:"ecosystems"`

	HasDevenvNix      bool `yaml:"has_devenv_nix"      json:"has_devenv_nix"`
	HasDevenvYaml     bool `yaml:"has_devenv_yaml"     json:"has_devenv_yaml"`
	HasClaudeDir      bool `yaml:"has_claude_dir"      json:"has_claude_dir"`
	HasClaudeMd       bool `yaml:"has_claude_md"       json:"has_claude_md"`
	HasClaudeSettings bool `yaml:"has_claude_settings" json:"has_claude_settings"`
	HasEnvrc          bool `yaml:"has_envrc"           json:"has_envrc"`
	HasMcpJson        bool `yaml:"has_mcp_json"        json:"has_mcp_json"`

	IsGitRepo   bool   `yaml:"is_git_repo"   json:"is_git_repo"`
	HasGitHooks bool   `yaml:"has_git_hooks" json:"has_git_hooks"`
	RemoteURL   string `yaml:"remote_url"    json:"remote_url"`
}

// NewDetectedProject returns a DetectedProject with all maps initialized.
func NewDetectedProject() DetectedProject {
	return DetectedProject{
		Ecosystems: make(map[string]bool),
	}
}

// HookChoices represents the user's selections for Claude Code automation hooks.
type HookChoices struct {
	AutoFormat  bool `yaml:"auto_format"  json:"auto_format"`
	SafetyBlock bool `yaml:"safety_block" json:"safety_block"`
	PreCommit   bool `yaml:"pre_commit"   json:"pre_commit"`
	AuditLog    bool `yaml:"audit_log"    json:"audit_log"`
}

// GeneratedFile represents a single file produced by a generator,
// ready to be written to disk by the generation pipeline.
type GeneratedFile struct {
	Path           string        `yaml:"path"            json:"path"`
	Content        []byte        `yaml:"content"         json:"content"`
	Mode           os.FileMode   `yaml:"mode"            json:"mode"`
	Strategy       MergeStrategy `yaml:"strategy"        json:"strategy"`
	SkipValidation bool          `yaml:"skip_validation" json:"skip_validation"`
	Owner          string        `yaml:"owner,omitempty" json:"owner,omitempty"`
}

// GeneratedState tracks what files were generated and their hashes,
// enabling modification detection on subsequent runs.
type GeneratedState struct {
	LastRun             time.Time            `yaml:"last_run"              json:"last_run"`
	Files               map[string]FileState `yaml:"files"                 json:"files"`
	TemplateVersion     string               `yaml:"template_version"      json:"template_version"`
	SkillLibraryVersion string               `yaml:"skill_library_version" json:"skill_library_version"`
	EnabledTools        map[string]bool      `yaml:"enabled_tools,omitempty" json:"enabled_tools,omitempty"`
}

// FileState tracks a single generated file's hash and merge strategy.
type FileState struct {
	Hash        string        `yaml:"hash"         json:"hash"`
	Strategy    MergeStrategy `yaml:"strategy"      json:"strategy"`
	Mode        os.FileMode   `yaml:"mode"          json:"mode"`
	BaseContent []byte        `yaml:"base_content,omitempty" json:"base_content,omitempty"`
	Owner       string        `yaml:"owner,omitempty"        json:"owner,omitempty"`
}

// Generator is the interface that devenv and claudecode addons implement
// to produce files from wizard answers.
type Generator interface {
	Generate(answers WizardAnswers) ([]GeneratedFile, error)
}

// IsComplete returns true when the answers have enough information for
// generators to produce all files without user input.
func (a *WizardAnswers) IsComplete() bool {
	if !a.Confirmed {
		return false
	}
	if len(a.Languages) == 0 {
		return false
	}
	if a.ClaudeCode && a.PermissionLevel == "" {
		return false
	}
	return true
}

// FillDefaults populates empty fields from detection results and hardcoded
// defaults. It does NOT overwrite fields that already have values.
func (a *WizardAnswers) FillDefaults(detected DetectedProject) {
	if a.EnvVars == nil {
		a.EnvVars = make(map[string]string)
	}
	a.Detected = detected
	// Fill languages from detection if none set.
	if len(a.Languages) == 0 {
		if detected.HasGoMod {
			a.Languages = append(a.Languages, LanguageChoice{Name: "go", Version: detected.GoVersion})
		}
		if detected.HasPackageJSON {
			a.Languages = append(a.Languages, LanguageChoice{Name: "javascript", Version: detected.NodeVersion, PackageManager: detected.PackageManager})
		}
		if detected.HasPyProject {
			a.Languages = append(a.Languages, LanguageChoice{Name: "python", Version: detected.PythonVersion})
		}
		if detected.HasCargoToml {
			a.Languages = append(a.Languages, LanguageChoice{Name: "rust"})
		}
		if detected.HasPomXML || detected.HasBuildGradle {
			a.Languages = append(a.Languages, LanguageChoice{Name: "java"})
		}
		if detected.HasCsproj {
			a.Languages = append(a.Languages, LanguageChoice{Name: "dotnet"})
		}
		if detected.HasDockerfile {
			a.Languages = append(a.Languages, LanguageChoice{Name: "docker"})
		}
		if detected.HasTerraform {
			a.Languages = append(a.Languages, LanguageChoice{Name: "terraform"})
		}
	}

	// Default permission level.
	if a.ClaudeCode && a.PermissionLevel == "" {
		a.PermissionLevel = "standard"
	}

	// Default hooks when Claude is enabled.
	if a.ClaudeCode && !a.Hooks.SafetyBlock && !a.Hooks.AutoFormat && !a.Hooks.PreCommit && !a.Hooks.AuditLog {
		a.Hooks.SafetyBlock = true
	}

	// Default agent tools when Claude is enabled.
	if a.ClaudeCode {
		a.AgentTools.PostmortemEnabled = true
		if a.AgentTools.VersionSentinelHours == 0 {
			a.AgentTools.VersionSentinelHours = 24
		}
		if a.AgentTools.SembleMode == "" {
			a.AgentTools.SembleMode = "mcp"
		}
		if hasVSSupportedLanguage(a.Languages) {
			a.AgentTools.VersionSentinel = true
		}
		if pythonVersionAtLeast(detected.PythonVersion, 3, 10) {
			a.AgentTools.SembleEnabled = true
		}
	}

	// Default MCP servers when Claude Code is enabled and none are configured.
	if a.ClaudeCode && len(a.MCPServers) == 0 {
		a.MCPServers = append(a.MCPServers, "context7", "github")
	}
}

func hasVSSupportedLanguage(langs []LanguageChoice) bool {
	for _, l := range langs {
		switch l.Name {
		case "javascript", "python", "rust", "dotnet":
			return true
		}
	}
	return false
}

func pythonVersionAtLeast(version string, major, minor int) bool {
	if version == "" {
		return false
	}
	parts := splitVersion(version)
	if len(parts) < 2 {
		return false
	}
	if parts[0] > major {
		return true
	}
	return parts[0] == major && parts[1] >= minor
}

func splitVersion(v string) []int {
	var result []int
	for _, part := range splitOnDot(v) {
		n := 0
		for _, c := range part {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			} else {
				break
			}
		}
		result = append(result, n)
	}
	return result
}

func splitOnDot(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}
