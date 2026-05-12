package profile

import "strings"

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
	VulnScannerOSV  VulnScannerType = "osv"
	VulnScannerSnyk VulnScannerType = "snyk"
	VulnScannerGrype VulnScannerType = "grype"
	VulnScannerNone VulnScannerType = "none"
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

// RegistryConfig holds package-registry proxy settings.
type RegistryConfig struct {
	Type       RegistryType `yaml:"type"                  json:"type"`
	URL        string       `yaml:"url,omitempty"         json:"url,omitempty"`
	Ecosystems []string     `yaml:"ecosystems,omitempty"  json:"ecosystems,omitempty"`
	AuthEnvVar string       `yaml:"auth_env_var,omitempty" json:"auth_env_var,omitempty"`
}

// EcosystemURL returns the registry URL for a given ecosystem based on the
// registry type and base URL. Unsupported combinations return "".
func (r RegistryConfig) EcosystemURL(ecosystem string) string {
	eco := strings.ToLower(ecosystem)
	url := strings.TrimRight(r.URL, "/")

	switch r.Type {
	case RegistryArtifactory:
		if url == "" {
			return ""
		}
		switch eco {
		case "npm":
			return url + "/api/npm/npm-virtual/"
		case "pypi":
			return url + "/api/pypi/pypi-virtual/simple"
		case "go":
			return url + "/api/go/go-virtual"
		case "cargo":
			return "sparse+" + url + "/api/cargo/cargo-virtual/index/"
		case "maven":
			return url + "/maven-virtual"
		case "nuget":
			return url + "/api/nuget/nuget-virtual"
		}

	case RegistryNexus:
		if url == "" {
			return ""
		}
		switch eco {
		case "npm":
			return url + "/repository/npm-group/"
		case "pypi":
			return url + "/repository/pypi-group/simple"
		case "go":
			return url + "/repository/go-group/"
		case "maven":
			return url + "/repository/maven-group/"
		case "nuget":
			return url + "/repository/nuget-group/"
		}

	case RegistryGitHub:
		switch eco {
		case "npm":
			return "https://npm.pkg.github.com/"
		case "maven":
			return "https://maven.pkg.github.com/"
		}

	case RegistryNone:
		return ""
	}

	return ""
}

// NixCacheConfig holds Nix binary cache settings.
type NixCacheConfig struct {
	Type          NixCacheType `yaml:"type"                       json:"type"`
	URL           string       `yaml:"url,omitempty"              json:"url,omitempty"`
	PublicKey     string       `yaml:"public_key,omitempty"       json:"public_key,omitempty"`
	SigningKeyRef string       `yaml:"signing_key_ref,omitempty"  json:"signing_key_ref,omitempty"`
	CacheName     string       `yaml:"cache_name,omitempty"       json:"cache_name,omitempty"`
}

// BuildCacheConfig holds build cache settings.
type BuildCacheConfig struct {
	Type        BuildCacheType    `yaml:"type"                    json:"type"`
	Backend     string            `yaml:"backend,omitempty"       json:"backend,omitempty"`
	URL         string            `yaml:"url,omitempty"           json:"url,omitempty"`
	AuthEnvVars map[string]string `yaml:"auth_env_vars,omitempty" json:"auth_env_vars,omitempty"`
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
