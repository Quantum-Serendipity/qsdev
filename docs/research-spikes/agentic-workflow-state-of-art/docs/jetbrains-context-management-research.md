# JetBrains Junie: Efficient Context Management Research

- **Source URL**: https://blog.jetbrains.com/research/2025/12/efficient-context-management/
- **Retrieved**: 2026-03-15
- **Note**: Content synthesized from web search results; not full page capture.

## Overview

JetBrains published research on efficient context management for LLM-powered coding agents (Junie), comparing three main approaches to compaction.

## Three Context Management Approaches

### 1. LLM Summarization
An LLM rewrites conversation history into natural language summaries.
- High compression ratio
- Human-readable output
- Lossy — details may be dropped
- Higher compute cost (requires LLM inference)

### 2. Observation Masking
When a tool call becomes stale, replace the output with a placeholder. The tool call itself stays visible so the agent remembers what it did.
- On SWE-bench: matched LLM summarization quality while using less compute
- Preserves the reasoning trace
- Key insight: the reasoning trace matters more than the raw data the tools returned

### 3. Verbatim Compaction
Keep surviving lines unchanged from the original (e.g., Morph Compact claims 98% verbatim accuracy).
- Deterministic — no LLM needed
- Preserves exact content for what survives
- Less flexible compression

## Key Insight

Aggressive compression of tool outputs is safe because the reasoning trace matters more than the raw data the tools returned. An agent typically needs to know "I searched for X and found it in file Y" rather than needing the full 500-line file content in context.
