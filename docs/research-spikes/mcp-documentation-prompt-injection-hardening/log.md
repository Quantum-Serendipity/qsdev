# Research Log: MCP Documentation Prompt Injection Hardening

## 2026-05-14 — Spike Promoted from Proposed
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: Spike scaffolded from proposed-spikes.md entry (MCP Content Signing & Verification Follow-Ons section). Awaiting scope confirmation and Phase 1 task decomposition.
- **Next**: Confirm research question with user; populate Phase 1 tasks in tasks.md.

## 2026-05-14 — P1-T2: Claude Code MCP Tool-Output Handling & Defenses
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Anthropic Prompt Injection Defenses](https://www.anthropic.com/research/prompt-injection-defenses) → `docs/anthropic-prompt-injection-defenses.md`
  - [Claude Code Sandboxing](https://www.anthropic.com/engineering/claude-code-sandboxing) → `docs/claude-code-sandboxing.md`
  - [MCP Security Best Practices](https://modelcontextprotocol.io/docs/tutorials/security/security_best_practices) → `docs/mcp-security-best-practices.md`
  - [Claude Code Security Docs](https://code.claude.com/docs/en/security) → `docs/claude-code-security-docs.md`
  - [Claude Code Permissions](https://code.claude.com/docs/en/permissions) → `docs/claude-code-permissions-docs.md`
  - [Anthropic Mitigate Jailbreaks](https://platform.claude.com/docs/en/test-and-evaluate/strengthen-guardrails/mitigate-jailbreaks) → `docs/anthropic-mitigate-jailbreaks-api-docs.md`
  - [Claude Code Auto Mode](https://www.anthropic.com/engineering/claude-code-auto-mode) → `docs/claude-code-auto-mode-safety.md`
  - [arxiv:2601.17548 Agentic Coding Assistants](https://arxiv.org/html/2601.17548v1) → `docs/arxiv-prompt-injection-agentic-coding-assistants.md`
  - [Simon Willison MCP Security](https://simonwillison.net/2025/Apr/9/mcp-prompt-injection/) → `docs/simonwillison-mcp-prompt-injection.md`
  - [Claude Code Architecture](https://www.penligent.ai/hackinglabs/inside-claude-code-the-architecture-behind-tools-memory-hooks-and-mcp/) → `docs/claude-code-mcp-architecture-penligent.md`
  - [InversePrompt CVEs](https://cymulate.com/blog/cve-2025-547954-54795-claude-inverseprompt/) → `docs/inverseprompt-cve-2025-54794-54795.md`
  - [Lasso Indirect Injection](https://www.lasso.security/blog/the-hidden-backdoor-in-claude-coding-assistant) → `docs/lasso-indirect-prompt-injection-claude-code.md`
  - [Oasis Data Exfiltration](https://www.oasis.security/blog/claude-ai-prompt-injection-data-exfiltration-vulnerability) → `docs/oasis-claude-prompt-injection-data-exfiltration.md`
  - [Trust Prompt RCE](https://www.theregister.com/security/2026/05/07/claude-code-trust-prompt-can-trigger-one-click-rce/5235319) → `docs/register-claude-code-trust-prompt-rce.md`
  - [API Tool Use](https://platform.claude.com/docs/en/agents-and-tools/tool-use/overview) → `docs/anthropic-api-tool-use-overview.md`
  - [API Handle Tool Calls](https://platform.claude.com/docs/en/agents-and-tools/tool-use/handle-tool-calls) → `docs/anthropic-api-handle-tool-calls.md`
  - [API How Tool Use Works](https://platform.claude.com/docs/en/agents-and-tools/tool-use/how-tool-use-works) → `docs/anthropic-api-how-tool-use-works.md`
  - [Soul Document](https://gist.github.com/Richard-Weiss/efe157692991535403bd7e7fb20b6695) → `docs/claude-soul-document-trust-hierarchy.md`
  - [Anthropic Constitution](https://www.anthropic.com/constitution) → `docs/anthropic-constitution-principal-hierarchy.md`
  - [Source Leak Analysis](https://claudefa.st/blog/guide/mechanics/claude-code-source-leak) → `docs/claude-code-source-leak-analysis.md`
- **Summary**: Comprehensive investigation of Claude Code's MCP tool-output handling and prompt injection defenses. Key findings:
  1. Tool results are embedded in `user` role messages as `tool_result` content blocks -- no separate `tool` role like OpenAI
  2. Anthropic's constitution explicitly states tool outputs are treated as "information rather than commands" (trained behavior, not structural)
  3. Auto mode has a two-layer safety system (input probe + output classifier) but this may not run outside auto mode
  4. Permission system and sandboxing provide hard blast-radius limits independent of model behavior
  5. MCP spec security guidance focuses on OAuth/transport, entirely silent on prompt injection through tool results
  6. No documented content-level sanitization or structural trust framing for MCP tool results
  7. Academic research shows 78-93% adaptive bypass rates against all evaluated injection defenses
  8. Multiple real-world CVEs demonstrated practical exploitation of Claude Code's trust boundaries
- **Next**: This feeds into P2-T1 synthesis. Remaining P1 tasks (T1, T3, T4, T5) address adjacent topics.

## 2026-05-14 — P1-T3: RAG Poisoning Prior Art Research Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Greshake et al. 2023 — Indirect Prompt Injection](https://arxiv.org/abs/2302.12173) → `docs/greshake-indirect-prompt-injection-2023.md`
  - [Zou et al. 2023 — Universal Adversarial Suffixes](https://arxiv.org/abs/2307.15043) → `docs/zou-adversarial-suffixes-2023.md`
  - [PoisonedRAG 2024](https://arxiv.org/abs/2402.07867) → `docs/poisonedrag-arxiv-abstract.md`, `docs/poisonedrag-full-details.md`
  - [OpenAI Instruction Hierarchy 2024](https://arxiv.org/html/2404.13208v1) → `docs/openai-instruction-hierarchy-2024.md`
  - [Microsoft Spotlighting 2024](https://arxiv.org/html/2403.14720v1) → `docs/microsoft-spotlighting-defense-2024.md`
  - [Adaptive Attacks Break IPI Defenses 2025](https://arxiv.org/html/2503.00061v2) → `docs/adaptive-attacks-break-ipi-defenses-2025.md`
  - [OWASP LLM01 2025](https://genai.owasp.org/llmrisk/llm01-prompt-injection/) → `docs/owasp-llm01-prompt-injection-2025.md`
  - [tldrsec Prompt Injection Defenses](https://github.com/tldrsec/prompt-injection-defenses) → `docs/tldrsec-prompt-injection-defenses.md`
  - [Simon Willison — MCP Prompt Injection 2025](https://simonwillison.net/2025/Apr/9/mcp-prompt-injection/) → `docs/willison-mcp-prompt-injection-2025.md`
  - [MCP Protocol Security Analysis 2025](https://arxiv.org/html/2601.17549v1) → `docs/mcp-protocol-security-analysis-2025.md`
  - [Deconvolute — RAG/MCP Attack Surfaces](https://deconvoluteai.com/blog/attack-surfaces-rag) → `docs/deconvolute-rag-mcp-attack-surfaces.md`
  - [Anthropic Prompt Injection Defenses 2025](https://www.anthropic.com/research/prompt-injection-defenses) → `docs/anthropic-prompt-injection-defenses-2025.md`
  - [CaMeL Framework — DeepMind 2025](https://afine.com/llm-security-prompt-injection-camel) → `docs/camel-framework-deepmind-2025.md`
  - [Simon Willison — Dual LLM Pattern 2023](https://simonwillison.net/2023/Apr/25/dual-llm-pattern/) → `docs/willison-dual-llm-pattern-2023.md`
  - [Microsoft MCP Indirect Injection Defense 2025](https://developer.microsoft.com/blog/protecting-against-indirect-injection-attacks-mcp) → `docs/microsoft-mcp-indirect-injection-defense-2025.md`
- **Summary**: Comprehensive research on RAG poisoning prior art. Covered 5 foundational papers, 5 attack methodology categories (knowledge poisoning, instruction injection, context manipulation, trigger-based attacks, backdoor injection), 8+ defense strategies (spotlighting/datamarking, instruction hierarchy, dual LLM/CaMeL, perplexity detection, canary tokens, structural framing, blast radius reduction, content sanitization), and applicability analysis for documentation-via-MCP. Key findings: (1) PoisonedRAG achieves 90-99% ASR with just 5 injected texts per question; (2) All 4 defenses tested against PoisonedRAG are insufficient; (3) Adaptive attacks break all 8 tested IPI defenses (>50% ASR); (4) Datamarking is the best server-side defense (ASR from ~50% to <3%); (5) Anthropic's model-level training achieves ~1% ASR but acknowledges residual risk; (6) Documentation's inherently instructional nature makes it harder to defend than general RAG; (7) MCP architecture amplifies attack success rates by 23-41% vs non-MCP baselines.
- **Next**: P1-T1 (attack taxonomy for documentation content), P1-T4 (server-side sanitization), P1-T5 (defense reference implementations)

## 2026-05-14 — P1-T1: Prompt Injection Attack Taxonomy for Documentation Content
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [OWASP Prompt Injection Overview](https://owasp.org/www-community/attacks/PromptInjection) → `docs/owasp-prompt-injection-overview.md`
  - [OWASP LLM01:2025](https://genai.owasp.org/llmrisk/llm01-prompt-injection/) → `docs/owasp-llm01-2025-prompt-injection.md`
  - [OWASP Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/LLM_Prompt_Injection_Prevention_Cheat_Sheet.html) → `docs/owasp-prompt-injection-prevention-cheatsheet.md`
  - [AI Dev Tools Prompt Injection](https://arxiv.org/html/2603.21642v1) → `docs/arxiv-ai-dev-tools-prompt-injection.md`
  - [MCP Threat Modeling](https://arxiv.org/html/2603.22489v1) → `docs/arxiv-mcp-threat-modeling-tool-poisoning.md`
  - [Hidden Comment Injection](https://arxiv.org/html/2602.10498v1) → `docs/arxiv-hidden-comment-injection-llm-agents.md`
  - [Sleeper Cell Temporal Backdoors](https://arxiv.org/html/2603.03371) → `docs/arxiv-sleeper-cell-temporal-backdoors.md`
  - [Promptfoo Invisible Unicode](https://www.promptfoo.dev/blog/invisible-unicode-threats/) → `docs/promptfoo-invisible-unicode-threats.md`
  - [Cycode Unicode Attacks](https://cycode.com/blog/invisible-code-hidden-prompts-unicode-attacks-sast/) → `docs/cycode-invisible-unicode-attacks-repos.md`
  - [Cisco Unicode Tag Injection](https://blogs.cisco.com/ai/understanding-and-mitigating-unicode-tag-prompt-injection) → `docs/cisco-unicode-tag-prompt-injection.md`
  - [AWS Unicode Smuggling](https://aws.amazon.com/blogs/security/defending-llm-applications-against-unicode-character-smuggling/) → `docs/aws-unicode-character-smuggling-defense.md`
  - [Microsoft MCP Indirect Injection](https://developer.microsoft.com/blog/protecting-against-indirect-injection-attacks-mcp) → `docs/microsoft-indirect-injection-mcp-defense.md`
  - [Lakera Indirect Prompt Injection](https://www.lakera.ai/blog/indirect-prompt-injection) → `docs/lakera-indirect-prompt-injection.md`
  - [HackerOne Bard Exfiltration](https://www.hackerone.com/blog/how-prompt-injection-vulnerability-led-data-exfiltration) → `docs/hackerone-prompt-injection-data-exfiltration.md`
  - [Unit 42 MCP Sampling](https://unit42.paloaltonetworks.com/model-context-protocol-attack-vectors/) → `docs/unit42-mcp-sampling-attack-vectors.md`
  - [CopyPasta AI Virus](https://www.hiddenlayer.com/research/prompts-gone-viral-practical-code-assistant-ai-viruses) → `docs/hiddenlayer-copypasta-ai-virus.md`
  - [Copilot CVE-2025-53773](https://embracethered.com/blog/posts/2025/github-copilot-remote-code-execution-via-prompt-injection/) → `docs/embracethered-copilot-rce-cve-2025-53773.md`
  - [NVIDIA AGENTS.md Injection](https://developer.nvidia.com/blog/mitigating-indirect-agents-md-injection-attacks-in-agentic-environments/) → `docs/nvidia-agents-md-injection-attacks.md`
  - [Pillar Rules File Backdoor](https://www.pillar.security/blog/new-vulnerability-in-github-copilot-and-cursor-how-hackers-can-weaponize-code-agents) → `docs/pillar-rules-file-backdoor-copilot-cursor.md`
  - [StackOne MCP Defense](https://www.stackone.com/blog/indirect-prompt-injection-mcp-tools-defense/) → `docs/stackone-indirect-injection-mcp-defense.md`
- **Summary**: Comprehensive attack taxonomy produced in `injection-attack-taxonomy-research.md`. 7 major categories with 17+ specific vectors:
  1. Direct instruction injection (explicit overrides, authority impersonation, role-play framing)
  2. Invisible/hidden text (Unicode tags U+E0000, zero-width binary, variation selectors, PUA, bidi controls, homoglyphs)
  3. Markdown/formatting exploits (HTML comments, hidden CSS/HTML, image URL exfiltration, payload splitting)
  4. Indirect via configuration/code (malicious code examples, insecure config defaults, dependency injection, shell commands)
  5. Multi-step/delayed patterns (conversation poisoning, conditional triggers, multi-document chains, self-propagating worms)
  6. Data exfiltration (image URL encoding, code example exfiltration, covert tool invocation)
  7. Obfuscation/evasion (encoding, typoglycemia, multilingual, surrogate recombination)
  Each vector includes mechanism, documentation location, severity, detectability, and real-world examples. 4 CVEs documented. Severity matrix and MCP-specific priority ranking provided.
- **Next**: P1-T4 (server-side sanitization), P1-T5 (defense reference implementations), then P2 synthesis

## 2026-05-14 — P1-T4: Server-Side Content Sanitization Approaches
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: 16 docs saved with `t4-` prefix (AWS, Cisco, Microsoft, OWASP, academic papers)
- **Summary**: Comprehensive investigation of server-side sanitization for MCP documentation. NFKC is the correct Unicode normalization form (NFC insufficient). Tag characters U+E0000-U+E007F are most dangerous invisible vector — Claude Opus 4 reaches 100% compliance on tag-encoded payloads. Strict delimiter templates achieve 96.3% defense; datamarking reduces ASR from ~50% to ~1-3% with no task degradation. Code blocks need special handling (indentation-sensitive languages break with whitespace replacement). Combined server-side defenses estimate 2-5% ASR against non-adaptive, 5-10% against adaptive attacks. Core tradeoff: documentation IS instructional content — you can transform representation without altering semantic content, but cannot strip instructions without destroying the documentation.
- **Next**: Phase 2 synthesis

## 2026-05-14 — P1-T5: Content-Aware Defense Patterns & Reference Implementations
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: 16 docs saved with `t5-` prefix (Lakera, NeMo, LLM Guard, CaMeL, StruQ, MCP security)
- **Summary**: Surveyed defense landscape — 4 commercial products, 7 open-source libraries, 5 architectural patterns, 6 content filtering approaches, MCP-specific defenses. Key finding: datamarking is single highest-impact server-side defense (ASR ~50% → <3%). All ML classifiers suffer documentation false-positive problems (instructional text triggers injection detectors). Architectural defenses (CaMeL, StruQ, SecAlign) require model-provider changes except tool description framing (free today). Tier 1 recommendations: datamarking, tool description trust framing, Unicode sanitization, content provenance metadata.
- **Next**: Phase 2 synthesis — all P1 tasks complete

## 2026-05-14 — P2-T1: Defense Architecture Synthesis
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Synthesized all 5 P1 reports into a layered defense architecture (`defense-architecture-research.md`). Five defense layers mapped: L1 content sanitization (eliminates invisible injection, ASR ~25%→~13%), L2 structural framing + datamarking (→~2-5%), L3 model-level (→~1%), L4 blast radius reduction (constrains damage), L5 human-in-the-loop (catches residual). Produced attack→defense coverage matrix for 17 vector types. Prioritized 13 implementation actions across 4 tiers. Identified semantic misdirection as the unsolvable residual — a governance problem, not a filtering problem.
- **Next**: P2-T2 executive summary

## 2026-05-14 — P2-T2: Executive Summary Written
- **Type**: analysis
- **Status**: success
- **Depth**: moderate
- **Summary**: Updated research.md with complete executive summary: 6 key findings, defense architecture summary table, implementation priorities (4 immediate, 3 soon, 3 upstream), residual risk assessment, 3 open questions, conclusions linking back to parent spike. All 6 detailed reports cross-linked.
- **Next**: Spike ready for completion review

## 2026-05-14 — Depth Checklist Gaps Filled
- **Type**: analysis
- **Status**: success
- **Depth**: moderate
- **Summary**: Depth checklist audit found 2 gaps: (1) claude-code-mcp-defenses missing cross-product comparison — added Section 7 comparing Claude Code vs Copilot vs Cursor vs Windsurf across 7 dimensions with 4 new sources; (2) defense-architecture missing rejected-alternatives rationale — added section explaining why dual-LLM, CaMeL, StruQ/SecAlign, and gateway approaches were not recommended.
- **Next**: Spike completion

## 2026-05-14 — Spike Completed
- **Type**: decision
- **Status**: success
- **Depth**: deep
- **Summary**: Spike finalized after all 7 tasks completed successfully and all 6 reports passed depth checklist review (2 gaps identified and filled). Key conclusion: layered defense reduces prompt injection ASR from ~25% (undefended) to ~1-2% (all layers), but semantically coherent misdirection in documentation is unsolvable by technical means. Implementation path: trivial sanitization/framing first, datamarking second, upstream advocacy third. 78+ source documents, 6 detailed reports, 1 defense architecture synthesis. No follow-on spikes proposed — remaining open questions are implementation-phase concerns, not research gaps.
