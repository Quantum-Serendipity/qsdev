// Package contentsign provides cryptographic content signing, signature
// verification, Unicode sanitization, datamarking, and provenance metadata for
// qsdev's documentation pipeline (Phase 30).
//
// # Scope and the external-server constraint
//
// qsdev's documentation MCP servers (local-docs-devdocs, local-docs-zim) are
// external third-party processes launched by the AI client from .mcp.json.
// qsdev has no hook into their request/response cycle, so it cannot inject
// sanitization, datamarking, or _meta provenance into their responses at query
// time. The integration points that exist today are therefore:
//
//   - Signing and verifying corpus files on disk (sign.go, verify.go, startup.go).
//   - Sanitizing DevDocs db.json at download time, before the external server
//     indexes it (sanitize.go, pipeline.go).
//   - Backing the "Attested" compliance grade for qsdev-controlled binaries
//     (attestation.go).
//
// The datamarking (datamark.go) and provenance/_meta (provenance.go) helpers are
// fully tested libraries whose response-time wiring is deferred to the
// qsdev-controlled "Universal MCP Server" (roadmap P32); they cannot be applied
// to the external doc servers.
//
// # Cryptography
//
// Signing and verification use the pure-Go aead.dev/minisign library (Minisign
// Ed25519 signatures), so no minisign binary is required on any machine. The
// streaming Reader (Blake2b-512 prehash / HashEdDSA) is used for both signing
// and verification so files of any size are handled uniformly without buffering.
package contentsign
