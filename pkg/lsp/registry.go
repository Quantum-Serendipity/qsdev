package lsp

import (
	"fmt"
	"sort"
)

// validSandboxCategories is the closed set of accepted SandboxCategory values.
var validSandboxCategories = map[string]struct{}{
	"A": {},
	"B": {},
	"C": {},
}

// LSPRegistry is the single source of truth for LSP server configurations,
// keyed by ecosystem name (EcosystemModule.Name()).
type LSPRegistry struct {
	byEcosystem map[string]*LSPServerConfig
}

// NewRegistry builds the registry from the canonical server definitions and
// validates it. It panics if validation fails, because the definitions are
// compile-time constants in this package: a failure is a programming error, not
// a runtime condition. Use NewRegistryChecked when a recoverable error is
// preferred.
func NewRegistry() *LSPRegistry {
	r, err := NewRegistryChecked()
	if err != nil {
		panic(fmt.Sprintf("lsp: building registry: %v", err))
	}
	return r
}

// NewRegistryChecked builds the registry from the canonical server definitions
// and returns an error instead of panicking if validation fails.
func NewRegistryChecked() (*LSPRegistry, error) {
	servers := defaultServers()
	r := &LSPRegistry{byEcosystem: make(map[string]*LSPServerConfig, len(servers))}
	for _, s := range servers {
		if s.EcosystemName == "" {
			return nil, fmt.Errorf("validating lsp registry: server %q (command %q) has empty EcosystemName", s.DisplayName, s.Command)
		}
		if _, dup := r.byEcosystem[s.EcosystemName]; dup {
			return nil, fmt.Errorf("validating lsp registry: duplicate EcosystemName %q", s.EcosystemName)
		}
		r.byEcosystem[s.EcosystemName] = s
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return r, nil
}

// ByEcosystem returns the config for the given ecosystem name (e.g. "go") and
// whether it was found.
func (r *LSPRegistry) ByEcosystem(name string) (*LSPServerConfig, bool) {
	cfg, ok := r.byEcosystem[name]
	return cfg, ok
}

// All returns every registered config, sorted by EcosystemName for
// deterministic output.
func (r *LSPRegistry) All() []*LSPServerConfig {
	out := make([]*LSPServerConfig, 0, len(r.byEcosystem))
	for _, cfg := range r.byEcosystem {
		out = append(out, cfg)
	}
	sortByEcosystem(out)
	return out
}

// DefaultOn returns every config with DefaultOn == true, sorted by
// EcosystemName for deterministic output.
func (r *LSPRegistry) DefaultOn() []*LSPServerConfig {
	var out []*LSPServerConfig
	for _, cfg := range r.byEcosystem {
		if cfg.DefaultOn {
			out = append(out, cfg)
		}
	}
	sortByEcosystem(out)
	return out
}

// Validate checks every registered server for required invariants:
//   - non-empty, unique EcosystemName (also enforced during construction),
//   - non-empty LanguageID,
//   - at least one Extension,
//   - a SandboxCategory in {A, B, C},
//   - a non-empty NixPackage unless the server is SDK-bundled.
//
// Errors are wrapped with the offending server's EcosystemName.
func (r *LSPRegistry) Validate() error {
	seen := make(map[string]struct{}, len(r.byEcosystem))
	for name, cfg := range r.byEcosystem {
		if cfg.EcosystemName == "" {
			return fmt.Errorf("validating lsp registry: server keyed %q has empty EcosystemName", name)
		}
		if cfg.EcosystemName != name {
			return fmt.Errorf("validating lsp server %q: EcosystemName %q does not match its registry key", name, cfg.EcosystemName)
		}
		if _, dup := seen[cfg.EcosystemName]; dup {
			return fmt.Errorf("validating lsp server %q: duplicate EcosystemName", cfg.EcosystemName)
		}
		seen[cfg.EcosystemName] = struct{}{}

		if cfg.LanguageID == "" {
			return fmt.Errorf("validating lsp server %q: empty LanguageID", cfg.EcosystemName)
		}
		if len(cfg.Extensions) == 0 {
			return fmt.Errorf("validating lsp server %q: no Extensions", cfg.EcosystemName)
		}
		if _, ok := validSandboxCategories[cfg.SandboxCategory]; !ok {
			return fmt.Errorf("validating lsp server %q: invalid SandboxCategory %q (want A, B, or C)", cfg.EcosystemName, cfg.SandboxCategory)
		}
		if cfg.NixPackage == "" && cfg.Devenv != DevenvSDKBundled {
			return fmt.Errorf("validating lsp server %q: empty NixPackage (only allowed when Devenv == DevenvSDKBundled)", cfg.EcosystemName)
		}
	}
	return nil
}

// sortByEcosystem sorts configs in place by EcosystemName.
func sortByEcosystem(configs []*LSPServerConfig) {
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].EcosystemName < configs[j].EcosystemName
	})
}
