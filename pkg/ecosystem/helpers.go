package ecosystem

import (
	"fmt"
	"slices"
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

// ExtraBuildCache is the ModuleConfig extra naming the shared build cache
// (for example "sccache"); ToModuleConfigWithInfra fills it from
// infrastructure.build_cache.
const ExtraBuildCache = "build_cache"

// ToModuleConfig converts a LanguageChoice from wizard answers into a
// ModuleConfig suitable for passing to EcosystemModule methods.
func ToModuleConfig(lang types.LanguageChoice) ModuleConfig {
	return ModuleConfig{
		Version:        lang.Version,
		PackageManager: lang.PackageManager,
		Extras:         ExtrasMap(lang.Extras),
	}
}

// ToModuleConfigWithInfra converts a LanguageChoice into a ModuleConfig with
// the project's infrastructure applied: the registry proxy URL resolved for
// the specific ecosystem, and infrastructure.build_cache as the "build_cache"
// extra unless the language sets that extra itself.
func ToModuleConfigWithInfra(lang types.LanguageChoice, infra types.InfraConfig) ModuleConfig {
	cfg := ToModuleConfig(lang)
	if infra.BuildCache != "" {
		if cfg.Extras == nil {
			cfg.Extras = make(map[string]string, 1)
		}
		if _, ok := cfg.Extras[ExtraBuildCache]; !ok {
			cfg.Extras[ExtraBuildCache] = infra.BuildCache
		}
	}
	// Some ecosystems (e.g. Java) record their build tool in
	// Extras["build_tool"] when it was detected rather than set explicitly;
	// an explicit PackageManager still wins.
	proxyKey := ProxyKeyForLanguage(lang.Name, cfg.PM(cfg.Extra("build_tool", "")))
	if proxyKey != "" {
		cfg.RegistryProxy = ResolveProxyURL(infra.RegistryProxyBase(), infra.RegistryProxyOverrides, proxyKey, infra.RegistryProxyPaths)
	}
	return cfg
}

// ToGenerationConfig converts a LanguageChoice into the ModuleConfig
// generation passes to a module: ToModuleConfigWithInfra plus the
// project-level module settings the answers carry from .qsdev.yaml
// (java.repository_allowlist).
func ToGenerationConfig(lang types.LanguageChoice, answers types.WizardAnswers) ModuleConfig {
	cfg := ToModuleConfigWithInfra(lang, answers.Infrastructure)
	cfg.RepositoryAllowlist = slices.Clone(answers.Java.RepositoryAllowlist)
	return cfg
}

// PipeToShellDenyRules returns deny-rule patterns that block pipe-to-shell
// execution. These patterns are common supply chain attack vectors and are
// shared across multiple ecosystem modules.
func PipeToShellDenyRules() []string {
	return []string{
		"Bash(curl * | sh*)",
		"Bash(curl * | bash*)",
		"Bash(wget * | sh*)",
		"Bash(wget * | bash*)",
	}
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
