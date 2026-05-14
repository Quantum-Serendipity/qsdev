# Mem0: Universal Memory Layer for AI Agents

- **Source URLs**:
  - https://mem0.ai/
  - https://arxiv.org/abs/2504.19413 (Mem0 paper)
  - https://mem0.ai/blog/ai-memory-layer-guide
  - https://mem0.ai/blog/graph-memory-solutions-ai-agents
- **Retrieved**: 2026-03-15
- **Note**: Content synthesized from web search results.

## Overview

Mem0 is a scalable memory-centric architecture that dynamically extracts, consolidates, and retrieves salient information from ongoing conversations. It sits between your application and the LLM, automatically extracting relevant information from conversations, storing it, and retrieving it when needed.

## Memory Architecture

Three memory types:
1. **Episodic memory**: Stores interaction-specific events
2. **Semantic memory**: Holds extracted knowledge without event context
3. **Procedural memory**: Encodes behavioral patterns

## Multi-Session Learning

When agents have a memory layer, they persist data across sessions:
- Store user preferences, past exchanges, and learned context in a retrieval layer
- When user returns, agent accesses stored information and picks up where conversation ended
- User memory persists across all conversations with a specific person

## Performance (2025 Benchmarks)

On the LOCOMO benchmark:
- 26% higher accuracy than OpenAI's built-in memory feature
- 91% faster response by selectively retrieving relevant memories
- ~90% token usage reduction compared to full-context approaches

## Graph Memory Enhancement (2025-2026)

Enhanced variant leverages graph-based memory representations to capture complex relational structures among conversational elements. Combines vector-based retrieval with graph structure for better relationship tracking.

## Key Insight

Mem0 demonstrates that selective memory retrieval significantly outperforms full-context approaches in both speed and accuracy. The key is extracting and storing the right information, not dumping everything into context.
