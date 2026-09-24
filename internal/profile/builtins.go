package profile

// The built-in profiles choose technologies, never endpoints: registry
// proxy, Nix cache and build cache URLs and keys are organization-specific,
// so InfraProfile.Resolve takes them from the project's infrastructure
// settings and refuses to apply a profile whose endpoints are missing or
// placeholders.

// ConsultingDefault is the zero-cost consulting-friendly profile.
// Uses Nexus for package proxying, Cachix for Nix, sccache/S3 for builds,
// OSV + Socket for scanning, Renovate with 3-day age gating, and Syft for SBOM.
var ConsultingDefault = &InfraProfile{
	Name:        "consulting-default",
	Description: "Consulting-friendly defaults ($0/mo): Nexus proxy, Cachix, sccache, OSV + Socket scanning, Renovate with 3-day age gate",
	Registry: RegistryConfig{
		Type:       RegistryNexus,
		Ecosystems: []string{"npm", "pypi", "maven", "go", "cargo"},
		AuthEnvVar: "NEXUS_TOKEN",
	},
	NixCache: NixCacheConfig{
		Type:            NixCacheCachix,
		PushTokenEnvVar: "CACHIX_AUTH_TOKEN",
	},
	BuildCache: BuildCacheConfig{
		Type:        BuildCacheSccache,
		Backend:     "s3",
		AuthEnvVars: []string{"SCCACHE_BUCKET", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"},
	},
	Scanning: ScanningConfig{
		Vulnerability: VulnScannerOSV,
		Behavioral:    BehavioralSocket,
		CIProtection:  CIProtectionHardenRunner,
	},
	Updates: UpdateConfig{
		Type:             UpdateToolRenovate,
		AgeGatingDays:    3,
		AutomergePatches: true,
	},
	SBOM: SBOMConfig{
		Generator: SBOMGeneratorSyft,
		Signing:   SBOMSigningNone,
	},
}

// StartupGitHub is a GitHub-native profile for startups.
// Uses GitHub Packages for the organization's own packages, Cachix for Nix,
// Turborepo for builds, OSV + Socket for scanning, Dependabot for updates,
// and Syft for SBOM.
var StartupGitHub = &InfraProfile{
	Name:        "startup-github",
	Description: "GitHub-native startup profile: GitHub Packages, Cachix, Turborepo, OSV + Socket scanning, Dependabot",
	Registry: RegistryConfig{
		Type:       RegistryGitHub,
		Ecosystems: []string{"npm", "maven"},
		AuthEnvVar: "GITHUB_TOKEN",
	},
	NixCache: NixCacheConfig{
		Type:            NixCacheCachix,
		PushTokenEnvVar: "CACHIX_AUTH_TOKEN",
	},
	BuildCache: BuildCacheConfig{
		Type:        BuildCacheTurborepo,
		AuthEnvVars: []string{"TURBO_TOKEN", "TURBO_TEAM"},
	},
	Scanning: ScanningConfig{
		Vulnerability: VulnScannerOSV,
		Behavioral:    BehavioralSocket,
		CIProtection:  CIProtectionHardenRunner,
	},
	Updates: UpdateConfig{
		Type:             UpdateToolDependabot,
		AgeGatingDays:    0,
		AutomergePatches: false,
	},
	SBOM: SBOMConfig{
		Generator: SBOMGeneratorSyft,
		Signing:   SBOMSigningNone,
	},
}

// Enterprise is the full enterprise profile.
// Uses Artifactory for package proxying (all ecosystems including NuGet),
// Cachix for Nix, sccache/S3 for builds, Snyk + Socket for scanning,
// Renovate with 7-day age gating, and Syft + Cosign for SBOM.
var Enterprise = &InfraProfile{
	Name:        "enterprise",
	Description: "Full enterprise profile: Artifactory, Cachix, sccache, Snyk + Socket scanning, Renovate with 7-day age gate, Cosign signing",
	Registry: RegistryConfig{
		Type:       RegistryArtifactory,
		Ecosystems: []string{"npm", "pypi", "maven", "go", "cargo", "nuget"},
		AuthEnvVar: "ARTIFACTORY_TOKEN",
	},
	NixCache: NixCacheConfig{
		Type:            NixCacheCachix,
		PushTokenEnvVar: "CACHIX_AUTH_TOKEN",
	},
	BuildCache: BuildCacheConfig{
		Type:        BuildCacheSccache,
		Backend:     "s3",
		AuthEnvVars: []string{"SCCACHE_BUCKET", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"},
	},
	Scanning: ScanningConfig{
		Vulnerability: VulnScannerSnyk,
		Behavioral:    BehavioralSocket,
		CIProtection:  CIProtectionHardenRunner,
	},
	Updates: UpdateConfig{
		Type:             UpdateToolRenovate,
		AgeGatingDays:    7,
		AutomergePatches: true,
	},
	SBOM: SBOMConfig{
		Generator: SBOMGeneratorSyft,
		Signing:   SBOMSigningCosign,
	},
}
