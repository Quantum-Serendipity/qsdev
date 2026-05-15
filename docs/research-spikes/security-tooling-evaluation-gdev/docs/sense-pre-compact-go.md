# Sense internal/hook/pre_compact.go
- **Source**: https://raw.githubusercontent.com/luuuc/sense/main/internal/hook/pre_compact.go
- **Retrieved**: 2026-05-15
- **Note**: WebFetch returned a summary rather than verbatim code.

---

## Key Components

**handlePreCompact function**: Main handler that generates a summary of the Sense index. It:
- Queries a SQLite database for symbol and edge counts
- Retrieves the top 5 "hub" symbols (most connected entities)
- Returns a formatted message with statistics and instructions

**topHubs function**: Helper that executes a complex SQL query to identify the most interconnected symbols by analyzing both inbound and outbound connections.

**hub struct**: A simple data structure representing a symbol with its inbound and outbound connection counts.

## Mechanism
Before Claude Code compacts its context window, this hook injects a summary of the codebase structure (key hub symbols, graph statistics) so that post-compaction the agent retains structural awareness. The message instructs the agent to "Use Sense MCP tools (sense_graph, sense_search, sense_blast, sense_conventions) for ALL codebase understanding."
