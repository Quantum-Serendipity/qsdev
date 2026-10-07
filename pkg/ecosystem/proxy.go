package ecosystem

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// DefaultProxyPaths returns the built-in Nexus-style repository path suffixes
// keyed by ecosystem. These are appended to InfraConfig.RegistryProxy when no
// per-ecosystem override or custom path is provided.
func DefaultProxyPaths() map[string]string {
	return map[string]string{
		"npm":      "/repository/npm-proxy/",
		"pypi":     "/repository/pypi-proxy/simple/",
		"go":       "/repository/go-proxy/",
		"maven":    "/repository/maven-central/",
		"gradle":   "/repository/maven-central/",
		"cargo":    "/repository/cargo-proxy/",
		"nuget":    "/repository/nuget-proxy/v3/index.json",
		"composer": "/repository/composer-proxy/",
	}
}

// JoinProxyURL appends an ecosystem repository path to a registry proxy base
// URL. path must be an absolute path on the proxy host: it has to start with
// "/" and must not start with "//". The join goes through url.URL.JoinPath, so
// the result always keeps the base's scheme, userinfo and host (a value such
// as "@attacker.io/npm/" can never move the URL to another host the way string
// concatenation would). A base that url.Parse rejects, for example one holding
// a newline or other control character, returns an error.
func JoinProxyURL(base, path string) (string, error) {
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return "", fmt.Errorf("proxy path %q must be an absolute path on the proxy host (start with a single \"/\")", path)
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parsing proxy base URL: %w", err)
	}
	return u.JoinPath(path).String(), nil
}

// ResolveProxyURL returns the registry proxy URL for a given ecosystem.
// Resolution order:
//  1. Per-ecosystem full-URL overrides (returned verbatim)
//  2. Base URL + custom path (from customPaths, typically InfraConfig.RegistryProxyPaths)
//  3. Base URL + built-in default path
//
// Cases 2 and 3 are joined with JoinProxyURL; if the join fails (invalid base
// or a path that is not absolute on the proxy host) the result is "" rather
// than a mangled URL. Callers must validate the infrastructure configuration
// with profile.ValidateInfra first so such values surface as errors; the
// devenv generator always does.
//
// Returns "" if no proxy is configured.
func ResolveProxyURL(baseURL string, overrides map[string]string, ecosystem string, customPaths ...map[string]string) string {
	if baseURL == "" && len(overrides) == 0 {
		return ""
	}
	if override, ok := overrides[ecosystem]; ok && override != "" {
		return override
	}
	if baseURL == "" {
		return ""
	}

	// Check custom paths first (from InfraConfig.RegistryProxyPaths).
	for _, paths := range customPaths {
		if path, ok := paths[ecosystem]; ok && path != "" {
			return joinOrEmpty(baseURL, path)
		}
	}

	// Fall back to built-in defaults.
	path, ok := DefaultProxyPaths()[ecosystem]
	if !ok {
		return ""
	}
	return joinOrEmpty(baseURL, path)
}

// joinOrEmpty is JoinProxyURL with errors collapsed to "" for callers that
// have already validated their inputs.
func joinOrEmpty(base, path string) string {
	joined, err := JoinProxyURL(base, path)
	if err != nil {
		return ""
	}
	return joined
}

// ProxyKeyForLanguage returns the registry proxy key (see ProxyKeyProvider)
// that the DefaultRegistry module for lang routes through with lang's
// configuration, or "" when the language has no module or is not routed.
func ProxyKeyForLanguage(lang types.LanguageChoice) string {
	return DefaultRegistry().proxyKeyFor(lang.Name, ToModuleConfig(lang))
}

// proxyKeyFor returns the registry proxy key of the module registered as name
// for config, or "" when there is no such module or it is not routed.
func (r *Registry) proxyKeyFor(name string, config ModuleConfig) string {
	m, ok := r.ByName(name)
	if !ok {
		return ""
	}
	return moduleProxyKey(m, config)
}

// moduleProxyKey returns m's registry proxy key for config, or "" when m does
// not implement ProxyKeyProvider.
func moduleProxyKey(m EcosystemModule, config ModuleConfig) string {
	p, ok := m.(ProxyKeyProvider)
	if !ok {
		return ""
	}
	return p.ProxyKey(config)
}
