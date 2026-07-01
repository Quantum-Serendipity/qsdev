package mcpserve

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// MetaAgentIDKey is the reverse-DNS _meta key for the OPTIONAL, qsdev-specific
// per-request agent-id extension. Its absence is normal: standard clients do
// not set it, and identity then falls back to the initialize handshake's
// clientInfo. This is an app-level extension to the MCP protocol, not part of
// the protocol itself.
//
// SECURITY: the _meta override is a self-asserted LABEL with NO security weight
// once a cryptographically-verified transport identity exists (an mTLS
// client-certificate CN/SAN; see authoritativeAgentID). A client can set this
// key to any value, so it must never be allowed to override a verified cert. It
// is honored only on the local trusted (stdio) path, where there is no transport
// identity to verify against.
const MetaAgentIDKey = "com.quantumserendipity.qsdev/agentId"

// unknownAgentID is used when neither the _meta extension nor clientInfo yields
// a usable identity.
const unknownAgentID = "unknown"

// clientInfoToSPI converts an mcp-go Implementation (from the initialize
// handshake's clientInfo) into the neutral spi.ClientInfo.
func clientInfoToSPI(info mcp.Implementation) spi.ClientInfo {
	return spi.ClientInfo{
		Name:    info.Name,
		Version: info.Version,
		Title:   info.Title,
	}
}

// agentIDFromMeta extracts the optional per-request agent id from a request's
// _meta map. The boolean reports whether the extension key was present with a
// non-empty string value. A nil meta or absent key returns ("", false), which
// callers treat as "no override".
func agentIDFromMeta(meta map[string]any) (string, bool) {
	if meta == nil {
		return "", false
	}
	raw, ok := meta[MetaAgentIDKey]
	if !ok {
		return "", false
	}
	s, ok := raw.(string)
	if !ok || s == "" {
		return "", false
	}
	return s, true
}

// resolveAgentID computes the SELF-ASSERTED agent identity for a request — the
// identity the client claims, used on the local trusted (stdio) path where there
// is no transport identity to verify against.
//
// Resolution order:
//  1. The per-request _meta override (com.quantumserendipity.qsdev/agentId),
//     when present and non-empty.
//  2. The client's reported Name from the initialize handshake.
//  3. The literal "unknown" when neither is available.
//
// SECURITY: every source here is self-asserted and carries no security weight in
// the presence of a verified transport identity. Callers that may run behind an
// authenticating transport (mTLS) MUST go through authoritativeAgentID, which
// lets a verified cert override this value.
func resolveAgentID(client spi.ClientInfo, meta map[string]any) string {
	if id, ok := agentIDFromMeta(meta); ok {
		return id
	}
	if client.Name != "" {
		return client.Name
	}
	return unknownAgentID
}

// authoritativeAgentID returns the effective, authoritative agent identity for a
// request. A cryptographically-verified transport identity (an mTLS
// client-certificate CN/SAN injected into ctx by the cert-identity middleware)
// is AUTHORITATIVE and overrides everything else: it cannot be forged by a
// client. Only when no such identity is present (the stdio / local trusted path)
// does it fall back to the self-asserted resolveAgentID(client, meta). The
// _meta override and clientInfo.Name are therefore non-security LABELS whenever a
// verified cert identity exists.
func authoritativeAgentID(ctx context.Context, client spi.ClientInfo, meta map[string]any) string {
	if trusted, ok := trustedAgentFromContext(ctx); ok {
		return trusted
	}
	return resolveAgentID(client, meta)
}

// metaFromMCP flattens an mcp.Meta into the neutral map[string]any carried on a
// spi.ToolRequest. It returns nil when there is nothing to carry.
func metaFromMCP(meta *mcp.Meta) map[string]any {
	if meta == nil || len(meta.AdditionalFields) == 0 {
		return nil
	}
	out := make(map[string]any, len(meta.AdditionalFields))
	for k, v := range meta.AdditionalFields {
		out[k] = v
	}
	return out
}
