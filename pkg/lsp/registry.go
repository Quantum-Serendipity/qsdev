package lsp

import (
	"fmt"
	"sort"
	"strings"
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
	if err := validateServers(servers); err != nil {
		return nil, err
	}
	r := &LSPRegistry{byEcosystem: make(map[string]*LSPServerConfig, len(servers))}
	for _, s := range servers {
		r.byEcosystem[s.EcosystemName] = s
	}
	return r, nil
}

// ByEcosystem returns the config for the given ecosystem name (e.g. "go") and
// whether it was found. The returned config is shared registry state and must
// be treated as read-only.
func (r *LSPRegistry) ByEcosystem(name string) (*LSPServerConfig, bool) {
	cfg, ok := r.byEcosystem[name]
	return cfg, ok
}

// All returns every registered config, sorted by EcosystemName for
// deterministic output. The configs are shared registry state and must be
// treated as read-only.
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

// Validate checks every registered server for the invariants generation
// relies on; see validateServers.
func (r *LSPRegistry) Validate() error {
	return validateServers(r.All())
}

// validateServers checks each server's own invariants (validateServer) and
// the cross-server ones:
//   - EcosystemName is unique (it is the registry key),
//   - LanguageID is unique (it keys the generated .lsp.json entries, so a
//     collision would silently drop a server),
//   - no extension is claimed by two default-on servers (extensionToLanguage
//     can route an extension to only one server).
//
// Errors are wrapped with the offending server's EcosystemName.
func validateServers(servers []*LSPServerConfig) error {
	ecosystems := make(map[string]struct{}, len(servers))
	languageIDs := make(map[string]string, len(servers))
	defaultOnExts := make(map[string]string)
	for _, cfg := range servers {
		if err := validateServer(cfg); err != nil {
			return err
		}
		if _, dup := ecosystems[cfg.EcosystemName]; dup {
			return fmt.Errorf("validating lsp server %q: duplicate EcosystemName", cfg.EcosystemName)
		}
		ecosystems[cfg.EcosystemName] = struct{}{}

		if other, dup := languageIDs[cfg.LanguageID]; dup {
			return fmt.Errorf("validating lsp server %q: LanguageID %q already used by %q", cfg.EcosystemName, cfg.LanguageID, other)
		}
		languageIDs[cfg.LanguageID] = cfg.EcosystemName

		if !cfg.DefaultOn {
			continue
		}
		for _, ext := range cfg.Extensions {
			if other, dup := defaultOnExts[ext]; dup {
				return fmt.Errorf("validating lsp server %q: extension %q already claimed by default-on server %q", cfg.EcosystemName, ext, other)
			}
			defaultOnExts[ext] = cfg.EcosystemName
		}
	}
	return nil
}

// validateServer checks a single server's invariants:
//   - non-empty EcosystemName, Command, and LanguageID,
//   - at least one Extension, each a leading dot followed by a suffix,
//   - a SandboxCategory in {A, B, C},
//   - a non-empty NixPackage unless the server is SDK-bundled,
//   - a DevenvLSPAttr exactly when the Devenv action emits lsp option lines
//     (otherwise NixLSPFragment would emit invalid Nix such as
//     "  .lsp.enable = true;").
func validateServer(cfg *LSPServerConfig) error {
	if cfg == nil {
		return fmt.Errorf("validating lsp registry: nil server config")
	}
	if cfg.EcosystemName == "" {
		return fmt.Errorf("validating lsp registry: server %q (command %q) has empty EcosystemName", cfg.DisplayName, cfg.Command)
	}
	if cfg.Command == "" {
		return fmt.Errorf("validating lsp server %q: empty Command", cfg.EcosystemName)
	}
	if cfg.LanguageID == "" {
		return fmt.Errorf("validating lsp server %q: empty LanguageID", cfg.EcosystemName)
	}
	if len(cfg.Extensions) == 0 {
		return fmt.Errorf("validating lsp server %q: no Extensions", cfg.EcosystemName)
	}
	for _, ext := range cfg.Extensions {
		if len(ext) < 2 || !strings.HasPrefix(ext, ".") {
			return fmt.Errorf("validating lsp server %q: extension %q must be a leading dot followed by a suffix", cfg.EcosystemName, ext)
		}
	}
	if _, ok := validSandboxCategories[cfg.SandboxCategory]; !ok {
		return fmt.Errorf("validating lsp server %q: invalid SandboxCategory %q (want A, B, or C)", cfg.EcosystemName, cfg.SandboxCategory)
	}
	if cfg.NixPackage == "" && cfg.Devenv != DevenvSDKBundled {
		return fmt.Errorf("validating lsp server %q: empty NixPackage (only allowed when Devenv == DevenvSDKBundled)", cfg.EcosystemName)
	}
	emitsLSPOption := cfg.Devenv == DevenvEnable || cfg.Devenv == DevenvEnableOverridePackage || cfg.Devenv == DevenvDisable
	if emitsLSPOption && cfg.DevenvLSPAttr == "" {
		return fmt.Errorf("validating lsp server %q: empty DevenvLSPAttr for a Devenv action that emits lsp option lines", cfg.EcosystemName)
	}
	if !emitsLSPOption && cfg.DevenvLSPAttr != "" {
		return fmt.Errorf("validating lsp server %q: DevenvLSPAttr %q is unused by its Devenv action", cfg.EcosystemName, cfg.DevenvLSPAttr)
	}
	return nil
}

// sortByEcosystem sorts configs in place by EcosystemName.
func sortByEcosystem(configs []*LSPServerConfig) {
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].EcosystemName < configs[j].EcosystemName
	})
}
