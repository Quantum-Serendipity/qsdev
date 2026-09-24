package types

import "slices"

// Schema version constants for .qsdev.yaml configuration files.
//
// Version 2 split the v1 `profile` key, which held the infrastructure profile
// name when written by init, into `infra_profile` (infrastructure profile) and
// `profile` (project-type profile). Version 1 files still load: the parser
// migrates them in memory and the next write records version 2.
const (
	ConfigVersionMin     = 1
	ConfigVersionMax     = 2
	ConfigVersionCurrent = 2
)

// QsdevConfig represents the parsed contents of a .qsdev.yaml file.
// It is the declarative project configuration that drives devinit behavior.
type QsdevConfig struct {
	Version      int    `yaml:"version"`
	QsdevVersion string `yaml:"qsdev_version,omitempty"`
	Tier         string `yaml:"tier,omitempty"`
	// Profile is the project-type profile (go-web, ts-fullstack, ...) the
	// project was created from; `qsdev init --profile` records it.
	Profile string `yaml:"profile,omitempty"`
	// InfraProfile is the infrastructure profile (consulting-default,
	// startup-github, enterprise) that selects the generated CI, dependency
	// update and security docs; `qsdev init --infra-profile` records it.
	InfraProfile   string           `yaml:"infra_profile,omitempty"`
	Languages      []LanguageConfig `yaml:"languages,omitempty"`
	Services       []ServiceConfig  `yaml:"services,omitempty"`
	Packages       []string         `yaml:"packages,omitempty"` // extra nixpkgs attribute paths (devenv add-package)
	Overlays       []string         `yaml:"overlays,omitempty"` // Nix overlay files (devenv add-overlay)
	Security       SecurityConfig   `yaml:"security,omitempty"`
	Tools          ToolsConfig      `yaml:"tools,omitempty"`
	ClaudeCode     ClaudeCodeConfig `yaml:"claude_code,omitempty"`
	Hooks          HooksConfig      `yaml:"hooks,omitempty"`
	Infrastructure InfraConfig      `yaml:"infrastructure,omitempty"`
	Client         *ClientConfig    `yaml:"client,omitempty"`
	Git            GitConfig        `yaml:"git,omitempty"`
	MCP            MCPConfig        `yaml:"mcp,omitempty"`
	Java           JavaConfig       `yaml:"java,omitempty"`
}

// LanguageConfig specifies a language/platform ecosystem in .qsdev.yaml.
type LanguageConfig struct {
	Name           string `yaml:"name"`
	Version        string `yaml:"version,omitempty"`
	PackageManager string `yaml:"package_manager,omitempty"`
}

// ServiceConfig specifies a development service (database, cache, etc.) in .qsdev.yaml.
type ServiceConfig struct {
	Name    string            `yaml:"name"`
	Version string            `yaml:"version,omitempty"`
	Options map[string]string `yaml:"options,omitempty"`
}

// SecurityConfig holds security posture settings in .qsdev.yaml.
type SecurityConfig struct {
	Level           string `yaml:"level,omitempty"`
	AgeGating       *bool  `yaml:"age_gating,omitempty"`
	ScriptBlocking  *bool  `yaml:"script_blocking,omitempty"`
	LockEnforcement *bool  `yaml:"lock_enforcement,omitempty"`
	VulnScanning    *bool  `yaml:"vuln_scanning,omitempty"`
	// CredentialVend opts the MCP server's qsdev_credential_vend tool in and
	// allow-lists what it may vend. The zero value (the default) leaves the
	// tool unmounted.
	CredentialVend CredentialVendConfig `yaml:"credential_vend,omitempty"`
}

// CredentialVendConfig is security.credential_vend in .qsdev.yaml: the opt-in
// and allow-lists for qsdev_credential_vend, which exchanges the host's
// ambient cloud identity for short-lived credentials. Nothing is vended unless
// Enabled is true, and then only for an identity a provider section lists.
type CredentialVendConfig struct {
	Enabled bool                      `yaml:"enabled,omitempty"`
	AWS     AWSCredentialVendConfig   `yaml:"aws,omitempty"`
	GCP     GCPCredentialVendConfig   `yaml:"gcp,omitempty"`
	Azure   AzureCredentialVendConfig `yaml:"azure,omitempty"`
}

// IsZero reports whether no credential_vend key is set.
func (c CredentialVendConfig) IsZero() bool {
	return !c.Enabled && c.AWS.IsZero() && c.GCP.IsZero() && c.Azure.IsZero()
}

