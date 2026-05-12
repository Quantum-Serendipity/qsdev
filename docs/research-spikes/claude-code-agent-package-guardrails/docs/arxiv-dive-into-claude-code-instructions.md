<!-- Source: https://arxiv.org/html/2604.14228v1 -->
<!-- Retrieved: 2026-05-12 -->
<!-- Note: Extracted sections relevant to CLAUDE.md instruction architecture and compliance -->

# Dive into Claude Code: Instruction Architecture and Compliance (Excerpts)

## CLAUDE.md Hierarchy

"The CLAUDE.md + memory subsystem provides a four-level instruction hierarchy (claudemd.ts) from managed settings to directory-specific files"

## Lazy Loading Strategy

"The base CLAUDE.md hierarchy is loaded at session start, but additional nested-directory instruction files and conditional rules are loaded only when the agent reads files in those directories, preventing unused instructions from consuming context."

## Advisory Nature — Values Over Rules

The design principle "Values over rules" suggests instructions function as guidance rather than enforcement:

"Rigid decision procedures, or contextual judgment backed by deterministic guardrails?"

The architecture emphasizes that the model exercises judgment informed by instructions, not mechanical rule-following.

## Prompt Injection Risks

"The auto-mode threat model explicitly targets four risk categories: overeager behavior, honest mistakes, prompt injection, and model misalignment."

## Context Window Effects on Compliance

"In Claude Code, the context window (200K for older models, 1m for the Claude 4.6 series) is the binding resource constraint."

Instructions compete for space with conversation history, tool outputs, and file contents — implying that very large projects or long conversations may cause instructions to be evicted through compaction mechanisms.

## Missing Coverage in the Analysis

The paper does not comprehensively address:
- Empirical reliability metrics for instruction following
- Failure modes when instructions conflict with conversation context
- How the compaction pipeline handles evicted instructions
- Testing or validation of instruction adherence
