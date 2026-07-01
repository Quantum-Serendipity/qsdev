package mcpserve

import (
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