// Clone returns a deep copy of c.
func (c CredentialVendConfig) Clone() CredentialVendConfig {
	out := c
	out.AWS.RoleARNs = slices.Clone(c.AWS.RoleARNs)
	out.GCP.ServiceAccounts = slices.Clone(c.GCP.ServiceAccounts)
	out.Azure.Scopes = slices.Clone(c.Azure.Scopes)
	out.Azure.Identities = slices.Clone(c.Azure.Identities)
	return out
}

// AWSCredentialVendConfig allow-lists AWS STS vending.
type AWSCredentialVendConfig struct {
	// RoleARNs are the roles an AssumeRole request may name (exact match).
	RoleARNs []string `yaml:"role_arns,omitempty"`
	// AllowSessionToken permits a request without role_arn, which calls
	// GetSessionToken and returns credentials carrying the full permissions
	// of the ambient IAM user. Denied by default.
	AllowSessionToken bool `yaml:"allow_session_token,omitempty"`
}

// IsZero reports whether no aws key is set.
func (c AWSCredentialVendConfig) IsZero() bool {
	return len(c.RoleARNs) == 0 && !c.AllowSessionToken
}

// GCPCredentialVendConfig allow-lists GCP service-account impersonation.
type GCPCredentialVendConfig struct {
	// ServiceAccounts are the service accounts (email or numeric unique ID) a
	// request may impersonate.
	ServiceAccounts []string `yaml:"service_accounts,omitempty"`
}

// IsZero reports whether no gcp key is set.
func (c GCPCredentialVendConfig) IsZero() bool { return len(c.ServiceAccounts) == 0 }

// AzureCredentialVendConfig allow-lists Azure Managed Identity tokens.
type AzureCredentialVendConfig struct {
	// Scopes are the token scopes (audiences) a request may ask for; the
	// tool's default scope must be listed to be used.
	Scopes []string `yaml:"scopes,omitempty"`
	// Identities are the user-assigned managed-identity client IDs a request
	// may select. A request naming no identity uses the system-assigned one.
	Identities []string `yaml:"identities,omitempty"`
}

// IsZero reports whether no azure key is set.
func (c AzureCredentialVendConfig) IsZero() bool {
	return len(c.Scopes) == 0 && len(c.Identities) == 0
}

// ToolsConfig controls which optional tools are enabled/disabled in .qsdev.yaml.
type ToolsConfig struct {
	Enabled  []string                  `yaml:"enabled,omitempty"`
	Disabled []string                  `yaml:"disabled,omitempty"`
	Config   map[string]map[string]any `yaml:"config,omitempty"`
}

// MCPConfig holds settings for qsdev's own MCP server (`qsdev mcp serve`) in
// .qsdev.yaml.
type MCPConfig struct {
	// DisabledTools names MCP tools (qsdev_nix_run, qsdev_security_scan, ...)
	// the server's Guardrail refuses to run for every caller. The names are
	// MCP tool names, not qsdev catalog tools (tools.disabled); `qsdev check`
	// validates them against the tools the server can mount.
	DisabledTools []string `yaml:"disabled_tools,omitempty"`
}

// ClaudeCodeConfig holds Claude Code agent settings in .qsdev.yaml.
type ClaudeCodeConfig struct {
	Enabled         *bool    `yaml:"enabled,omitempty"`
	PermissionLevel string   `yaml:"permission_level,omitempty"`
	Skills          []string `yaml:"skills,omitempty"`
	MCPServers      []string `yaml:"mcp_servers,omitempty"`
}

// HooksConfig holds settings for the generated Claude Code hooks in
// .qsdev.yaml. It is team policy: only the committed file sets it, never
// .qsdev.local.yaml.
type HooksConfig struct {
	FileBoundary FileBoundaryConfig `yaml:"file_boundary,omitempty"`
	ToolGates    ToolGatesConfig    `yaml:"tool_gates,omitempty"`
}

// ToolGatesConfig is the policy the tool-gates hook enforces on every tool
// call. Entries are Claude Code tool names ("Bash", "WebFetch",
// "mcp__github__delete_repo"), where "*" matches any run of characters
// ("mcp__github__*"). A tool matching Denied is always blocked; when Allowed
// is non-empty, a tool matching none of its entries is blocked too. With both
// empty the hook has no policy and allows every tool.
type ToolGatesConfig struct {
	Allowed []string `yaml:"allowed,omitempty"`
	Denied  []string `yaml:"denied,omitempty"`
}

