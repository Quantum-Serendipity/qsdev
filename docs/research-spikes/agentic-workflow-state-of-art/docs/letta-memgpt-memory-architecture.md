# Letta/MemGPT Memory Architecture

- **Source URLs**:
  - https://docs.letta.com/concepts/memgpt/
  - https://docs.letta.com/guides/agents/memory/
  - https://docs.letta.com/advanced/memory-management/
  - https://www.letta.com/blog/benchmarking-ai-agent-memory
  - https://www.letta.com/blog/agent-memory
- **Retrieved**: 2026-03-15
- **Note**: Content synthesized from multiple web search results; not full page captures.

## Overview

MemGPT (now Letta) introduced an LLM-as-Operating-System paradigm where the model manages its own memory, context, and reasoning loops — much like a traditional OS manages RAM and disk.

## Memory Hierarchy

### Core Memory (In-Context)
Core memory consists of in-context memory blocks that are always injected into the agent's prompt. These blocks can be managed by the agent itself or by other agents, focusing on specific topics such as memories about the user, organization, or the current task.

Key features:
- Blocks are editable via APIs and remain pinned to the agent's context window
- Provides an abstraction for managed context units
- Default character limit per block: 2,000 characters
- Defined by extending the BaseMemory class with a self.memory dictionary mapping labeled sections (e.g., "human", "persona") to MemoryModule objects

### Archival Memory (External/Long-Term)
A table in a vector DB that stores long-running memories and external data that the agent needs access to but that doesn't fit in the context window.

### Recall Memory (Conversation History)
Preserves the complete history of interactions that can be searched and retrieved when needed. In Letta, recall memory saves to disk automatically.

## Self-Editing Memory Implementation

MemGPT provides flexible memory management by enabling the agent to self-manage memory via tool calls:
- `core_memory_append` — add to a core memory block
- `core_memory_replace` — edit content in a core memory block
- `conversation_search` — search conversation history
- `archival_memory_insert` — store in archival memory
- `archival_memory_search` — retrieve from archival memory

The agent itself decides what information to place into its context at any given time.

## Benchmarking Results

Letta published benchmarks showing their approach is competitive with simpler filesystem-based memory approaches. Letta Code (their memory-first coding agent) achieved #1 model-agnostic open source agent ranking on Terminal-Bench.

## Key Insight
The OS-inspired approach of treating memory as a resource the agent actively manages (rather than something managed for it) enables agents that persist, evolve, and maintain identity across sessions.
