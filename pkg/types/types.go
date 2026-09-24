package types

import (
	"os"
	"slices"
	"time"
)

// WizardAnswers holds user selections and detected project state from the init wizard.
// It flows through generators to produce output files and is serialized to the
// per-addon answers file for incremental updates.
type WizardAnswers struct {
	ProjectName        string            `yaml:"project_name"     json:"project_name"`
	ProjectRoot        string            `yaml:"project_root"     json:"project_root"`
	Detected           DetectedProject   `yaml:"detected"         json:"detected"`
	Languages          []LanguageChoice  `yaml:"languages"        json:"languages"`
	Services           []ServiceChoice   `yaml:"services"         json:"services"`
	Direnv             bool              `yaml:"direnv"           json:"direnv"`
	GitHooks           []string          `yaml:"git_hooks"        json:"git_hooks"`
	ExtraPackages      []string          `yaml:"extra_packages"   json:"extra_packages"`
	EnvVars            map[string]string `yaml:"env_vars"         json:"env_vars"`
	ClaudeCode         bool              `yaml:"claude_code"         json:"claude_code"`
	NixHardeningGuide  bool              `yaml:"nix_hardening_guide" json:"nix_hardening_guide"`
	ProfileName        string            `yaml:"profile_name"        json:"profile_name"`
	ProjectTypeProfile string            `yaml:"project_type_profile" json:"project_type_profile"`
	Tier               string            `yaml:"tier,omitempty"          json:"tier,omitempty"`
	PermissionLevel    string            `yaml:"permission_level" json:"permission_level"`
	Skills             []string          `yaml:"skills"           json:"skills"`
	Hooks              HookChoices       `yaml:"hooks"            json:"hooks"`
	MCPServers         []string          `yaml:"mcp_servers"      json:"mcp_servers"`
	QuickChoice        string            `yaml:"quick_choice"     json:"quick_choice"`
	Confirmed          bool              `yaml:"confirmed"        json:"confirmed"`
	MergeMode          string            `yaml:"merge_mode"       json:"merge_mode"`
	AgentTools         AgentToolsAnswers `yaml:"agent_tools"      json:"agent_tools"`
	EnabledTools       map[string]bool   `yaml:"enabled_tools,omitempty" json:"enabled_tools,omitempty"`
	CIPlatform         string            `yaml:"ci_platform,omitempty"   json:"ci_platform,omitempty"`
	HookTier           string            `yaml:"hook_tier,omitempty"     json:"hook_tier,omitempty"`
	ConfigVersion      int               `yaml:"config_version,omitempty"   json:"config_version,omitempty"`
	ComplianceLevel    string            `yaml:"compliance_level,omitempty"  json:"compliance_level,omitempty"`
	ModelSize          string            `yaml:"model_size,omitempty"        json:"model_size,omitempty"`
	Infrastructure     InfraConfig       `yaml:"infrastructure"              json:"infrastructure"`
	Overlays           []string          `yaml:"overlays,omitempty"          json:"overlays,omitempty"`
	LSP                LSPSettings       `yaml:"lsp"                         json:"lsp"`
	// MCPPolicy is the committed client MCP policy; the Claude Code generator
	// drops every server it does not permit. It is refreshed from .qsdev.yaml
	// by init, join and update, never chosen interactively.
	MCPPolicy MCPPolicy `yaml:"mcp_policy,omitempty" json:"mcp_policy,omitempty"`
}

// AgentToolsAnswers holds AI agent tool selections from the wizard.
type AgentToolsAnswers struct {
	PostmortemEnabled    bool   `yaml:"postmortem_enabled"     json:"postmortem_enabled"`
	VersionSentinel      bool   `yaml:"version_sentinel"       json:"version_sentinel"`
	VersionSentinelHours int    `yaml:"version_sentinel_hours" json:"version_sentinel_hours"`
	SembleEnabled        bool   `yaml:"semble_enabled"         json:"semble_enabled"`
	SembleMode           string `yaml:"semble_mode"            json:"semble_mode"`
	SembleTextFiles      bool   `yaml:"semble_text_files"      json:"semble_text_files"`
}

// LSPSettings configures the generated LSP integration (Phase 31). LSP is
// "enabled" implicitly whenever any detected ecosystem maps to a default-on
// server (nixd is always-on, so in practice it is always enabled); these
// settings only tune the enforcement behavior of the lsp-first-guard hook.
type LSPSettings struct {
	// Enforcement controls the lsp-first-guard PreToolUse hook: "block" (the
	// default), "warn", or "off". An empty value is treated as "block".
	Enforcement string `yaml:"enforcement,omitempty" json:"enforcement,omitempty"`
}

