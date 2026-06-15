package contentsign

import "time"

// Provenance metadata attaches a verifiable origin record to a documentation
// response so a downstream consumer can tell whether the content was
// signature-verified, hash-verified, or unverified, and where it came from.
//
// # P32-deferred library
//
// mcp-go v0.54.1 supports per-result _meta via Result.Meta and
// mcp.NewMetaFromMap, but qsdev's embedded MCP handlers return plain strings
// today (see mcp.NewToolResultText usage in addons/claudecode/mcp_command.go).
// Wiring ToMeta's map onto a tool result's _meta is therefore future work for
// the qsdev-controlled "Universal MCP Server" (roadmap P32); this file is a
// tested library only.

// MCP _meta key names, namespaced under "qsdev/" so they do not collide with
// other producers' metadata on the same result.
const (
	metaKeyVerificationStatus = "qsdev/verificationStatus"
	metaKeyContentHash        = "qsdev/contentHash"
	metaKeySource             = "qsdev/source"
	metaKeyLastVerified       = "qsdev/lastVerified"
)

// ToMeta renders the provenance as the qsdev/-namespaced map that becomes the
// MCP result's _meta fields. All four keys are always present; empty values are
// emitted as empty strings so consumers can distinguish "known empty" from
// "absent" only by the value, keeping the shape stable.
func (p ProvenanceMetadata) ToMeta() map[string]any {
	return map[string]any{
		metaKeyVerificationStatus: p.VerificationStatus,
		metaKeyContentHash:        p.ContentHash,
		metaKeySource:             p.Source,
		metaKeyLastVerified:       p.LastVerified,
	}
}

// BuildProvenanceMetadata maps a VerificationResult, the content's source label,
// and its content hash into a ProvenanceMetadata. The verification status is
// taken directly from result.Status (already one of the Status* constants) and
// LastVerified is result.CheckedAt formatted as RFC 3339.
func BuildProvenanceMetadata(result VerificationResult, source, contentHash string) ProvenanceMetadata {
	return ProvenanceMetadata{
		VerificationStatus: result.Status,
		ContentHash:        contentHash,
		Source:             source,
		LastVerified:       result.CheckedAt.Format(time.RFC3339),
	}
}
