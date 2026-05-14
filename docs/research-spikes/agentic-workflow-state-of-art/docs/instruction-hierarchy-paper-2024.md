# The Instruction Hierarchy: Training LLMs to Prioritize Privileged Instructions

- **Source URL**: https://arxiv.org/abs/2404.13208
- **Authors**: Eric Wallace, Kai Xiao, Reimar Leike, Lilian Weng, Johannes Heidecke, Alex Beutel
- **Published**: April 2024 (ICLR 2025)
- **Retrieved**: 2026-03-15
- **Note**: Summary compiled from arxiv abstract and related search results.

---

## Problem Statement
LLMs often treat system prompts with the same priority as text from untrusted users and third parties. They cannot inherently determine which instructions should take precedence, creating a critical vulnerability.

## Proposed Solution: Instruction Hierarchy
- System messages take precedence over user messages
- User messages take precedence over third-party content
- Models trained to selectively ignore lower-privileged instructions when conflicts arise

## Key Results
- Drastically increases robustness even for attack types not seen during training
- Minimal degradation on standard capabilities
- Applied to GPT-3.5 in the paper

## Related 2025 Work
- **Instructional Segment Embedding** (ICLR 2025): Incorporates instruction-type information directly into the model to better distinguish and prioritize instructions based on privilege
- **SecAlign**: Constructs preference dataset with prompt-injected inputs, reducing prompt injection success rates to less than 10%
- Prompt injection ranked as #1 in OWASP Top 10 for LLM Applications 2025

## Practical Implications
- System prompts should contain the highest-priority instructions
- User-provided content should be clearly demarcated from system instructions
- Third-party content (tool outputs, retrieved documents) needs the lowest trust level