// EnforcementTier returns the configured enforcement tier, defaulting to
// "block" when unset. Centralizing the default keeps generators consistent.
func (s LSPSettings) EnforcementTier() string {
	if s.Enforcement == "" {
		return "block"
	}
	return s.Enforcement
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

// DetectedProject is the result of scanning a project root for language markers,
// existing configuration files, and git state. It seeds WizardAnswers.Detected
// and drives auto-detection of ecosystems during init and join.
type DetectedProject struct {
	HasGoMod         bool   `yaml:"has_go_mod"       json:"has_go_mod"`
	GoVersion        string `yaml:"go_version"       json:"go_version"`
	HasPackageJSON   bool   `yaml:"has_package_json" json:"has_package_json"`
	NodeVersion      string `yaml:"node_version"     json:"node_version"`
	PackageManager   string `yaml:"package_manager"  json:"package_manager"`
	HasCargoToml     bool   `yaml:"has_cargo_toml"   json:"has_cargo_toml"`
	HasPyProject     bool   `yaml:"has_py_project"   json:"has_py_project"`
	PythonVersion    string `yaml:"python_version"   json:"python_version"`
	HasPomXML        bool   `yaml:"has_pom_xml"      json:"has_pom_xml"`
	HasBuildGradle   bool   `yaml:"has_build_gradle" json:"has_build_gradle"`
	HasCsproj        bool   `yaml:"has_csproj"       json:"has_csproj"`
	HasDockerfile    bool   `yaml:"has_dockerfile"      json:"has_dockerfile"`
	ContainerRuntime string `yaml:"container_runtime"   json:"container_runtime"`
	OSFamily         string `yaml:"os_family"            json:"os_family"`
	Username         string `yaml:"username"             json:"username"`
	HasTerraform     bool   `yaml:"has_terraform"       json:"has_terraform"`

	// Forward-compatible extensibility: new ecosystem modules can register
	// presence here without requiring struct changes.
	Ecosystems map[string]bool `yaml:"ecosystems" json:"ecosystems"`

	// Suggested holds each detected module's suggested configuration
	// (Version, PackageManager and module Extras as "key=value"), keyed by
	// module name. It carries what detection learned (Java build tool, uv,
	// OpenTofu, .NET SDK, TypeScript, ...) into LanguageChoice entries; see
	// WithSuggested.
	Suggested map[string]LanguageChoice `yaml:"suggested,omitempty" json:"suggested,omitempty"`

	// ProbableEcosystems marks ecosystems detected only from weak, generic
	// indicators (probable confidence). FillDefaults does not auto-enable
	// such tier 2+ ecosystems; they remain available as explicit choices.
	ProbableEcosystems map[string]bool `yaml:"probable_ecosystems,omitempty" json:"probable_ecosystems,omitempty"`

	HasDevenvNix      bool `yaml:"has_devenv_nix"      json:"has_devenv_nix"`
	HasDevenvYaml     bool `yaml:"has_devenv_yaml"     json:"has_devenv_yaml"`
	HasClaudeDir      bool `yaml:"has_claude_dir"      json:"has_claude_dir"`
	HasClaudeMd       bool `yaml:"has_claude_md"       json:"has_claude_md"`
	HasClaudeSettings bool `yaml:"has_claude_settings" json:"has_claude_settings"`
	HasEnvrc          bool `yaml:"has_envrc"           json:"has_envrc"`
	HasMcpJson        bool `yaml:"has_mcp_json"        json:"has_mcp_json"`

	IsGitRepo   bool `yaml:"is_git_repo"   json:"is_git_repo"`
	HasGitHooks bool `yaml:"has_git_hooks" json:"has_git_hooks"`
	// RemoteURL is the origin remote with any embedded credentials removed
	// (see RedactURLCredentials); it is persisted to the answers file.
	RemoteURL string `yaml:"remote_url" json:"remote_url"`
}

// NewDetectedProject returns a DetectedProject with all maps initialized.
func NewDetectedProject() DetectedProject {
	return DetectedProject{
		Ecosystems: make(map[string]bool),
		Suggested:  make(map[string]LanguageChoice),
	}
}

// HookChoices represents the user's selections for Claude Code automation hooks.
type HookChoices struct {
	AutoFormat            bool `yaml:"auto_format"             json:"auto_format"`
	SafetyBlock           bool `yaml:"safety_block"            json:"safety_block"`
	PreCommit             bool `yaml:"pre_commit"              json:"pre_commit"`
	AuditLog              bool `yaml:"audit_log"               json:"audit_log"`
	CredentialScan        bool `yaml:"credential_scan"         json:"credential_scan"`
	DestructivePrevention bool `yaml:"destructive_prevention"  json:"destructive_prevention"`
	SOC2Audit             bool `yaml:"soc2_audit"              json:"soc2_audit"`
	FileBoundary          bool `yaml:"file_boundary"           json:"file_boundary"`
	ToolGates             bool `yaml:"tool_gates"              json:"tool_gates"`
	SandboxEnabled        bool `yaml:"sandbox_enabled"         json:"sandbox_enabled"`
	SecurityEnforcement   bool `yaml:"security_enforcement"    json:"security_enforcement"`
	SelfProtection        bool `yaml:"self_protection"         json:"self_protection"`
}

// GeneratedFile represents a single file to be written by the generation pipeline.
// Generators return slices of these; the pipeline handles atomicity, merge
// strategies, and permission modes.
type GeneratedFile struct {
	Path           string        `yaml:"path"            json:"path"`
	Content        []byte        `yaml:"content"         json:"content"`
	Mode           os.FileMode   `yaml:"mode"            json:"mode"`
	Strategy       MergeStrategy `yaml:"strategy"        json:"strategy"`
	SkipValidation bool          `yaml:"skip_validation" json:"skip_validation"`
	Owner          string        `yaml:"owner,omitempty" json:"owner,omitempty"`
	// BaseContent, when non-nil, is the generator's own output for a file
	// whose Content holds the merged bytes written to disk. State recording
	// keeps it as the three-way merge base instead of Content. Generators
	// leave it nil.
	BaseContent []byte `yaml:"-" json:"-"`
}

// McpServerState tracks the lifecycle state of an installed MCP server.
type McpServerState struct {
	InstalledVersion  string     `yaml:"installed_version,omitempty"  json:"installed_version,omitempty"`
	InstallMethod     string     `yaml:"install_method,omitempty"     json:"install_method,omitempty"`
	LastHealthCheck   *time.Time `yaml:"last_health_check,omitempty"  json:"last_health_check,omitempty"`
	LastHealthStatus  string     `yaml:"last_health_status,omitempty" json:"last_health_status,omitempty"`
	DataCorpusVersion string     `yaml:"data_corpus_version,omitempty" json:"data_corpus_version,omitempty"`
}

// GeneratedState tracks what files were generated and their hashes,
// enabling modification detection on subsequent runs.
type GeneratedState struct {
	LastRun             time.Time                        `yaml:"last_run"              json:"last_run"`
	QsdevVersion        string                           `yaml:"qsdev_version,omitempty" json:"qsdev_version,omitempty"`
	Files               map[string]FileState             `yaml:"files"                 json:"files"`
	TemplateVersion     string                           `yaml:"template_version"      json:"template_version"`
	SkillLibraryVersion string                           `yaml:"skill_library_version" json:"skill_library_version"`
	EnabledTools        map[string]bool                  `yaml:"enabled_tools,omitempty" json:"enabled_tools,omitempty"`
	Fragments           map[string][]FragmentLedgerEntry `yaml:"fragments,omitempty"     json:"fragments,omitempty"`
	McpServers          map[string]McpServerState        `yaml:"mcp_servers,omitempty"    json:"mcp_servers,omitempty"`
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
	if a.ClaudeCode && a.PermissionLevel == "" && a.Tier == "" {
		return false
	}
	return true
}

// DefaultsProvider supplies catalog-driven default values for FillDefaults.
// This interface decouples pkg/types from internal/catalog, allowing tests to
// inject mock defaults and breaking the architectural inversion.
type DefaultsProvider interface {
	DefaultPostmortem() bool
	DefaultVersionSentinel() bool
	DefaultVersionSentinelHours() int
	DefaultSembleEnabled() bool
	DefaultSembleMode() string
	DefaultMCPServers() []string
	DefaultTier() string
	TierCompliance(tier string) string
	TierEnabledTools(tier string) []string
}

// FillDefaults populates empty user-configurable fields from detection results
// and catalog-driven defaults. The Detected field is always set from the
// provided detection results regardless of prior value.
func (a *WizardAnswers) FillDefaults(detected DetectedProject, defaults DefaultsProvider) {
	if a.EnvVars == nil {
		a.EnvVars = make(map[string]string)
	}
	a.Detected = detected
	// Fill languages from detection if none set.
	if len(a.Languages) == 0 {
		a.Languages = detected.LanguageChoices()
	}

	// Merge detected configuration into language entries: versions from the
	// dedicated fields, then every module's suggested version, package
	// manager and extras for whatever the entry does not already set.
	for i := range a.Languages {
		if a.Languages[i].Version == "" {
			switch a.Languages[i].Name {
			case "go":
				a.Languages[i].Version = detected.GoVersion
			case "javascript":
				a.Languages[i].Version = detected.NodeVersion
				if a.Languages[i].PackageManager == "" {
					a.Languages[i].PackageManager = detected.PackageManager
				}
			case "python":
				a.Languages[i].Version = detected.PythonVersion
			}
		}
		a.Languages[i] = detected.WithSuggested(a.Languages[i])
	}

	// The tier is resolved before any permission default, so a create without
	// --tier is exactly a create with --tier <default>: the tier's preset
	// applies and no permission level is recorded that the tier did not imply.
	a.resolveTier(defaults)

	// Only a provider without a default tier leaves the tier unset; the
	// standard preset then applies. Filling it whenever a tier is set would
	// mask the tier's own preset.
	if a.ClaudeCode && a.PermissionLevel == "" && a.Tier == "" {
		a.PermissionLevel = "standard"
	}

	a.ApplyClaudeHookDefaults()

	// Tier-derived posture applies to every tier, including supply-chain-only,
	// so the recorded compliance level always matches the selected tier.
	a.deriveFromTier(defaults)

	if a.Tier == supplyChainOnly {
		return
	}

	// Default agent tools when Claude is enabled — only when no source set
	// them at all. Every configuring source (flags, wizard, saved answers)
	// also records a mode or window, so an all-false selection is an explicit
	// opt-out and must survive; only the zero value means "unconfigured".
	if a.ClaudeCode && a.AgentTools == (AgentToolsAnswers{}) {
		a.AgentTools.PostmortemEnabled = defaults.DefaultPostmortem()
		a.AgentTools.VersionSentinel = defaults.DefaultVersionSentinel()
		a.AgentTools.SembleEnabled = defaults.DefaultSembleEnabled()
	}
	if a.ClaudeCode {
		if a.AgentTools.VersionSentinelHours == 0 {
			a.AgentTools.VersionSentinelHours = defaults.DefaultVersionSentinelHours()
		}
		if a.AgentTools.SembleMode == "" {
			a.AgentTools.SembleMode = defaults.DefaultSembleMode()
		}
	}

	// Default MCP servers when Claude Code is enabled and none are configured.
	if a.ClaudeCode && len(a.MCPServers) == 0 {
		a.MCPServers = append(a.MCPServers, defaults.DefaultMCPServers()...)
	}
}

// SembleMCPServer is the MCP server provided by the semble agent tool.
const SembleMCPServer = "semble"

// ConfiguredMCPServers returns the MCP servers to configure: MCPServers, with
// the semble server present exactly when the semble agent tool is enabled in a
// mode that uses its MCP server. AgentTools.SembleEnabled is semble's single
// source of truth, so a stale MCPServers entry (e.g. from an older default
// server list) cannot keep launching a server the answers record as disabled,
// and enabling semble always provisions its server.
func (a *WizardAnswers) ConfiguredMCPServers() []string {
	servers := make([]string, 0, len(a.MCPServers)+1)
	for _, s := range a.MCPServers {
		if s != SembleMCPServer {
			servers = append(servers, s)
		}
	}
	if a.AgentTools.SembleEnabled {
		switch a.AgentTools.SembleMode {
		case "", "mcp", "both":
			servers = append(servers, SembleMCPServer)
		}
	}
	return servers
}

// ApplyClaudeHookDefaults enforces the hook invariants every Claude Code
// configuration must carry, whichever path produced the answers: the
// self-protection hook is always on, and the package-guard safety block is
// enabled when no other primary hook was chosen. It is a no-op when Claude
// Code is disabled.
func (a *WizardAnswers) ApplyClaudeHookDefaults() {
	if !a.ClaudeCode {
		return
	}
	a.Hooks.SelfProtection = true
	if !a.Hooks.SafetyBlock && !a.Hooks.AutoFormat && !a.Hooks.PreCommit && !a.Hooks.AuditLog {
		a.Hooks.SafetyBlock = true
	}
}

// supplyChainOnly names both the lowest tier and its permission preset.
const supplyChainOnly = "supply-chain-only"

// resolveTier settles the tier once, so it is always persisted and no reader
// has to infer it: an unset tier becomes the supply-chain-only tier when that
// permission level was chosen, else the catalog's default tier.
func (a *WizardAnswers) resolveTier(defaults DefaultsProvider) {
	if a.Tier != "" {
		return
	}
	if a.PermissionLevel == supplyChainOnly {
		a.Tier = supplyChainOnly
		return
	}
	a.Tier = defaults.DefaultTier()
}

// deriveFromTier fills ComplianceLevel and EnabledTools from the selected
// Tier when they are not explicitly set.
func (a *WizardAnswers) deriveFromTier(defaults DefaultsProvider) {
	if a.Tier == "" {
		return
	}
	if a.ComplianceLevel == "" {
		a.ComplianceLevel = defaults.TierCompliance(a.Tier)
	}
	if a.EnabledTools == nil {
		if tools := defaults.TierEnabledTools(a.Tier); len(tools) > 0 {
			a.EnabledTools = make(map[string]bool, len(tools))
			for _, t := range tools {
				a.EnabledTools[t] = true
			}
		}
	}
}

// appendUnique appends val to slice only if it is not already present.
func appendUnique(slice []string, val string) []string {
	if slices.Contains(slice, val) {
		return slice
	}
	return append(slice, val)
}
