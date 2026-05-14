# Research Summary: MCP Content Signing & Verification

## Overview

Investigate cryptographic content signing and verification for MCP server content delivery. Neither ZIM files nor DevDocs data use cryptographic signatures today — integrity depends on download source trust and Nix hash pinning. Research whether content signing (e.g., Sigstore/cosign for ZIM downloads, GPG signatures on DevDocs data) is feasible, what upstream changes would be needed, and whether a verification layer in the MCP server or gdev could detect tampered content. The last-mile integrity gap in an otherwise strong local-first security story.

## Topics

- **ZIM Integrity Mechanisms & Signing Roadmap** — Complete
  - Summary: `zim-signing-research.md`
  - In-file MD5 + SHA-256 sidecars — corruption detection only, no authentication
  - libzim#40 ("content signing") open 9 years, no progress; project lead considers HTTPS sufficient
  - WACZ Auth is the gold standard comparison for signed offline archives
  - Detached `.zim.sig` sidecar is the pragmatic gdev path

- **DevDocs Integrity & Signing Feasibility** — Complete
  - Summary: `devdocs-signing-research.md`
  - Zero integrity mechanisms at any layer; industry-wide gap across all doc aggregators
  - Upstream adoption unlikely (6+ year stale SRI issue)
  - Consumer-side hash pinning via Nix is the practical path

- **Sigstore/Cosign Ecosystem Applicability** — Complete
  - Summary: `sigstore-applicability-research.md`
  - Cosign blob signing works for large files; strong ecosystem adoption (npm, PyPI, K8s)
  - No content-signing precedent anywhere — this would be novel
  - **Recommendation: Minisign as primary** (~200 KB, fully offline, single pubkey string), cosign as optional secondary for CI provenance
  - GPG/TUF/SSH not recommended

- **Nix Hash Pinning — Coverage & Gaps** — Complete
  - Summary: `nix-signing-research.md`
  - Nix SRI hashes guarantee bit-identical content but have TOFU, update-trust, and provenance gaps
  - nixpkgs has rejected build-time signature verification (PR #43233 closed without merge)
  - **Recommendation: verify in CI/update pipeline, then pin hash** — matches Nix philosophy

- **Threat Model for Content Tampering in MCP Documentation Pipeline** — Complete
  - Full model: `docs/threat-model.md`
  - Summary: `threat-model-research.md`
  - 8 attack vectors enumerated, risk-prioritized into 3 tiers
  - Key finding: signing addresses distribution integrity (MITM, mirrors, Nix supply chain) but not the highest-priority risks (compromised source docs, prompt injection)
  - Assessment: signing is a hardening measure, not a critical risk closure; defense-in-depth required

- **MCP Response Provenance & Differential Trust** — Complete
  - Summary: `mcp-provenance-research.md`
  - Detailed findings: `docs/mcp-provenance-research.md`
  - MCP `_meta` field is a spec-compliant open container for custom provenance metadata (on both result envelope and individual content items)
  - Claude Code only processes Anthropic-namespaced `_meta` keys (`anthropic/maxResultSizeChars`, `anthropic/alwaysLoad`); custom fields are invisible to the model
  - No MCP server, AI framework, or specification implements response-level differential trust
  - RAG systems implement trust at retrieval/reranking layer (pre-model), not via in-context metadata signals
  - CoSAI OASIS flags "Missing Integrity/Verification Controls" (MCP-T6) as a critical gap — recommendation only, no implementation
  - Assessment: download-time verification is sufficient; `_meta` provenance is a low-cost forward investment; runtime hash check at MCP server startup catches post-download tampering

- **Verification Architecture & Implementation Recommendation** — Complete
  - Summary: `verification-architecture-research.md`
  - Three-layer architecture: CI signing + MCP startup verification + `_meta` provenance
  - No upstream changes required — gdev acts as re-signing intermediary
  - Minisign as signing tool, 1–3 person-days total effort
  - Residual risks: compromised source docs (partially mitigated by content diffing), prompt injection (out of scope)

## Open Questions

- None. All Phase 1 and Phase 2 tasks complete.

## Conclusions

### The gap

Neither ZIM files nor DevDocs content uses cryptographic signing. ZIM has checksums (MD5 in-file, SHA-256 sidecar) for corruption detection but no authentication. DevDocs has zero integrity mechanisms. No documentation aggregator anywhere implements content signing — this is an industry-wide gap. Upstream is unlikely to add signing: ZIM's signing issue (libzim#40) has been open 9 years with no progress; DevDocs has never discussed it.

### The risk assessment

Signing is a **hardening measure**, not a critical risk closure. The highest-priority threats in gdev's AI-consumer context — compromised source documentation and prompt injection via documentation content — are completely unaffected by cryptographic signing. Signing addresses the medium-priority distribution integrity tier: MITM on download, mirror tampering, and Nix package supply chain compromise. The Nix supply chain vector is the strongest justification — an attacker who compromises the gdev flake to change a content hash would also need to forge a Minisign signature, which they cannot.

### The architecture

**Three verification layers:**

1. **CI/update pipeline** (P0): Download content from upstream → verify upstream checksums → content diff against previous version → Minisign sign → update Nix SRI hash → human-reviewed PR merge. gdev acts as a re-signing intermediary, attesting "we downloaded this from the legitimate source and verified it."

2. **MCP server startup** (P1): Verify Minisign signatures of all content files at process start. Refuse to serve on verification failure. Report to gdev health system. Catches post-download filesystem tampering.

3. **MCP response `_meta`** (P2): Include `gdev/verificationStatus`, `gdev/contentHash`, `gdev/source` in MCP response metadata. Forward investment — Claude Code ignores custom `_meta` today, but the MCP ecosystem is moving toward trust differentiation (CoSAI OASIS flagged this as a critical gap).

### The tool

**Minisign** — ~200 KB binary, fully offline, single Ed25519 public key string embeddable in Nix config, pre-hash mode handles files of any size. Available in nixpkgs. Battle-tested (WireGuard, OpenBSD signify-compatible). Cosign recommended as optional secondary for CI provenance attestation if transparency logging or OIDC identity binding becomes needed.

### The effort

1–3 person-days total. Minisign key generation is one command, CI signing integration is ~20 lines of shell, MCP server verification is ~50 lines of code. Near-zero operational cost given gdev's infrequent content update cadence.

### Residual risks

| Risk | Status after implementation |
|------|---------------------------|
| Compromised source documentation | Partially mitigated (content diffing + human review) |
| Prompt injection via documentation | Not addressed (separate concern) |
| MITM / mirror tampering | Fully mitigated |
| Nix supply chain compromise | Fully mitigated (requires forging both hash and signature) |
| Post-download filesystem tampering | Fully mitigated (MCP startup verification) |
| gdev CI compromise | Mitigated (dual control: hash + signature) |
