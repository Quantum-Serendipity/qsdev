# Verification Architecture & Implementation Recommendation

This document synthesizes Phase 1 findings (T1–T6) into a concrete verification architecture for gdev's MCP documentation pipeline. It covers where verification lives, what upstream changes are needed, what to implement, and what residual risk remains.

## Verification Layer Architecture (P2-T1)

### Where signing and verification live

The architecture has three layers, each addressing different threat tiers:

**Layer 1: CI/Update Pipeline (sign + verify + pin)**
- **When**: gdev maintainers update documentation content (ZIM files, DevDocs bundles)
- **What happens**: 
  1. Download content from upstream (Kiwix, DevDocs)
  2. Verify upstream integrity (SHA-256 sidecar for ZIM; hash-on-first-download for DevDocs)
  3. Sign the content with Minisign (gdev's own key pair)
  4. Record the Nix SRI hash of the signed content in the gdev flake
  5. Commit signature files (`.minisig`) alongside content references
- **What it protects against**: MITM on download, upstream mirror tampering, first-download TOFU (human verifies before signing), Nix supply chain compromise (attacker must forge both hash AND Minisign signature)
- **Why this placement**: Matches nixpkgs philosophy ("once we have our own checksum, PKI is redundant"). Avoids Nix sandbox network restrictions. Leverages gdev's infrequent content update cadence (weekly-to-monthly) — the CI cost per update is negligible.

**Layer 2: MCP Server Startup (runtime hash verification)**
- **When**: MCP server process starts (or content files change on disk)
- **What happens**:
  1. Read Minisign public key from qsdev configuration
  2. Verify `.minisig` signature against each content file
  3. If verification fails: refuse to serve content, log error, report to gdev health system
- **What it protects against**: Post-download filesystem tampering, corrupted content on disk, accidental overwrites
- **Why this placement**: Catches the "local filesystem tampering" threat vector without per-query overhead. One-time cost at startup (~1ms for Minisign signature verification; SHA-256 of large ZIM files takes seconds but only runs once). Integrates naturally with gdev's health reporting.

**Layer 3: MCP Response Metadata (forward investment)**
- **When**: Every MCP tool response
- **What happens**: Include `_meta` provenance fields on responses:
  ```json
  {
    "_meta": {
      "gdev/verificationStatus": "verified",
      "gdev/contentSource": "kiwix:mdn-web-docs-2026-04",
      "gdev/contentHash": "sha256:abc123...",
      "gdev/lastVerified": "2026-05-14T10:00:00Z"
    }
  }
  ```
- **What it protects against**: Nothing today — Claude Code discards custom `_meta` fields. This is a forward investment for when the MCP ecosystem adds trust differentiation (CoSAI OASIS has flagged this as a critical gap).
- **Why include it**: Near-zero implementation cost. Positions gdev as ready for ecosystem evolution. Useful for non-Claude MCP clients that may process metadata. Useful for gdev's own logging/auditing.

### What was rejected and why

| Approach | Reason for rejection |
|----------|---------------------|
| Nix build-time signature verification | nixpkgs philosophy: "once we have our own checksum, no PKI needed." PR #43233 closed without merge. Nix sandbox blocks network (Rekor). |
| Per-query MCP verification | Unnecessary overhead. Startup verification is sufficient — content doesn't change during server lifetime. |
| Embedding provenance in text content | Pollutes documentation content. Model has no mechanism to use it for trust. Attacker who controls content can inject fake provenance. |
| Upstream signature verification (ZIM/DevDocs) | Neither format supports signing. No upstream roadmap. gdev must sign independently. |

## Upstream Change Requirements (P2-T2)

### Required upstream changes: None

gdev's verification architecture operates entirely independently of upstream. No changes are needed from Kiwix, DevDocs, or any content provider. This is by design — the research found:

- **ZIM**: libzim#40 (content signing) has been open 9 years with no progress. Project lead considers HTTPS sufficient. Even if signing were added, the ZIM format would need a spec revision, libzim would need OpenSSL linkage, and all existing ZIM files would remain unsigned.
- **DevDocs**: Zero integrity mechanisms. No upstream discussion of signing. freeCodeCamp/devdocs issue #1113 (SRI for CDN assets — not even content signing) has been stale for 6+ years.
- **Other doc aggregators**: Dash, Zeal, Velocity — none implement content signing. Industry-wide gap.

### Upstream changes that would be beneficial (but aren't prerequisites)

| Upstream change | Benefit | Likelihood |
|----------------|---------|------------|
| Kiwix publishes Minisign/GPG signatures for ZIM files | Eliminates gdev's TOFU problem — upstream signature is ground truth | Very low (9-year-stale issue) |
| DevDocs publishes content hash manifests | Enables automated hash comparison on updates | Very low (no discussion) |
| MCP spec adds standard provenance annotations | gdev's `_meta` fields become interoperable across clients | Possible (CoSAI flagged the gap) |
| Claude Code processes custom `_meta` for trust signals | gdev provenance metadata becomes actionable | Unknown (Anthropic roadmap) |

### gdev's independent signing model

gdev acts as a **re-signing intermediary**:
1. Download upstream content (trusting HTTPS for transport)
2. Human-in-the-loop verification (maintainer reviews content before signing)
3. Sign with gdev's Minisign key (attests: "gdev maintainers verified this content")
4. Distribute signed content + `.minisig` files to users
5. Users verify against gdev's embedded public key

This model means gdev's signature attests **"we downloaded this from the legitimate upstream source and verified it"** — a provenance claim, not an authenticity claim about the original documentation. It's analogous to how Linux distributions re-sign upstream packages with their own GPG key.

## Implementation Feasibility & Recommendation (P2-T3)

### Recommended implementation

**Signing tool: Minisign**
- ~200 KB binary (vs cosign's ~70 MB)
- Available in nixpkgs (`pkgs.minisign`)
- Single public key string embeds in gdev's Nix configuration
- Pre-hash mode (`-H`) handles files of any size
- Fully offline — no infrastructure dependency
- Battle-tested: used by WireGuard, OpenBSD signify-compatible

**Key management:**
- One Ed25519 key pair for gdev
- Private key in CI secrets (GitHub Actions secret or equivalent)
- Public key hardcoded in gdev's Nix flake (single string, e.g., `RWQ...`)
- Key rotation: generate new key, publish new public key in qsdev update, sign content with both keys during transition period

**Signature format:**
- Detached `.minisig` files alongside content (e.g., `mdn-web-docs-2026-04.zim.minisig`)
- For DevDocs: sign the JSON data file; manifest maps `{slug, version, data_hash, signature_file}`
- Store signatures in the gdev content repository or alongside Nix derivation inputs

**CI workflow (content update):**
```
1. Download ZIM/DevDocs from upstream
2. Verify upstream checksums (SHA-256 sidecar for ZIM)
3. Content diff against previous version (detect unexpected changes)
4. Minisign sign: `minisign -S -H -s $SECRET_KEY -m content.zim`
5. Update Nix SRI hash in flake
6. Commit: content reference + .minisig + updated hash
7. PR review by maintainer (human-in-the-loop)
```

**MCP server startup verification:**
```
1. Read gdev Minisign public key from config
2. For each content file:
   a. Locate corresponding .minisig
   b. `minisign -V -p $PUBLIC_KEY -m content.zim` (or equivalent library call)
   c. If fail: log error, refuse to serve, report to gdev health
3. Cache verification status in memory
4. Include verification status in _meta on all responses
```

**Content diffing at update time (defense-in-depth):**
- For DevDocs JSON: structural diff of index entries, flag unexpected page additions/removals/modifications
- For ZIM: compare article counts, check for new/removed articles, flag unexpected changes outside the documented upstream changelog
- Purpose: detect compromised-source-documentation attacks (the #1 threat that signing cannot address)

### What to implement now vs later

| Component | Priority | Effort | Implement when |
|-----------|----------|--------|---------------|
| Minisign key pair generation | P0 | Trivial | Now |
| CI signing workflow | P0 | Small | Now |
| Nix hash pinning (already exists) | P0 | Zero | Already done |
| MCP server startup verification | P1 | Small | With MCP server implementation |
| Content diffing at update time | P1 | Medium | With CI signing workflow |
| `_meta` provenance in MCP responses | P2 | Trivial | With MCP server implementation |
| Cosign CI provenance attestation | P3 | Small | When/if provenance auditing is needed |

### What NOT to implement

- **Build-time Nix signature verification**: Against nixpkgs philosophy, adds complexity, Nix hash already catches tampering.
- **Per-query MCP verification**: Startup verification is sufficient; per-query adds latency for zero security benefit.
- **Custom MCP trust differentiation protocol**: No client would consume it. Wait for ecosystem standardization.
- **Upstream contribution to ZIM/DevDocs signing**: Low-probability-of-impact use of engineering time. Monitor libzim#40 and CoSAI OASIS instead.

### Residual risk after implementation

| Risk | Severity | Mitigation status |
|------|----------|-------------------|
| Compromised source documentation (e.g., malicious MDN edit gets scraped) | Medium-High | **Partially mitigated** by content diffing at update time + human review. Cannot be fully eliminated — legitimate content changes are indistinguishable from subtle poisoning. |
| Prompt injection via documentation content | Medium-High | **Not addressed** by this spike's scope. Requires MCP-level content sanitization or Claude Code-side defenses. Flagged as separate concern. |
| gdev CI compromise (attacker has signing key) | Medium | **Mitigated** by Nix hash pinning (attacker must compromise both CI and the Nix flake commit). GitHub Actions OIDC + cosign transparency log would further mitigate (P3 item). |
| Key compromise (Minisign private key leaked) | Low-Medium | **Mitigated** by key rotation procedure. Transparency: all signed content is in version control, so a compromised key's blast radius is auditable. |
| Post-download filesystem tampering | Low | **Fully mitigated** by MCP server startup verification. |
| MITM/mirror tampering | Low | **Fully mitigated** by Minisign signatures + Nix hashes. |

### Cost assessment

The entire signing infrastructure is **small effort** (1–3 person-days):
- Minisign is a single binary with a trivial CLI
- Key generation is one command
- CI integration is ~20 lines of shell in a GitHub Actions workflow
- MCP server verification is ~50 lines of code (shell out to `minisign -V` or use a library)
- `_meta` provenance is a few extra fields in JSON response construction

The operational cost is near-zero: content updates happen infrequently, signing is fast, and verification is a one-time startup cost.
