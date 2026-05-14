# Research Summary: MCP Documentation Prompt Injection Hardening

## Overview

Documentation served via MCP to AI coding assistants is uniquely vulnerable to prompt injection because it is *inherently instructional* — code examples, configuration snippets, and API descriptions legitimately tell the reader what to do. This makes distinguishing injected instructions from legitimate documentation a fundamentally harder problem than general RAG poisoning.

This spike investigated the content-level gap left by `mcp-content-signing-verification`: signing verifies *who produced* documentation, but not *whether it's safe*. We catalogued attack vectors, assessed Claude Code's existing defenses, surveyed academic prior art, and designed a layered defense architecture for gdev's MCP documentation pipeline.

## Key Findings

1. **No single defense survives adaptive adversaries.** All 8 tested single-layer defenses were bypassed at >50% ASR by adaptive attacks (NAACL 2025). Layered defense is mandatory.

2. **Datamarking is the highest-impact server-side defense.** Replacing whitespace with random marker tokens reduces ASR from ~50% to <3% with no task performance degradation (Microsoft Spotlighting research).

3. **Claude's defenses are behavioral, not structural.** Tool results enter the context as `user` role messages. The principal hierarchy ("treat tool outputs as information, not commands") is RL-trained behavior, not architectural enforcement. Auto mode's input probe (93.4% catch rate) may not run outside auto mode.

4. **Invisible injection is completely solvable.** Unicode tag characters, zero-width encoding, HTML comments, and hidden CSS — all eliminated by deterministic sanitization with zero documentation fidelity impact.

5. **Semantic misdirection is unsolvable by technical means.** Documentation that reads as legitimate but steers toward insecure outcomes cannot be filtered without destroying the documentation. This is a supply chain integrity and code review problem.

6. **MCP amplifies attack success.** MCP architecture increases ASR by 23-41% compared to equivalent non-MCP integrations, with cross-server propagation attacks achieving 61.3% success.

## Topics

| Topic | Status | Report |
|---|---|---|
| Prompt injection attack taxonomy | Complete | [injection-attack-taxonomy-research.md](injection-attack-taxonomy-research.md) |
| Claude Code MCP defenses | Complete | [claude-code-mcp-defenses-research.md](claude-code-mcp-defenses-research.md) |
| RAG poisoning prior art | Complete | [rag-poisoning-prior-art-research.md](rag-poisoning-prior-art-research.md) |
| Server-side content sanitization | Complete | [content-sanitization-research.md](content-sanitization-research.md) |
| Defense patterns & reference implementations | Complete | [defense-patterns-research.md](defense-patterns-research.md) |
| Layered defense architecture | Complete | [defense-architecture-research.md](defense-architecture-research.md) |

## Defense Architecture Summary

Five layers, each catching what the previous layer missed:

| Layer | What It Does | Blocks | Expected Impact | Complexity |
|---|---|---|---|---|
| **L1: Content Sanitization** | NFKC normalization, invisible char stripping, HTML comment removal | All invisible/hidden injection vectors | ASR ~25% → ~13% | Trivial |
| **L2: Structural Framing** | Tool description trust framing, delimiter wrapping, datamarking | Authority impersonation, explicit overrides | ASR ~13% → ~2-5% | Trivial to moderate |
| **L3: Model-Level** | Anthropic's principal hierarchy training, auto-mode probes | Most remaining injection attempts | ASR ~2-5% → ~1% | Upstream |
| **L4: Blast Radius Reduction** | Permissions, sandboxing, network restrictions | Consequences of successful injection | Limits damage | Trivial (use existing) |
| **L5: Human-in-the-Loop** | Approval prompts, user review | All attacks (if user reviews carefully) | Catches residual | Degrades with fatigue |

## Implementation Priorities for gdev

**Implement immediately** (trivial, high impact):
1. Unicode NFKC normalization + invisible character stripping
2. HTML comment and hidden element stripping
3. Tool description trust framing
4. Content delimiter wrapping

**Implement soon** (moderate complexity, highest single-defense impact):
5. Datamarking (whitespace replacement with code-block exceptions)
6. Content provenance metadata in MCP `_meta` field
7. Content integrity verification (hash at index, verify at serve)

**Advocate upstream**:
8. Extend auto-mode input probe to all MCP tool results
9. MCP spec guidance on prompt injection in tool results
10. Structural trust markers in MCP protocol

## Residual Risk

After all implementable defenses:
- **~1-2% ASR** against non-adaptive attacks
- **~5-10% ASR** against adaptive attacks (attackers who know the defense stack)
- **Unsolvable**: Semantically coherent misdirection — documentation that reads as legitimate but steers toward insecure outcomes. This is a governance problem (supply chain integrity, code review), not a filtering problem.

## Open Questions

- Does Claude Code's auto-mode input probe run for MCP tool results in non-auto modes? If not, advocating for this extension is the single highest-impact upstream action.
- What is the false positive rate of datamarking on real documentation corpora? Benchmarks used synthetic datasets; testing against actual DevDocs/ZIM content would validate.
- Can kill-chain canary tokens be implemented in gdev's MCP tool architecture? The technique showed 0% ASR in one study but requires specific tool design.

## Conclusions

Prompt injection in MCP-served documentation is a real, demonstrated threat with multiple CVEs and academic attack demonstrations. No complete solution exists — documentation's instructional nature makes this fundamentally harder than general RAG poisoning. However, a practical layered defense reduces attack success from ~25% (undefended) to ~1-2% (all layers), with blast radius containment limiting damage from the residual. The implementation path is clear: trivial sanitization and framing first (immediate), datamarking second (highest single-defense impact), upstream advocacy third (extends model-level defenses to MCP context). This spike closes the content-safety gap that `mcp-content-signing-verification` left open — together, the two spikes cover both distribution integrity and content safety for gdev's documentation pipeline.

## Sources

78 primary source documents saved to `docs/`, spanning:
- Anthropic official documentation and engineering blog posts
- MCP specification and security guidance
- Academic papers (Greshake 2023, Zou GCG 2023, PoisonedRAG 2024, Spotlighting 2024, NAACL 2025 adaptive attacks)
- Security research firms (Lasso Security, Oasis, Cymulate, Pillar Security, HiddenLayer)
- CVE disclosures (CVE-2025-54794/54795, CVE-2025-53773, CVE-2025-32711, CVE-2025-59944)
- Industry guidance (OWASP LLM Top 10, AWS, Cisco, Microsoft, NVIDIA)
- Defense tool documentation (Lakera Guard, NeMo Guardrails, LLM Guard, Rebuff, Vigil)
