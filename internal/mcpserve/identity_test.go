package mcpserve

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

func TestClientInfoToSPI(t *testing.T) {
	t.Parallel()

	in := mcp.Implementation{Name: "claude-code", Version: "1.2.3", Title: "Claude Code"}
	got := clientInfoToSPI(in)
	want := spi.ClientInfo{Name: "claude-code", Version: "1.2.3", Title: "Claude Code"}
	if got != want {
		t.Errorf("clientInfoToSPI = %+v, want %+v", got, want)
	}
}

func TestAgentIDFromMeta(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		meta   map[string]any
		wantID string
		wantOK bool
	}{
		{
			name:   "present and non-empty",
			meta:   map[string]any{MetaAgentIDKey: "agent-42"},
			wantID: "agent-42",
			wantOK: true,
		},
		{name: "nil meta", meta: nil, wantOK: false},
		{name: "key absent", meta: map[string]any{"other": "x"}, wantOK: false},
		{name: "empty string value", meta: map[string]any{MetaAgentIDKey: ""}, wantOK: false},
		{name: "non-string value", meta: map[string]any{MetaAgentIDKey: 7}, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotID, gotOK := agentIDFromMeta(tt.meta)
			if gotOK != tt.wantOK || (tt.wantOK && gotID != tt.wantID) {
				t.Errorf("agentIDFromMeta = (%q,%v), want (%q,%v)", gotID, gotOK, tt.wantID, tt.wantOK)
			}
		})
	}
}

func TestResolveAgentID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		client spi.ClientInfo
		meta   map[string]any
		want   string
	}{
		{
			name:   "meta override wins over clientInfo",
			client: spi.ClientInfo{Name: "claude-code"},
			meta:   map[string]any{MetaAgentIDKey: "override-id"},
			want:   "override-id",
		},
		{
			name:   "clientInfo name used when no meta",
			client: spi.ClientInfo{Name: "claude-code"},
			meta:   nil,
			want:   "claude-code",
		},
		{
			name:   "empty meta string falls back to clientInfo",
			client: spi.ClientInfo{Name: "claude-code"},
			meta:   map[string]any{MetaAgentIDKey: ""},
			want:   "claude-code",
		},
		{
			name:   "unknown when neither available",
			client: spi.ClientInfo{},
			meta:   nil,
			want:   unknownAgentID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := resolveAgentID(tt.client, tt.meta); got != tt.want {
				t.Errorf("resolveAgentID = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestCallContextPrincipalIgnoresMetaOverride is the regression test for rate
// limiting keyed on the self-asserted _meta agent id: the Principal carried on
// every call context must be stable across rotated _meta overrides within one
// session, differ between sessions, and prefer a verified cert identity.
func TestCallContextPrincipalIgnoresMetaOverride(t *testing.T) {
	t.Parallel()
	srv := New(WithProjectRoot(t.TempDir()))
	ctx := ctxWithClient(srv, "claude-code")

	a := srv.callContext(ctx, "tool", map[string]any{MetaAgentIDKey: "id-a"})
	b := srv.callContext(ctx, "tool", map[string]any{MetaAgentIDKey: "id-b"})
	if a.AgentID == b.AgentID {
		t.Fatalf("precondition: _meta override should change AgentID, got %q for both", a.AgentID)
	}
	if a.Principal == "" || a.Principal != b.Principal {
		t.Errorf("Principal = %q / %q, want one stable non-empty value across rotated _meta ids", a.Principal, b.Principal)
	}
	if res := srv.callContext(ctx, "qsdev://r", nil); res.Principal != a.Principal {
		t.Errorf("resource read Principal = %q, want the tool call's %q", res.Principal, a.Principal)
	}

	certCtx := withTrustedAgent(ctx, "ci-bot")
	if got := srv.callContext(certCtx, "tool", map[string]any{MetaAgentIDKey: "spoof"}).Principal; got != "cert:ci-bot" {
		t.Errorf("verified Principal = %q, want cert:ci-bot", got)
	}

	// Streamable HTTP accepts any well-formed Mcp-Session-Id, so a caller can
	// rotate never-initialized session ids per request. Those ephemeral sessions
	// carry no handshake name and must share one principal, not mint a new one.
	rotated := func(id string) string {
		sess := &fakeClientSession{id: id}
		return srv.callContext(srv.mcp.WithContext(context.Background(), sess), "tool", nil).Principal
	}
	if p1, p2 := rotated("mcp-session-1"), rotated("mcp-session-2"); p1 != p2 {
		t.Errorf("uninitialized sessions got distinct principals %q / %q, want one shared principal", p1, p2)
	}
}

func TestMetaFromMCP(t *testing.T) {
	t.Parallel()

	if got := metaFromMCP(nil); got != nil {
		t.Errorf("metaFromMCP(nil) = %v, want nil", got)
	}
	if got := metaFromMCP(&mcp.Meta{}); got != nil {
		t.Errorf("metaFromMCP(empty) = %v, want nil", got)
	}

	src := &mcp.Meta{AdditionalFields: map[string]any{MetaAgentIDKey: "a", "k": 1}}
	got := metaFromMCP(src)
	if got[MetaAgentIDKey] != "a" || got["k"] != 1 {
		t.Errorf("metaFromMCP returned %v", got)
	}
	// Ensure it is a copy, not the same map.
	got["mutate"] = true
	if _, ok := src.AdditionalFields["mutate"]; ok {
		t.Errorf("metaFromMCP did not copy the map")
	}
}
