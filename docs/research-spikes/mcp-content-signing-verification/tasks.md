# Tasks: MCP Content Signing & Verification

## Phase 1: Scoping & Initial Research

### Pending

### Active

### Completed

- **P1-T6: MCP response provenance and differential trust**
  - Priority: medium
  - Estimate: medium
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: MCP `_meta` field on CallToolResult and content items is a fully open extensibility container — provenance metadata is spec-compliant. However, Claude Code only processes Anthropic-namespaced `_meta` keys; custom fields are invisible to the model. No MCP server, AI framework, or specification implements response-level provenance or differential trust. RAG systems implement pre-model trust (reranking/filtering) not in-context trust signals. **Recommendation: download-time verification (Minisign + Nix hashes) is sufficient; include `_meta` provenance as low-cost forward investment; runtime hash verification at MCP server startup catches post-download tampering.**
  - Output: `mcp-provenance-research.md`, `docs/mcp-provenance-research.md` + 8 source docs

- **P1-T2: DevDocs integrity mechanisms and signing feasibility**
  - Priority: high
  - Estimate: medium
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: Zero integrity mechanisms at any layer. One stale SRI issue (#1113, 6+ years open). No doc aggregator (Dash/Zeal/Velocity) implements signing — industry-wide gap. Consumer-side hash pinning via Nix is the practical path.
  - Output: `devdocs-signing-research.md`, `docs/devdocs-integrity-mechanisms.md`

- **P1-T3: Sigstore/cosign ecosystem applicability**
  - Priority: high
  - Estimate: medium
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: Cosign blob signing works for large files (signs digest). Strong npm/PyPI/K8s adoption but no content-signing precedent anywhere. Nix integration possible but sandbox blocks Rekor online verification. **Recommendation: Minisign as primary** (~200 KB binary, fully offline, single public key string) with cosign as optional secondary for CI provenance attestation. GPG/TUF/SSH not recommended.
  - Output: `sigstore-applicability-research.md`, `docs/sigstore-ecosystem-research.md` + 10 source docs

- **P1-T1: ZIM integrity mechanisms and signing roadmap**
  - Priority: high
  - Estimate: medium
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: In-file MD5 (16 bytes at EOF) + SHA-256 sidecars at download.kiwix.org — neither provides authentication. libzim#614 (write-corruption) open since 2021, stalled. libzim#40 ("Add spec to allow content signing") open since 2017, no assignee, no spec, no implementation — project lead considers HTTPS sufficient. WACZ Auth is the gold standard comparison (ECDSA + domain-ownership + timestamps). Detached `.zim.sig` sidecar is the pragmatic gdev path — zero format changes needed.
  - Output: `zim-signing-research.md`, `docs/zim-integrity-mechanisms.md` + 8 source docs

- **P1-T4: Nix hash pinning — coverage and gaps**
  - Priority: medium
  - Estimate: small
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: Nix SRI hashes guarantee bit-identical content after correct hash recorded but have 4 gaps: first-download TOFU, hash-update trust, compromised nixpkgs commits, no provenance. nixpkgs PR #43233 (GPG verification) was the most serious attempt — closed without merge. No Sigstore precedent in nixpkgs. Scott Worley pattern (runCommand + gpg --verify) works today. **Recommendation: verify in CI/update pipeline, then pin hash** — matches nixpkgs philosophy and gdev's infrequent update cadence.
  - Output: `nix-signing-research.md`, `docs/nix-hash-pinning-analysis.md` + 6 source docs

- **P1-T5: Threat model for content tampering in MCP documentation pipeline**
  - Priority: high
  - Estimate: small
  - Started: 2026-05-14
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: Full threat model with 8 attack vectors, 5 impact categories, 7 existing mitigations, gap analysis table, risk prioritization (3 tiers), and 6 recommended defense-in-depth layers. Key finding: signing is a meaningful hardening measure for distribution integrity but does not address the highest-priority risks (compromised source documentation, prompt injection), which require content-level mitigations. See `docs/threat-model.md` (full model) and `threat-model-research.md` (summary).

## Phase 2: Architecture & Recommendations

### Pending

### Active

### Completed

- **P2-T1: Verification layer architecture — where signing lives**
  - Priority: high
  - Estimate: medium
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: Three-layer architecture: (1) CI/update pipeline — Minisign signing + Nix hash pinning at content update time; (2) MCP server startup — runtime Minisign verification, refuse to serve on failure; (3) MCP response `_meta` — provenance fields as forward investment (Claude Code ignores custom `_meta` today). Rejected: Nix build-time verification (against nixpkgs philosophy), per-query verification (unnecessary overhead), upstream signature verification (no upstream support).
  - Output: `verification-architecture-research.md`

- **P2-T2: Upstream change requirements**
  - Priority: high
  - Estimate: small
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: **No upstream changes required.** gdev operates as a re-signing intermediary — downloads from upstream, verifies via HTTPS + checksums, signs with gdev's own Minisign key. Beneficial upstream changes (Kiwix Minisign signatures, DevDocs hash manifests, MCP provenance spec) are all very low likelihood. gdev's model is analogous to Linux distros re-signing upstream packages.
  - Output: `verification-architecture-research.md`

- **P2-T3: Implementation feasibility and recommendation**
  - Priority: high
  - Estimate: medium
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: **Minisign** as signing tool (~200 KB, offline, single pubkey string). CI workflow: download → verify upstream checksums → content diff → Minisign sign → update Nix hash → PR review. MCP server: verify at startup, refuse on failure, report to health system. Total effort: 1–3 person-days. Residual risks: compromised source docs (partially mitigated by content diffing), prompt injection (out of scope — needs separate mitigation).
  - Output: `verification-architecture-research.md`
