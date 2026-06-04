package ecosystem

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// RegisterModule registers a module with the default registry.
// Returns an error if registration fails (e.g., duplicate name).
func RegisterModule(m EcosystemModule) error {
	return DefaultRegistry().Register(m)
}

// MustRegisterModule registers a module with the default registry and panics on failure.
// Intended for use in init() blocks where error handling is not possible.
func MustRegisterModule(m EcosystemModule) {
	if err := RegisterModule(m); err != nil {
		panic(fmt.Sprintf("%s: failed to register ecosystem module: %v", m.Name(), err))
	}
}

// DetectionAbsent returns a DetectionResult indicating no ecosystem was detected.
func DetectionAbsent() DetectionResult {
	return DetectionResult{
		Detected:   false,
		Confidence: ConfidenceAbsent,
	}
}

// SimpleNixFragment returns a basic Nix fragment that enables a single language.
func SimpleNixFragment(langName string) string {
	return fmt.Sprintf("  languages.%s.enable = true;\n", langName)
}

// ToModuleConfig converts a LanguageChoice from wizard answers into a
// ModuleConfig suitable for passing to EcosystemModule methods.
func ToModuleConfig(lang types.LanguageChoice) ModuleConfig {
	return ModuleConfig{
		Version:        lang.Version,
		PackageManager: lang.PackageManager,
		Extras:         ExtrasMap(lang.Extras),
	}
}

// ToModuleConfigWithProxy converts a LanguageChoice into a ModuleConfig with
// the registry proxy URL resolved for the specific ecosystem.
func ToModuleConfigWithProxy(lang types.LanguageChoice, infra types.InfraConfig) ModuleConfig {
	cfg := ToModuleConfig(lang)
	proxyKey := ProxyKeyForLanguage(lang.Name, lang.PackageManager)
	if proxyKey != "" {
		cfg.RegistryProxy = ResolveProxyURL(infra.RegistryProxy, infra.RegistryProxyOverrides, proxyKey)
	}
	return cfg
}

// ExtrasMap converts a []string of extras from LanguageChoice into a
// map[string]string for ModuleConfig.Extras. Each string is either:
//   - "key=value" → map[key] = value
//   - "key"       → map[key] = "true"
func ExtrasMap(extras []string) map[string]string {
	if len(extras) == 0 {
		return nil
	}
	m := make(map[string]string, len(extras))
	for _, e := range extras {
		if k, v, ok := strings.Cut(e, "="); ok {
			m[k] = v
		} else {
			m[e] = "true"
		}
	}
	return m
}
