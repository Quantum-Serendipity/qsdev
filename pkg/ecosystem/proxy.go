package ecosystem

import "strings"

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

// ResolveProxyURL returns the registry proxy URL for a given ecosystem.
// Resolution order:
//  1. Per-ecosystem full-URL overrides
//  2. Base URL + custom path (from customPaths, typically InfraConfig.RegistryProxyPaths)
//  3. Base URL + built-in default path
//
// Returns "" if no proxy is configured.
func ResolveProxyURL(baseURL string, overrides map[string]string, ecosystem string, customPaths ...map[string]string) string {
	if baseURL == "" && len(overrides) == 0 {
		return ""
	}
	if url, ok := overrides[ecosystem]; ok && url != "" {
		return url
	}
	if baseURL == "" {
		return ""
	}
	base := strings.TrimRight(baseURL, "/")

	// Check custom paths first (from InfraConfig.RegistryProxyPaths).
	for _, paths := range customPaths {
		if path, ok := paths[ecosystem]; ok && path != "" {
			return base + path
		}
	}

	// Fall back to built-in defaults.
	defaults := DefaultProxyPaths()
	path, ok := defaults[ecosystem]
	if !ok {
		return ""
	}
	return base + path
}

// ProxyKeyForLanguage maps a language name and package manager to the proxy
// key used in DefaultProxyPaths.
func ProxyKeyForLanguage(langName, packageManager string) string {
	switch langName {
	case NameJavaScript:
		return "npm"
	case NamePython:
		return "pypi"
	case NameGo:
		return "go"
	case NameJava:
		if packageManager == "gradle" {
			return "gradle"
		}
		return "maven"
	case NameRust:
		return "cargo"
	case NameDotnet:
		return "nuget"
	case NamePHP:
		return "composer"
	default:
		return ""
	}
}
