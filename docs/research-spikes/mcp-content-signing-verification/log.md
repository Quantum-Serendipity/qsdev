# Research Log: MCP Content Signing & Verification

## 2026-05-14 — Spike Promoted from Pending
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: Spike scaffolded from pending-spikes.md entry. Investigates cryptographic content signing and verification for MCP server content delivery — the last-mile integrity gap in the local-first documentation strategy. Neither ZIM files nor DevDocs data use cryptographic signatures today; integrity depends on download source trust and Nix hash pinning.
- **Next**: Confirm research question with user; populate Phase 1 tasks in tasks.md.

## 2026-05-14 — Phase 1 Tasks Populated
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: Populated 6 Phase 1 tasks (ZIM integrity + signing roadmap, DevDocs integrity + signing feasibility, Sigstore/cosign applicability, Nix hash pinning coverage/gaps, threat model, MCP response provenance) and 3 Phase 2 tasks (verification layer architecture, upstream change requirements, implementation recommendation). Parent spike context reviewed — key gaps are: no crypto signing on ZIM or DevDocs, Nix hashes cover post-recording integrity but not first-download provenance, no attestation at MCP query time.
- **Next**: Begin Phase 1 research — launch parallel sub-agents for T1-T5.

## 2026-05-14 — Threat Model Complete (P1-T5)
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Built structured threat model covering 8 attack vectors (compromised upstream, compromised source docs, MITM, unofficial mirrors, Nix package compromise, local filesystem tampering, MCP server compromise, prompt injection via content). Mapped each against signing effectiveness and existing mitigations. Risk prioritization places compromised source documentation and prompt injection as highest-priority (neither addressed by signing), with Nix package supply chain compromise as the top vector where signing provides strong value. Assessment: signing is a meaningful hardening measure for distribution integrity, not a critical risk closure. Defense-in-depth beyond signing is essential.
- **Artifacts**: `docs/threat-model.md` (full model, ~4000 words), `threat-model-research.md` (summary, ~1400 words)
- **Next**: Continue Phase 1 — P1-T1 through P1-T4 and P1-T6 remain pending.

## 2026-05-14 — Phase 1 Tasks T1-T4 Complete; T6 Launched
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Summary**: Five of six Phase 1 tasks completed via parallel sub-agents. Key findings converging:
  - **T1 (ZIM)**: libzim#40 (content signing) open 9 years with no progress. WACZ Auth is the gold standard comparison. Detached `.zim.sig` sidecar is the pragmatic path.
  - **T2 (DevDocs)**: Zero integrity anywhere. Industry-wide gap — no doc aggregator signs content.
  - **T3 (Sigstore)**: Minisign recommended as primary signing tool (~200 KB, fully offline). Cosign as optional secondary for CI provenance.
  - **T4 (Nix)**: nixpkgs rejected build-time signature verification. CI-pipeline verification + Nix hash pinning is the recommended layering.
  - **T5 (Threat model)**: Signing is a hardening measure for distribution integrity, not a critical risk closure. Highest-priority threats (source doc compromise, prompt injection) need other controls.
- **Artifacts**: 4 research summaries, 1 threat model, 20+ source docs saved to docs/
- **Next**: T6 (MCP response provenance) running. Once complete, proceed to Phase 2 synthesis.

## 2026-05-14 — MCP Response Provenance Research Complete (P1-T6)
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: MCP Spec 2025-11-25 (tools, schema.json), CoSAI OASIS MCP Security Analysis, 2026 MCP Roadmap, Claude Code MCP docs, Lakera MCP trust analysis, Stacklok server trust, RAG trust frameworks survey → 8 docs saved to `docs/`
- **Summary**: MCP's `_meta` field (open-schema object on CallToolResult and content items) is the protocol-compliant mechanism for provenance metadata. A gdev server could include verification status, content hashes, and source URLs without violating the spec. However, Claude Code only processes Anthropic-namespaced `_meta` keys — custom fields are invisible to the model. No MCP server, AI framework, or specification implements response-level differential trust. RAG systems implement trust at the retrieval/reranking layer (pre-model), not via in-context metadata signals. The CoSAI OASIS security analysis flags "Missing Integrity/Verification Controls" as a critical gap but no implementation exists. **Assessment**: Download-time verification (Minisign + Nix hashes) is sufficient. MCP-level provenance is a low-cost forward investment but not currently actionable for trust decisions. Runtime hash verification at MCP server startup is the practical complement — catches post-download tampering without protocol changes.
- **Artifacts**: `mcp-provenance-research.md` (summary), `docs/mcp-provenance-research.md` (detailed findings), 8 source docs
- **Next**: All Phase 1 tasks complete. Proceed to Phase 2 synthesis (P2-T1: verification layer architecture, P2-T2: upstream change requirements, P2-T3: implementation recommendation).

## 2026-05-14 — Phase 2 Synthesis Complete (P2-T1, P2-T2, P2-T3)
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Synthesized all Phase 1 findings into a single architecture document covering verification layer placement, upstream requirements, and implementation recommendation. Three-layer architecture: (1) CI/update pipeline with Minisign signing + Nix hash pinning, (2) MCP server startup verification, (3) `_meta` provenance as forward investment. No upstream changes required — gdev acts as re-signing intermediary. Minisign selected as signing tool. Total effort: 1–3 person-days. Residual risks documented: compromised source docs partially mitigated by content diffing, prompt injection out of scope.
- **Artifacts**: `verification-architecture-research.md` (full synthesis, ~2500 words)
- **Next**: Spike ready for completion. All Phase 1 and Phase 2 tasks done. Run depth checklist review before closing.

## 2026-05-14 — Spike Completed
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Spike completed after depth checklist review (all 7 reports pass all 6 criteria). Key conclusions: (1) Neither ZIM nor DevDocs has signing; upstream won't add it — industry-wide gap. (2) Signing is a hardening measure, not a critical risk closure — highest-priority threats (source doc compromise, prompt injection) are unaffected. (3) Recommended architecture: Minisign signing in CI + Nix hash pinning + MCP server startup verification + `_meta` provenance as forward investment. (4) No upstream changes required — gdev acts as re-signing intermediary. (5) Total effort: 1–3 person-days. One follow-on candidate flushed to proposed-spikes.md (prompt injection hardening for MCP documentation).
