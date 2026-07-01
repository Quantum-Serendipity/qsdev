package mcpserve_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// spyAuditSink is a concurrency-safe middleware.AuditSink that captures every
// audit record so the test can assert the chain ran through the bridge (and that
// a policy denial was recorded).
type spyAuditSink struct {
	mu      sync.Mutex
	records []middleware.AuditRecord
}

func (s *spyAuditSink) Record(_ context.Context, rec middleware.AuditRecord) {
	s.mu.Lock()
	s.records = append(s.records, rec)
	s.mu.Unlock()
}

// decisionFor returns the recorded decision for the named tool/URI, if any.
func (s *spyAuditSink) decisionFor(tool string) (middleware.Decision, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.records {
		if r.Tool == tool {
			return r.Decision, true
		}
	}
	return "", false
}

// observed returns a "tool:decision" summary of every captured record for
// diagnostics on assertion failure.
func (s *spyAuditSink) observed() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.records))
	for _, r := range s.records {
		out = append(out, r.Tool+":"+string(r.Decision))
	}
	return out
}

// plantedContributor satisfies mcpserve.ProjectContributor with deliberately
// hostile fixtures: tools and a resource that emit a planted secret, a tool the
// policy denies by category, and a per-package URI TEMPLATE that echoes the
// concrete read URI so template resolution (R11) is observable.
type plantedContributor struct {
	secret         string
	deniedCategory string
}

func (p plantedContributor) Tools() []spi.ToolRegistration {
	return []spi.ToolRegistration{
		{
			Name:        "qsdev_test_secret_tool",
			Description: "returns a planted secret in both text and structured content",
			Handler: func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
				return &spi.ToolResult{
					Text: "leaked credential in text: " + p.secret,
					Structured: map[string]any{
						// "token" exercises the key-deny redaction path; the
						// nested value exercises the value-pattern path.
						"token":  p.secret,
						"nested": map[string]any{"deep": p.secret},
					},
				}, nil
			},
		},
		{
			Name:        "qsdev_test_denied_tool",
			Description: "a tool the test policy denies by category",
			Category:    p.deniedCategory,
			Handler: func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
				// Must never run: Guardrail short-circuits before the handler.
				return &spi.ToolResult{Text: "handler ran despite policy denial"}, nil
			},
		},
	}
}

func (p plantedContributor) Resources() []spi.ResourceRegistration {
	return []spi.ResourceRegistration{
		{
			URI:         "qsdev://test/secret",
			Name:        "Planted secret resource",
			Description: "returns a planted secret in its content text",
			MIMEType:    "text/plain",
			Handler: func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ResourceRequest) (*spi.ResourceResult, error) {
				return &spi.ResourceResult{Contents: []spi.ResourceContent{{
					MIMEType: "text/plain",
					Text:     "settings.json leaked: " + p.secret,
				}}}, nil
			},
		},
		{
			URI:         "qsdev://test/{pkg}/context",
			Name:        "Planted per-package template",
			Description: "resolves a concrete package URI through a resource template",
			MIMEType:    "application/json",
			Handler: func(_ context.Context, _ *spi.ToolCallContext, req *spi.ResourceRequest) (*spi.ResourceResult, error) {
				// Echo the CONCRETE URI so the test can prove the template was
				// matched and the concrete read URI forwarded (R11).
				return &spi.ResourceResult{Contents: []spi.ResourceContent{{
					URI:      req.URI,
					MIMEType: "application/json",
					Text:     `{"resolved":true,"uri":"` + req.URI + `"}`,
				}}}, nil
			},
		},
	}
}

func (p plantedContributor) Prompts() []spi.PromptRegistration { return nil }

