# Sense internal/blast/engine.go
- **Source**: https://raw.githubusercontent.com/luuuc/sense/main/internal/blast/engine.go
- **Retrieved**: 2026-05-15
- **Note**: WebFetch returned a summary rather than verbatim code.

---

## Core Functionality
Computes which symbols would be affected if a subject symbol changed, using reverse-direction breadth-first search on structural edges (calls, inherits, includes, composes, temporal, tests) with confidence decay as the primary depth control.

## Key Components

### Options & Configuration
- `defaultMaxHops = 3` for traversal depth
- `defaultMinConfidence = 0.5` as the cumulative path confidence threshold
- `MaxFrontierWidth = 500` to cap BFS frontier per hop

### Risk Classification
Three tiers based on direct caller count:
- "high": >=10 direct callers
- "medium": >=3 direct callers or temporal coupling present
- "low": otherwise

### Main Function
`Compute()` executes the BFS algorithm, managing visited nodes, confidence paths, and producing a comprehensive `Result` containing affected symbols categorized by relationship type (subclasses, composition, includes).

### Supporting Functions
- `expandFrontier()`: Executes BFS hop queries with chunking for SQLite variable limits
- `classifyTier()`: Maps edge kinds to relevance tiers
- `kindDecay()`: Returns confidence multipliers per edge type
- Data loading functions for symbols, children, and tests

### Implementation Details
- Handles Ruby class reopenings by accepting multiple seed symbol IDs
- Provides result truncation when exceeding `MaxResults`
- Uses structural edges only (calls, inherits, includes, composes, temporal, tests)
- Confidence decays multiplicatively per hop based on edge type
