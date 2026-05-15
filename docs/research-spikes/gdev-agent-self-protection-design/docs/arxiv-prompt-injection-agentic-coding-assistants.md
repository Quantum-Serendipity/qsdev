<!-- Source: https://arxiv.org/html/2601.17548v1 -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: Content returned via WebFetch AI summary -->

# Prompt Injection Attacks on Agentic Coding Assistants: A Systematic Analysis

## Three-Dimensional Attack Taxonomy

The paper establishes a systematic classification framework across three orthogonal dimensions:

### Dimension 1: Delivery Vectors

**Direct Prompt Injection (D1)** involves explicit malicious instructions through primary input channels, including role hijacking and context overrides.

**Indirect Prompt Injection (D2)** embeds malicious content in external sources like repository files, documentation, and web content that agents process.

**Protocol-Level Attacks (D3)** exploit communication protocols, particularly MCP tool poisoning and transport-layer vulnerabilities.

### Dimension 2: Attack Modality

**Text-Based (M1)** uses natural language injection with encoding obfuscation techniques.

**Semantic (M2)** exploits code understanding through context poisoning and implicit instructions.

**Multimodal (M3)** leverages non-textual vectors like images and audio.

### Dimension 3: Propagation Behavior

**Single-Shot (P1)** attacks complete in isolated interactions. **Persistent (P2)** establish ongoing access through configuration modification. **Viral (P3)** self-propagate through repositories and dependency chains.

## Empirical Findings

The research synthesizes 78 studies, revealing that "attack success rates against state-of-the-art defenses exceed 85% when adaptive attack strategies are employed." MCPSecBench evaluation found that 85%+ of identified attacks compromise major platforms.

## Defense Mechanism Analysis

Critical evaluation of 18 mechanisms demonstrates fundamental limitations: "All evaluated defenses could be bypassed with attack success rates exceeding 78% using adaptive optimization." Detection-based approaches prove insufficient against obfuscated payloads.

## Recommended Defense Framework

The paper proposes defense-in-depth combining:
- Cryptographic tool identity with immutable versioning
- Fine-grained capability scoping following least-privilege principles
- Multi-agent validation pipelines
- Mandatory sandboxing with strict egress controls
- End-to-end provenance tracking
- Human-in-the-loop gates for high-impact actions

The fundamental challenge remains architectural: "LLMs cannot reliably distinguish between instructions and data," making prompt injection a persistent vulnerability class requiring sustained security investment.
