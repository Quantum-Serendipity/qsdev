# Zep/Graphiti: Temporal Knowledge Graph for Agent Memory

- **Source URLs**:
  - https://arxiv.org/abs/2501.13956 (Zep paper)
  - https://neo4j.com/blog/developer/graphiti-knowledge-graph-memory/
  - https://github.com/getzep/graphiti
- **Retrieved**: 2026-03-15
- **Note**: Content synthesized from web search results.

## Overview

Zep addresses fundamental limitations in RAG frameworks through Graphiti — a temporally-aware knowledge graph engine that dynamically synthesizes both unstructured conversational data and structured business data while maintaining historical relationships.

## Architecture

A context graph is a temporal graph of entities, relationships, and facts with validity windows — indicating when a fact became true and when (if ever) it was superseded.

### Bi-Temporal Model
Tracks two time dimensions:
1. When an event occurred (event time)
2. When it was ingested (ingestion time)

Every graph edge includes explicit validity intervals.

### Conflict Resolution
When conflicts arise, Graphiti uses temporal metadata to update or invalidate (but not discard) outdated information, preserving historical accuracy without large-scale recomputation.

### Incremental Updates
Real-time updates through temporally aware processing — engineers no longer need to recompute entire graphs when data changes. Graphiti incrementally integrates updates and resolves conflicts based on temporal metadata.

## Performance Benchmarks

- DMR benchmark: Zep 94.8% vs baseline 93.4%
- LongMemEval benchmark: Up to 18.5% accuracy improvement
- Response latency: 90% reduction compared to baseline implementations

## Key Insight

Unlike static RAG, which requires batch recomputation when knowledge changes, a temporal knowledge graph can incrementally update in real-time. This is critical for agents operating in dynamic environments where facts change over time (e.g., codebase state, project status, user preferences).
