# Sense internal/hook/subagent_start.go
- **Source**: https://raw.githubusercontent.com/luuuc/sense/main/internal/hook/subagent_start.go
- **Retrieved**: 2026-05-15
- **Note**: WebFetch returned partial content.

---

## Function: handleSubagentStart

1. Queries symbol count from `sense_symbols` table, returns early if zero or error
2. Queries edge count from `sense_edges` table
3. Constructs informational text with:
   - Index statistics (symbol and edge counts)
   - Instructions to load Sense tools before using grep/find
   - Four tool descriptions: sense_graph, sense_search, sense_blast, sense_conventions
4. Returns a hookResponse struct containing the context string

## Purpose
When Claude Code spawns a sub-agent, this hook injects Sense awareness into the sub-agent's context so it knows to use indexed tools rather than raw file searching.
