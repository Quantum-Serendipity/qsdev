<!-- Source: https://arxiv.org/html/2404.13208v1 -->
<!-- Retrieved: 2026-05-14 -->

# The Instruction Hierarchy: Training LLMs to Prioritize Privileged Instructions

## Core Problem

Modern LLMs treat all input text equally, creating vulnerabilities where adversaries can override system instructions. The paper identifies this as "the lack of instruction privileges in LLMs."

## The Hierarchy Framework

The proposed solution establishes priority levels:
- **Priority 0 (Critical)**: System messages from developers
- **Priority 10 (High)**: User messages
- **Priority 30 (Low)**: Tool outputs and third-party content

When instructions conflict, models should ignore lower-priority directives or refuse compliance when necessary.

## Training Methodology

The approach uses two complementary principles:

**Context Synthesis** (Aligned Instructions): Models learn to follow compatible lower-level instructions. For example, decomposing "write a 20-line Spanish poem" into separate constraints and training the model to integrate them appropriately.

**Context Ignorance** (Misaligned Instructions): Models train to produce identical outputs as if lower-level instructions were invisible, using red-teamer LLMs to generate adversarial examples across attack types.

## Defense Against Tool-Based Attacks

For indirect prompt injections through tool outputs, the researchers applied "context ignorance" after injecting instructions into simulated web search results. Models learned to "predict the original ground-truth answer as if the adversarial string was not present," effectively treating tool content as data rather than directives.

## Evaluation Results

Testing showed substantial improvements:
- System prompt extraction defense: 63% improvement
- Jailbreak robustness: 30%+ improvement
- Generalization to unseen attack types: Demonstrated across multiple vectors

## Acknowledged Limitations

The approach exhibits "over-refusal" in some cases — models occasionally refuse benign requests resembling attacks. The researchers note this represents a tradeoff requiring further data collection refinement rather than fundamental incompatibility with normal instruction-following.
