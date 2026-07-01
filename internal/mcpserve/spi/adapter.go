package spi

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/Quantum-Serendipity/qsdev/internal/registry"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

// errNilHandler is returned when a Chain is executed without a final handler.
var errNilHandler = errors.New("spi: nil final tool handler")

// FrameworkAdapter is the contract concrete framework adapters implement to
// contribute framework-specific tools, resources, and prompts to the universal
// server. Adapters live under internal/mcpserve/adapters/* and delegate to
// addons; they self-register into the DefaultRegistry from cmd/qsdev/main.go.
//
// The interface is intentionally minimal: identity, an applicability check, and
// three contribution methods. Later tasks implement concrete adapters against
// it.
type FrameworkAdapter interface {
	// ID returns the framework this adapter serves.
	ID() aiframework.FrameworkID
	// Applies reports whether this adapter should contribute for the given
	// resolved project root. Adapters that are always applicable return true.
	Applies(ctx context.Context, projectRoot string) bool
	// Tools returns the tool registrations this adapter contributes.
	Tools() []ToolRegistration
	// Resources returns the resource registrations this adapter contributes.
	Resources() []ResourceRegistration
	// Prompts returns the prompt registrations this adapter contributes.
	Prompts() []PromptRegistration
}

// ClientMatcher is an OPTIONAL interface a FrameworkAdapter may implement to
// customize how it is matched against an MCP client's self-reported identity
// (the initialize-handshake clientInfo). Adapters that do not implement it fall
// back to DefaultClientMatch, which compares the client name against a token
// derived from the adapter's FrameworkID.
//
// It is deliberately separate from FrameworkAdapter so that adding client-aware
// matching never changes the base contract every existing adapter already
// satisfies. DetectFrameworks honors it when present.
type ClientMatcher interface {
	// MatchesClient reports whether this adapter should serve the given MCP
	// client, as identified by the initialize-handshake clientInfo.
	MatchesClient(client ClientInfo) bool
}

// normalizeIdentToken lowercases s and strips every character that is not an
// ASCII letter or digit, so "Claude Code" and "claude-code" both collapse to
// "claudecode". It is the shared normalization the framework-id token and the
// client name pass through before the containment test in DefaultClientMatch.
func normalizeIdentToken(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		}
	}
	return b.String()
}

// DefaultClientMatch is the fallback client-matching rule used for any adapter
// that does not implement ClientMatcher. It reports whether the client's
// reported name, once normalized (lowercased, non-alphanumerics stripped),
// contains the framework id token normalized the same way. Both the token and
// the name must be non-empty for a match.
//
// Examples: id "claudecode" matches client names "claude-code" and "Claude
// Code" (both normalize to "claudecode"); id "gemini" matches "Gemini CLI".
func DefaultClientMatch(id aiframework.FrameworkID, client ClientInfo) bool {
	token := normalizeIdentToken(string(id))
	name := normalizeIdentToken(client.Name)
	if token == "" || name == "" {
		return false
	}
	return strings.Contains(name, token)
}

// matchesClient applies an adapter's ClientMatcher when it implements one, else
// the DefaultClientMatch rule keyed on the adapter's FrameworkID.
func matchesClient(a FrameworkAdapter, client ClientInfo) bool {
	if m, ok := a.(ClientMatcher); ok {
		return m.MatchesClient(client)
	}
	return DefaultClientMatch(a.ID(), client)
}

// AdapterRegistry is a thread-safe collection of FrameworkAdapter
// implementations, keyed by FrameworkID. It mirrors the pattern used by
// pkg/ecosystem and internal/mcpserver: embed the generic registry and add a
// domain-typed Register/All.
type AdapterRegistry struct {
	*registry.Registry[FrameworkAdapter]
}

// NewAdapterRegistry creates an empty AdapterRegistry.
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		Registry: registry.New[FrameworkAdapter](
			registry.WithEntityName("framework adapter"),
		),
	}
}

// Register adds an adapter keyed by its ID(). It returns an error if an adapter
// with the same ID is already registered.
func (r *AdapterRegistry) Register(a FrameworkAdapter) error {
	return r.Registry.Register(string(a.ID()), a)
}

// All returns every registered adapter, sorted by FrameworkID for determinism.
func (r *AdapterRegistry) All() []FrameworkAdapter {
	adapters := r.Values()
	sort.Slice(adapters, func(i, j int) bool {
		return adapters[i].ID() < adapters[j].ID()
	})
	return adapters
}

// DetectFrameworks returns every registered adapter that matches the given MCP
// client identity, in the deterministic order of All(). An adapter matches when
// it implements ClientMatcher and MatchesClient returns true, or—when it does
// not—when DefaultClientMatch(adapter.ID(), client) returns true. The result is
// empty when no adapter matches (e.g. an unknown client), which the universal
// server's tool filter treats as generic-only fallback mode.
func (r *AdapterRegistry) DetectFrameworks(client ClientInfo) []FrameworkAdapter {
	all := r.All()
	matched := make([]FrameworkAdapter, 0, len(all))
	for _, a := range all {
		if matchesClient(a, client) {
			matched = append(matched, a)
		}
	}
	return matched
}

var (
	defaultRegistryOnce sync.Once
	defaultRegistry     *AdapterRegistry
)

// DefaultRegistry returns the package-level singleton AdapterRegistry, lazily
// initialized on first use. Concrete adapters register into this from
// cmd/qsdev/main.go; the server consumes it. Explicit registration is preferred
// over init() so wiring order stays visible at the program entry point.
func DefaultRegistry() *AdapterRegistry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewAdapterRegistry()
	})
	return defaultRegistry
}
