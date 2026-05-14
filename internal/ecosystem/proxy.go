package ecosystem

import "strings"

// DefaultProxyPaths maps ecosystem proxy keys to Nexus-style repository path
// suffixes. These are appended to InfraConfig.RegistryProxy when no
// per-ecosystem override is provided.
var DefaultProxyPaths = map[string]string{
	"npm":      "/repository/npm-proxy/",
	"pypi":     "/repository/pypi-proxy/simple/",
	"go":       "/repository/go-proxy/",
	"maven":    "/repository/maven-central/",
	"gradle":   "/repository/maven-central/",
	"cargo":    "/repository/cargo-proxy/",
	"nuget":    "/repository/nuget-proxy/v3/index.json",
	"composer": "/repository/composer-proxy/",
}

// ResolveProxyURL returns the registry proxy URL for a given ecosystem.
// Per-ecosystem overrides take precedence over the base URL + default path.
// Returns "" if no proxy is configured.
func ResolveProxyURL(baseURL string, overrides map[string]string, ecosystem string) string {
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
	path, ok := DefaultProxyPaths[ecosystem]
	if !ok {
		return ""
	}
	return base + path
}

// ProxyKeyForLanguage maps a language name and package manager to the proxy
// key used in DefaultProxyPaths.
func ProxyKeyForLanguage(langName, packageManager string) string {
	switch langName {
	case "javascript":
		return "npm"
	case "python":
		return "pypi"
	case "go":
		return "go"
	case "java":
		if packageManager == "gradle" {
			return "gradle"
		}
		return "maven"
	case "rust":
		return "cargo"
	case "dotnet":
		return "nuget"
	case "php":
		return "composer"
	default:
		return ""
	}
}