// HasPolicy reports whether c restricts any tool.
func (c ToolGatesConfig) HasPolicy() bool {
	return len(c.Allowed) > 0 || len(c.Denied) > 0
}

// FileBoundaryConfig configures the file-boundary hook.
type FileBoundaryConfig struct {
	// ExtraReadPaths are directories outside the project that the Read, Grep
	// and Glob tools may reach, in addition to the dependency caches the hook
	// always allows reading. Each is an absolute or ~/-relative path. They are
	// never writable through the hook.
	ExtraReadPaths []string `yaml:"extra_read_paths,omitempty"`
}

// Clone returns a deep copy of c.
func (c HooksConfig) Clone() HooksConfig {
	out := c
	out.FileBoundary.ExtraReadPaths = slices.Clone(c.FileBoundary.ExtraReadPaths)
	out.ToolGates.Allowed = slices.Clone(c.ToolGates.Allowed)
	out.ToolGates.Denied = slices.Clone(c.ToolGates.Denied)
	return out
}

// InfraDisabled is the InfraConfig value that switches off a component an
// infrastructure profile would otherwise require (registry_proxy, nix_cache).
const InfraDisabled = "none"

// InfraConfig holds infrastructure settings (registry proxy, caches) in
// .qsdev.yaml. These are the organization's real endpoints; an explicit
// infra_profile selects the technology and requires the endpoints it needs
// (internal/profile InfraProfile.Resolve).
type InfraConfig struct {
	RegistryProxy          string            `yaml:"registry_proxy,omitempty"`
	RegistryProxyOverrides map[string]string `yaml:"registry_proxy_overrides,omitempty"`
	RegistryProxyPaths     map[string]string `yaml:"registry_proxy_paths,omitempty"`
	// NixCache is the binary cache substituter URL (a bare Cachix cache name
	// is also accepted when the infra profile uses Cachix).
	NixCache string `yaml:"nix_cache,omitempty"`
	// NixCachePublicKey is the cache's signing public key ("name-1:base64").
	NixCachePublicKey string `yaml:"nix_cache_public_key,omitempty"`
	// BuildCache names the shared build cache technology (e.g. "sccache").
	BuildCache string `yaml:"build_cache,omitempty"`
	// BuildCacheURL is the remote build cache endpoint (e.g. a self-hosted
	// Turborepo remote cache, exported as TURBO_API).
	BuildCacheURL string `yaml:"build_cache_url,omitempty"`
}

// RegistryProxyBase returns RegistryProxy, or "" when it is unset or
// InfraDisabled.
func (c InfraConfig) RegistryProxyBase() string {
	if c.RegistryProxy == InfraDisabled {
		return ""
	}
	return c.RegistryProxy
}

// NixCacheURL returns NixCache, or "" when it is unset or InfraDisabled.
func (c InfraConfig) NixCacheURL() string {
	if c.NixCache == InfraDisabled {
		return ""
	}
	return c.NixCache
}

// GitConfig holds git workflow settings in .qsdev.yaml.
type GitConfig struct {
	// BranchPattern is the POSIX extended regular expression the always-on
	// branch-naming pre-push hook checks the current branch against. Empty
	// selects the built-in default (gitworkflow.DefaultBranchPattern).
	BranchPattern string `yaml:"branch_pattern,omitempty"`
}

// JavaConfig holds JVM ecosystem settings in .qsdev.yaml.
type JavaConfig struct {
	// RepositoryAllowlist lists the ids of Maven repositories (declared in
	// pom.xml <repositories> or <pluginRepositories>) that Maven resolves
	// from their own URL. The generated .mvn/settings.xml mirror redirects
	// every other repository to Maven Central or the registry proxy.
	RepositoryAllowlist []string `yaml:"repository_allowlist,omitempty" json:"repository_allowlist,omitempty"`
}

// ClientConfig holds client-specific constraints in .qsdev.yaml.
type ClientConfig struct {
	Name               string   `yaml:"name"`
	Compliance         []string `yaml:"compliance,omitempty"`
	SecurityLevel      string   `yaml:"security_level,omitempty"`
	RegistryProxy      string   `yaml:"registry_proxy,omitempty"`
	NixCache           string   `yaml:"nix_cache,omitempty"`
	AllowedMCP         []string `yaml:"allowed_mcp_servers,omitempty"`
	BlockedMCP         []string `yaml:"blocked_mcp_servers,omitempty"`
	DataClassification string   `yaml:"data_classification,omitempty"`
}
