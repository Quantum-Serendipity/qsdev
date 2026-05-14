# Tasks: MCP Documentation Prompt Injection Hardening

## Phase 1: Scoping & Initial Research

### Pending

### Active

### Completed

- **P1-T4: Server-Side Content Sanitization Approaches**
  - Priority: medium | Estimate: medium
  - What can an MCP server do to sanitize documentation content before serving it? Unicode normalization, control character stripping, structural separation of content from metadata, content diffing against known-good baselines. Tradeoffs between sanitization aggressiveness and documentation fidelity.
  - Started: 2026-05-14
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: Comprehensive analysis in `content-sanitization-research.md`. 16 sources fetched (t4- prefix). Key findings: NFKC is correct normalization form; tag characters U+E0000-U+E007F are most dangerous invisible vector (100% compliance on Claude Opus 4); strict delimiter templates achieve 96.3% defense; datamarking reduces ASR from ~50% to ~1-3% with no task degradation; code blocks need special handling for whitespace-based datamarking; combined server-side defenses estimate 2-5% ASR against non-adaptive, 5-10% against adaptive attacks; residual risk must be addressed at model/app/human layers.


- **P1-T1: Prompt Injection Attack Taxonomy for Documentation Content**
  - Priority: high | Estimate: medium
  - What documentation patterns function as effective prompt injection? Code examples with hidden instructions, configuration snippets with malicious defaults, API descriptions with embedded directives. Categorize by vector type, severity, and detectability. Include real-world examples from security research.
  - Started: 2026-05-14
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: Comprehensive taxonomy in `injection-attack-taxonomy-research.md`. 20 primary sources fetched and saved to docs/. 7 major categories identified with 17+ specific vector types. Categories: direct instruction injection (3 vectors), invisible/hidden text (6 vectors), markdown/formatting exploits (4 vectors), indirect via config/code (4 vectors), multi-step/delayed patterns (4 vectors), data exfiltration (3 vectors), obfuscation/evasion (4 techniques). Highest-priority for MCP documentation: HTML comments in markdown, Unicode invisible characters, malicious code examples, shell command exfiltration. 4 CVEs documented. Key finding: documentation is the ideal injection vehicle because instructional language, executable code examples, and configuration snippets are all legitimate content that overlaps perfectly with attack payloads.

- **P1-T3: RAG Poisoning Prior Art**
  - Priority: high | Estimate: medium
  - Academic and industry research on poisoning retrieval-augmented generation systems. Attack methodologies, demonstrated exploits, defense strategies. Focus on techniques applicable to documentation-as-context (vs. general RAG over arbitrary corpora).
  - Started: 2026-05-14
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: Comprehensive analysis in `rag-poisoning-prior-art-research.md`. 16 primary sources fetched covering foundational papers (Greshake 2023, Zou GCG 2023, PoisonedRAG 2024), defense strategies (Spotlighting, Instruction Hierarchy, Dual LLM/CaMeL, canary tokens, perplexity detection), adaptive attacks that break all 8 tested defenses, MCP-specific security analysis, and applicability mapping to documentation-via-MCP. Key finding: layered defense is mandatory; datamarking is best server-side defense (ASR <3%); no single defense survives adaptive adversaries; documentation's inherently instructional nature is the core unsolved problem.

- **P1-T2: Claude Code MCP Tool-Output Handling & Defenses**
  - Priority: high | Estimate: medium
  - How does Claude Code currently handle MCP tool results? What system-level defenses exist against prompt injection in tool output? How are tool results framed/sandboxed in the context window? What has Anthropic published about their approach?
  - Started: 2026-05-14
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: Comprehensive analysis in `claude-code-mcp-defenses-research.md`. 20 primary sources fetched. Key finding: defenses are multi-layered but tool result content enters context without documented content-level sanitization or structural trust markers. MCP spec silent on prompt injection through tool results.

- **P1-T5: Content-Aware Defense Patterns & Reference Implementations**
  - Priority: medium | Estimate: small
  - Existing tools, libraries, or patterns for detecting/mitigating prompt injection in served content. Prior art from LLM firewall products, guardrail frameworks, content filtering approaches. What's production-ready vs. research-stage?
  - Started: 2026-05-14
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: Comprehensive analysis in `defense-patterns-research.md`. 16 sources fetched. Covers 4 commercial products (Lakera Guard, NeMo Guardrails, OpenAI Guardrails, Guardrails AI), 7 open-source libraries (LLM Guard, Rebuff, Vigil, etc.), 5 architectural patterns (CaMeL, StruQ, SecAlign, Instruction Hierarchy, Dual LLM), 6 content filtering approaches, and MCP-specific defenses. Key finding: datamarking is single highest-impact server-side defense (ASR ~50% → <3%); all ML classifiers suffer documentation false-positive problems; architectural defenses require model-provider changes except tool description framing (free today). Ranked Tier 1: datamarking, tool description trust framing, Unicode sanitization, content provenance metadata.

## Phase 2: Synthesis & Recommendations

### Pending

### Active

### Completed

- **P2-T1: Synthesize Findings into Defense Architecture**
  - Priority: high | Estimate: medium
  - Combine P1 findings into a layered defense recommendation for gdev's MCP documentation pipeline. Map attacks to defenses. Identify which mitigations are practical now vs. require upstream changes.
  - Started: 2026-05-14
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: Full defense architecture in `defense-architecture-research.md`. Five-layer model: L1 content sanitization (ASR ~25%→~13%), L2 structural framing with datamarking (→~2-5%), L3 model-level (→~1%), L4 blast radius reduction, L5 human-in-the-loop. Attack→defense coverage matrix, implementation priority tiers, residual risk assessment. Key insight: combined implementable defenses reduce ASR to ~1-2% non-adaptive / ~5-10% adaptive; semantic misdirection is unsolvable by technical means.

- **P2-T2: Write Executive Summary**
  - Priority: high | Estimate: small
  - Update research.md with complete findings, actionable recommendations, and residual risk assessment.
  - Started: 2026-05-14
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: research.md updated with key findings, defense architecture summary table, implementation priorities, residual risk assessment, open questions, and conclusions. Links all 6 detailed reports.
