package profile

import (
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// RegistryType identifies a package registry / proxy technology.
type RegistryType string

const (
	RegistryNexus          RegistryType = "nexus"
	RegistryArtifactory    RegistryType = "artifactory"
	RegistryGitHub         RegistryType = "github"
	RegistryGitLab         RegistryType = "gitlab"
	RegistryAWS            RegistryType = "aws"
	RegistryGCP            RegistryType = "gcp"
	RegistryAzure          RegistryType = "azure"
	RegistryVerdaccio      RegistryType = "verdaccio"
	RegistryArtifactKeeper RegistryType = "artifact-keeper"
	RegistryNone           RegistryType = "none"
)

// NixCacheType identifies a Nix binary cache technology.
type NixCacheType string

const (
	NixCacheCachix   NixCacheType = "cachix"
	NixCacheAttic    NixCacheType = "attic"
	NixCacheNixServe NixCacheType = "nix-serve"
	NixCacheNone     NixCacheType = "none"
)

// BuildCacheType identifies a build cache technology.
type BuildCacheType string

const (
	BuildCacheSccache     BuildCacheType = "sccache"
	BuildCacheCcache      BuildCacheType = "ccache"
	BuildCacheTurborepo   BuildCacheType = "turborepo"
	BuildCacheNx          BuildCacheType = "nx"
	BuildCacheBazelRemote BuildCacheType = "bazel-remote"
	BuildCacheNone        BuildCacheType = "none"
)

// VulnScannerType identifies a vulnerability scanner.
type VulnScannerType string

const (
	VulnScannerOSV   VulnScannerType = "osv"
	VulnScannerSnyk  VulnScannerType = "snyk"
	VulnScannerGrype VulnScannerType = "grype"
	VulnScannerNone  VulnScannerType = "none"
)

// BehavioralType identifies a behavioral analysis tool.
type BehavioralType string

const (
	BehavioralSocket BehavioralType = "socket"
	BehavioralNone   BehavioralType = "none"
)

// CIProtectionType identifies a CI protection tool.
type CIProtectionType string

const (
	CIProtectionHardenRunner CIProtectionType = "harden-runner"
	CIProtectionNone         CIProtectionType = "none"
)

// UpdateToolType identifies a dependency update tool.
type UpdateToolType string

const (
	UpdateToolRenovate   UpdateToolType = "renovate"
	UpdateToolDependabot UpdateToolType = "dependabot"
)

// SBOMGeneratorType identifies an SBOM generator.
type SBOMGeneratorType string

const (
	SBOMGeneratorSyft    SBOMGeneratorType = "syft"
	SBOMGeneratorSbomnix SBOMGeneratorType = "sbomnix"
	SBOMGeneratorNone    SBOMGeneratorType = "none"
)

// SBOMSigningType identifies an SBOM signing tool.
type SBOMSigningType string

const (
	SBOMSigningCosign SBOMSigningType = "cosign"
	SBOMSigningNone   SBOMSigningType = "none"
)

// InfraProfile encodes an organization's infrastructure choices: registry
// proxy, Nix cache, build cache, scanning, update policy, and SBOM config.
type InfraProfile struct {
	Name        string           `yaml:"name"                  json:"name"`
	Description string           `yaml:"description,omitempty" json:"description,omitempty"`
	Registry    RegistryConfig   `yaml:"registry"              json:"registry"`
	NixCache    NixCacheConfig   `yaml:"nix_cache"             json:"nix_cache"`
	BuildCache  BuildCacheConfig `yaml:"build_cache"           json:"build_cache"`
	Scanning    ScanningConfig   `yaml:"scanning"              json:"scanning"`
	Updates     UpdateConfig     `yaml:"updates"               json:"updates"`
	SBOM        SBOMConfig       `yaml:"sbom"                  json:"sbom"`
}

// RegistryConfig holds package-registry proxy settings. URL, Overrides and
// Paths are the project's real endpoints (.qsdev.yaml infrastructure:),
// applied by InfraProfile.Resolve; the built-in profiles carry none.
type RegistryConfig struct {
	Type       RegistryType `yaml:"type"                   json:"type"`
	URL        string       `yaml:"url,omitempty"          json:"url,omitempty"`
	Ecosystems []string     `yaml:"ecosystems,omitempty"   json:"ecosystems,omitempty"`
	AuthEnvVar string       `yaml:"auth_env_var,omitempty" json:"auth_env_var,omitempty"`
	// Overrides are per-ecosystem full URLs (infrastructure.registry_proxy_overrides).
	Overrides map[string]string `yaml:"overrides,omitempty" json:"overrides,omitempty"`
	// Paths are per-ecosystem path suffixes that replace the registry type's
	// layout (infrastructure.registry_proxy_paths).
	Paths map[string]string `yaml:"paths,omitempty" json:"paths,omitempty"`
}

// registryLayouts maps a pull-through proxy technology to the path, below
// its base URL, of the group/virtual repository serving each ecosystem.
// Ecosystems a layout omits fall back to ecosystem.DefaultProxyPaths.
var registryLayouts = map[RegistryType]map[string]string{
	RegistryArtifactory: {
		"npm":   "/api/npm/npm-virtual/",
		"pypi":  "/api/pypi/pypi-virtual/simple",
		"go":    "/api/go/go-virtual",
		"cargo": "/api/cargo/cargo-virtual/index/",
		"maven": "/maven-virtual",
		"nuget": "/api/nuget/v3/nuget-virtual/index.json",
	},
	RegistryNexus: {
		"npm":   "/repository/npm-group/",
		"pypi":  "/repository/pypi-group/simple",
		"go":    "/repository/go-group/",
		"maven": "/repository/maven-group/",
		"nuget": "/repository/nuget-group/index.json",
	},
}

// IsProxy reports whether the registry is a pull-through proxy that package
// installs are routed through. GitHub Packages is not: it hosts an
// organization's own packages, needs a token even to read, and does not serve
// the public registries, so qsdev routes nothing through it automatically.
func (r RegistryConfig) IsProxy() bool {
	return r.Type != "" && r.Type != RegistryNone && r.Type != RegistryGitHub
}

// Layout returns the path suffix per ecosystem key (as used by
// ecosystem.ProxyKeyForLanguage) for the ecosystems this registry serves:
// the registry type's layout, with gradle sharing maven's repository.
func (r RegistryConfig) Layout() map[string]string {
	layout := registryLayouts[r.Type]
	out := make(map[string]string, len(r.Ecosystems)+1)
	for _, eco := range r.Ecosystems {
		key := strings.ToLower(eco)
		if p, ok := layout[key]; ok {
			out[key] = p
			if key == "maven" {
				out["gradle"] = p
			}
		}
	}
	return out
}

// serves reports whether the registry routes the given ecosystem key.
func (r RegistryConfig) serves(eco string) bool {
	for _, e := range r.Ecosystems {
		e = strings.ToLower(e)
		if e == eco || (e == "maven" && eco == "gradle") {
			return true
		}
	}
	return false
}

// EcosystemURL returns the URL package installs for the ecosystem are routed
// to: a per-ecosystem override, else the base URL plus the configured or
// layout path (ecosystem.DefaultProxyPaths for ecosystems the layout lacks).
// It returns "" for a registry that is not a proxy, an ecosystem it does not
// serve, or when no endpoint is configured. This is the same resolution
// ecosystem.ToModuleConfigWithInfra applies to the effective Infrastructure.
func (r RegistryConfig) EcosystemURL(eco string) string {
	key := strings.ToLower(eco)
	if !r.IsProxy() || !r.serves(key) {
		return ""
	}
	return ecosystem.ResolveProxyURL(r.URL, r.Overrides, key, r.Paths, r.Layout())
}

// NixCacheConfig holds Nix binary cache settings. URL, CacheName and
// PublicKey are the project's real cache (infrastructure.nix_cache and
// nix_cache_public_key), applied by InfraProfile.Resolve.
type NixCacheConfig struct {
	Type      NixCacheType `yaml:"type"                 json:"type"`
	URL       string       `yaml:"url,omitempty"        json:"url,omitempty"`
	PublicKey string       `yaml:"public_key,omitempty" json:"public_key,omitempty"`
	CacheName string       `yaml:"cache_name,omitempty" json:"cache_name,omitempty"`
	// PushTokenEnvVar names the credential CI uses to push to the cache. It
	// is documented, never written into the developer environment.
	PushTokenEnvVar string `yaml:"push_token_env_var,omitempty" json:"push_token_env_var,omitempty"`
}

// BuildCacheConfig holds build cache settings.
type BuildCacheConfig struct {
	Type    BuildCacheType `yaml:"type"              json:"type"`
	Backend string         `yaml:"backend,omitempty" json:"backend,omitempty"`
	URL     string         `yaml:"url,omitempty"     json:"url,omitempty"`
	// AuthEnvVars name the credentials the cache backend reads from the
	// developer's environment; qsdev documents them and never sets them.
	AuthEnvVars []string `yaml:"auth_env_vars,omitempty" json:"auth_env_vars,omitempty"`
}

// ScanningConfig holds vulnerability and behavioral scanning settings.
type ScanningConfig struct {
	Vulnerability VulnScannerType  `yaml:"vulnerability"         json:"vulnerability"`
	Behavioral    BehavioralType   `yaml:"behavioral"            json:"behavioral"`
	CIProtection  CIProtectionType `yaml:"ci_protection"         json:"ci_protection"`
}

// UpdateConfig holds dependency update tool settings.
type UpdateConfig struct {
	Type               UpdateToolType `yaml:"type"                          json:"type"`
	AgeGatingDays      int            `yaml:"age_gating_days"               json:"age_gating_days"`
	EcosystemOverrides map[string]int `yaml:"ecosystem_overrides,omitempty" json:"ecosystem_overrides,omitempty"`
	AutomergePatches   bool           `yaml:"automerge_patches"             json:"automerge_patches"`
}

// SBOMConfig holds software bill-of-materials generation settings.
type SBOMConfig struct {
	Generator SBOMGeneratorType `yaml:"generator" json:"generator"`
	Signing   SBOMSigningType   `yaml:"signing"   json:"signing"`
}
