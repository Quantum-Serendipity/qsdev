package profile

import (
	"slices"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// EnvironmentVars returns the non-secret environment variables the resolved
// profile sets in the development shell (devenv.nix env). Registry routing
// is not among them: the ecosystem modules write it into each package
// manager's own configuration from Infrastructure(). Credentials are never
// returned; see CredentialEnvVars.
func (p *InfraProfile) EnvironmentVars() map[string]string {
	env := make(map[string]string)
	if p.BuildCache.Type == BuildCacheTurborepo && p.BuildCache.URL != "" {
		// Self-hosted Turborepo remote cache; without it turbo uses Vercel's.
		env["TURBO_API"] = p.BuildCache.URL
	}
	return env
}

// CredentialEnvVars returns the names of the credentials the profile's
// services read from the developer's (or CI's) environment, sorted. qsdev
// documents them and never writes their values: a devenv env entry would
// replace the real secret with a literal string.
func (p *InfraProfile) CredentialEnvVars() []string {
	var names []string
	if p.Registry.Type != RegistryNone && p.Registry.AuthEnvVar != "" {
		names = append(names, p.Registry.AuthEnvVar)
	}
	if p.NixCache.Type != NixCacheNone && p.NixCache.PushTokenEnvVar != "" {
		names = append(names, p.NixCache.PushTokenEnvVar)
	}
	if p.BuildCache.Type != BuildCacheNone {
		names = append(names, p.BuildCache.AuthEnvVars...)
	}
	if p.Scanning.Vulnerability == VulnScannerSnyk {
		names = append(names, "SNYK_TOKEN")
	}
	slices.Sort(names)
	return slices.Compact(names)
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
		f, err := p.generateSecurityScanWorkflow(in)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}

	// Security documentation
	doc, err := p.generateSecurityDoc(in)
	if err != nil {
		return nil, err
	}
	files = append(files, doc)

	return files, nil
}

// NixCacheNixConfig returns the substituter URL and trusted public key of
// the profile's Nix binary cache ("" when it has none or none is configured).
func (p *InfraProfile) NixCacheNixConfig() (substituter, trustedKey string) {
	switch p.NixCache.Type {
	case NixCacheCachix:
		if p.NixCache.URL != "" {
			substituter = p.NixCache.URL
		} else if p.NixCache.CacheName != "" {
			substituter = "https://" + p.NixCache.CacheName + ".cachix.org"
		}
	case NixCacheAttic, NixCacheNixServe:
		substituter = p.NixCache.URL
	default:
		return "", ""
	}
	if substituter == "" {
		return "", ""
	}
	return substituter, p.NixCache.PublicKey
}
