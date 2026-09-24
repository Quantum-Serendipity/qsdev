package profile

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// SecurityDocData holds data for rendering the security overview documentation template.
type SecurityDocData struct {
	ProfileName    string
	VulnScanner    string
	BehavioralTool string
	CIProtection   string
	UpdateTool     string
	AgeGatingDays  int
	SBOMGenerator  string
	// RegistryProxy, NixCache and BuildCache describe the infrastructure
	// actually applied (endpoints included), or state that it is not.
	RegistryProxy string
	NixCache      string
	BuildCache    string
	// Credentials are the environment variables the profile's services read.
	Credentials []string
}

// securityDocTmpl is parsed once from the embedded templates, so a broken
// template fails the tests (and startup) instead of being written over the
// committed security overview at generation time.
var securityDocTmpl = template.Must(
	template.New("security-overview.md.tmpl").Option("missingkey=error").
		ParseFS(templateFS, "templates/security-overview.md.tmpl"))

// generateSecurityDoc produces docs/security-overview.md. The
// infrastructure section describes in.Infrastructure, the settings the
// generators actually applied, so it never claims a proxy or cache that is
// not configured.
func (p *InfraProfile) generateSecurityDoc(in ProjectInputs) (types.GeneratedFile, error) {
	data := SecurityDocData{
		ProfileName:    p.Name,
		VulnScanner:    string(p.Scanning.Vulnerability),
		BehavioralTool: string(p.Scanning.Behavioral),
		CIProtection:   string(p.Scanning.CIProtection),
		UpdateTool:     string(p.Updates.Type),
		AgeGatingDays:  p.Updates.AgeGatingDays,
		SBOMGenerator:  string(p.SBOM.Generator),
		RegistryProxy:  registrySummary(in),
		NixCache:       nixCacheSummary(in.Infrastructure),
		BuildCache:     p.buildCacheSummary(in.Infrastructure),
		Credentials:    p.CredentialEnvVars(),
	}

	var buf bytes.Buffer
	if err := securityDocTmpl.Execute(&buf, data); err != nil {
		return types.GeneratedFile{}, fmt.Errorf("rendering security overview: %w", err)
	}

	return types.GeneratedFile{
		Path:     "docs/security-overview.md",
		Content:  buf.Bytes(),
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.Overwrite,
	}, nil
}

// registrySummary describes where the project's package installs are routed.
func registrySummary(in ProjectInputs) string {
	infra := in.Infrastructure
	var routed []string
	for _, eco := range in.Ecosystems {
		u := ecosystem.ResolveProxyURL(infra.RegistryProxyBase(), infra.RegistryProxyOverrides, eco, infra.RegistryProxyPaths)
		if u != "" {
			routed = append(routed, fmt.Sprintf("%s via %s", eco, u))
		}
	}
	if len(routed) == 0 {
		return "not configured; installs use the public registries (infrastructure.registry_proxy)"
	}
	return strings.Join(routed, "; ")
}

// nixCacheSummary describes the project's Nix binary cache.
func nixCacheSummary(infra types.InfraConfig) string {
	if u := infra.NixCacheURL(); u != "" {
		return u
	}
	return "not configured (infrastructure.nix_cache)"
}

// buildCacheSummary describes the project's shared build cache.
func (p *InfraProfile) buildCacheSummary(infra types.InfraConfig) string {
	if infra.BuildCache == "" || infra.BuildCache == string(BuildCacheNone) {
		return "none"
	}
	switch {
	case infra.BuildCacheURL != "" && infra.BuildCache == string(BuildCacheTurborepo):
		// Only Turborepo reads build_cache_url (as TURBO_API).
		return fmt.Sprintf("%s at %s", infra.BuildCache, infra.BuildCacheURL)
	case infra.BuildCache == string(p.BuildCache.Type) && p.BuildCache.Backend != "":
		return fmt.Sprintf("%s (%s backend)", infra.BuildCache, p.BuildCache.Backend)
	}
	return infra.BuildCache
}