// TestChainEnforcementOverProtocol is the Phase 32 security linchpin: it drives
// the REAL built-in middleware chain end-to-end through the bridge over the stdio
// transport and proves the chain enforces on every surface.
//
//   - R7: a resource read is redacted, proving resource reads are routed THROUGH
//     the chain (previously they bypassed it).
//   - R10: the chain runs at all for a server constructed via WithChain wrapping
//     middleware.DefaultChain — and, by extension, that DefaultChain is the
//     non-fail-open default.
//   - R11: a concrete read of a templated URI resolves, proving the templated
//     resource is registered as a resource template rather than a static URI.
//
// It injects a spy AuditSink and a category-deny policy so it can observe audit
// records and force a Guardrail denial. The planted secret is assembled by
// concatenation so ripsecrets does not flag a literal credential in the source.
//
// NOTE: not parallel — the stdio transport uses a process-global session
// singleton (see integration_test.go), so it must not race other stdio tests.
func TestChainEnforcementOverProtocol(t *testing.T) {
	const secret = "AKIA" + "IOSFODNN7EXAMPLE" // AWS access key shape; not a real key
	const deniedCategory = "test_blocked"

	spy := &spyAuditSink{}
	denyPolicy := &middleware.Policy{
		Default:    middleware.VerdictAllow,
		ByToolType: map[string]middleware.Verdict{deniedCategory: middleware.VerdictDeny},
	}
	chain := middleware.DefaultChain(
		middleware.WithAuditSink(spy),
		middleware.WithPolicy(denyPolicy),
	)

	srv := mcpserve.New(
		mcpserve.WithProjectRoot(t.TempDir()),
		mcpserve.WithChain(chain),
	)
	srv.MountProjectContext(plantedContributor{secret: secret, deniedCategory: deniedCategory})

	c := newTestClient(t, srv)
	c.initialize("chain-enforcement-client")

	// (a) tools/call: the planted secret is redacted in BOTH text and structured.
	t.Run("tool result redacted in text and structured", func(t *testing.T) {
		res := c.callTool("qsdev_test_secret_tool", nil)
		if res.isError {
			t.Fatalf("secret tool returned an error result: %q", res.text)
		}
		if strings.Contains(res.text, secret) {
			t.Errorf("tools/call text leaked the planted secret: %q", res.text)
		}
		if !strings.Contains(res.text, "[REDACTED]") {
			t.Errorf("tools/call text was not redacted: %q", res.text)
		}
		sc := string(res.structured)
		if strings.Contains(sc, secret) {
			t.Errorf("tools/call structuredContent leaked the planted secret: %s", sc)
		}
		if !strings.Contains(sc, "[REDACTED]") {
			t.Errorf("tools/call structuredContent was not redacted: %s", sc)
		}
	})

	// (b) resources/read: the planted secret is redacted — proves R7 routing.
	t.Run("resource read redacted (R7)", func(t *testing.T) {
		_, body := c.readResource("qsdev://test/secret")
		if strings.Contains(body, secret) {
			t.Errorf("resources/read leaked the planted secret (chain not wired): %q", body)
		}
		if !strings.Contains(body, "[REDACTED]") {
			t.Errorf("resources/read content was not redacted: %q", body)
		}
	})

	// (c) resources/read of a CONCRETE templated URI resolves — proves R11.
	t.Run("templated resource resolves concrete URI (R11)", func(t *testing.T) {
		mime, body := c.readResource("qsdev://test/web/context")
		if mime != "application/json" {
			t.Errorf("template mime = %q, want application/json", mime)
		}
		if !strings.Contains(body, "qsdev://test/web/context") {
			t.Errorf("template did not resolve the concrete read URI; body=%q", body)
		}
	})

	// (d) the denied tool short-circuits AND is audited — proves the chain runs
	// through the bridge and Audit wraps the Guardrail decision.
	t.Run("denied tool short-circuited and audited", func(t *testing.T) {
		res := c.callTool("qsdev_test_denied_tool", nil)
		if !res.isError {
			t.Errorf("denied tool did not return an error result: %q", res.text)
		}
		if strings.Contains(res.text, "handler ran") {
			t.Errorf("denied tool handler executed despite policy denial: %q", res.text)
		}
		dec, ok := spy.decisionFor("qsdev_test_denied_tool")
		if !ok {
			t.Fatalf("no audit record for the denied tool; observed=%v", spy.observed())
		}
		if dec != middleware.DecisionDenied {
			t.Errorf("denied tool audit decision = %q, want %q; observed=%v",
				dec, middleware.DecisionDenied, spy.observed())
		}
	})

	c.close()
	if err, ok := c.serveError(); ok && !isCleanShutdown(err) {
		t.Errorf("Listen returned unexpected error on shutdown: %v", err)
	}
}
