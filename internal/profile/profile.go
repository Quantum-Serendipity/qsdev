package profile

import (
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// EnvironmentVars returns the environment variables implied by this profile.
// Secret values are never hardcoded; instead, shell-style ${VAR} references
// are used so that the actual secrets are resolved at runtime.
func (p *InfraProfile) EnvironmentVars() map[string]string {
	env := make(map[string]string)

	// --- Registry environment variables ---
	if p.Registry.Type != RegistryNone {
		for _, eco := range p.Registry.Ecosystems {
			u := p.Registry.EcosystemURL(eco)
			if u == "" {
				continue
			}
			switch eco {
			case "npm":
				env["NPM_CONFIG_REGISTRY"] = u
			case "pypi":
				env["PIP_INDEX_URL"] = u
			case "go":
				// No ",direct": the go command falls back to the next
				// entry on any 404/410, bypassing the proxy's policy.
				env["GOPROXY"] = u
			case "cargo":
				env["CARGO_REGISTRIES_INTERNAL_INDEX"] = u
			case "maven":
				env["MAVEN_REPO_URL"] = u
			case "nuget":
				env["NUGET_SOURCE_URL"] = u
			}
		}
	}

	// --- Build cache ---
	switch p.BuildCache.Type {
	case BuildCacheSccache:
		env["RUSTC_WRAPPER"] = "sccache"
		if p.BuildCache.Backend == "s3" {
			env["SCCACHE_BUCKET"] = "${SCCACHE_BUCKET}"
		}
		for k, v := range p.BuildCache.AuthEnvVars {
			env[k] = v
		}
	case BuildCacheTurborepo:
		if p.BuildCache.URL != "" {
			env["TURBO_API"] = p.BuildCache.URL
		}
	}

	// --- Nix cache ---
	if p.NixCache.Type == NixCacheCachix && p.NixCache.SigningKeyRef != "" {
		env["CACHIX_SIGNING_KEY"] = "${CACHIX_SIGNING_KEY}"
	}

	// --- Scanning ---
	if p.Scanning.Vulnerability == VulnScannerSnyk {
		env["SNYK_TOKEN"] = "${SNYK_TOKEN}"
	}
	// Socket.dev uses the public HTTP endpoint — no API key required.

	return env
}

// ConfigFiles returns generated configuration files implied by the profile's
// update-tool selection (e.g., renovate.json or .github/dependabot.yml),
// CI vulnerability scanning workflow, and security documentation. in carries
// the project facts (such as its package ecosystems) the files depend on.
func (p *InfraProfile) ConfigFiles(in ProjectInputs) ([]types.GeneratedFile, error) {
	var files []types.GeneratedFile

	switch p.Updates.Type {
	case UpdateToolRenovate:
		files = append(files, p.generateRenovateJSON())
	case UpdateToolDependabot:
		f, err := p.generateDependabotYML(in)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}

	// CI vulnerability scanning workflow
	if p.generatesSecurityScanWorkflow() {
		f, err := p.generateSecurityScanWorkflow()
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}

	// Security documentation
	doc, err := p.generateSecurityDoc()
	if err != nil {
		return nil, err
	}
	files = append(files, doc)

	return files, nil
}

// NixCacheNixConfig returns nix.conf snippet values for substituters and
// trusted-public-keys based on the configured cache.
func (p *InfraProfile) NixCacheNixConfig() (substituters, trustedKeys string) {
	switch p.NixCache.Type {
	case NixCacheCachix:
		if p.NixCache.URL != "" {
			substituters = p.NixCache.URL
		} else if p.NixCache.CacheName != "" {
			substituters = "https://" + p.NixCache.CacheName + ".cachix.org"
		}
		trustedKeys = p.NixCache.PublicKey
	case NixCacheAttic:
		substituters = p.NixCache.URL
		trustedKeys = p.NixCache.PublicKey
	case NixCacheNixServe:
		substituters = p.NixCache.URL
		trustedKeys = p.NixCache.PublicKey
	}
	return
}
