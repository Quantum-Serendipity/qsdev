# Sense internal/hook/post_tool_use.go
- **Source**: https://raw.githubusercontent.com/luuuc/sense/main/internal/hook/post_tool_use.go
- **Retrieved**: 2026-05-15
- **Note**: WebFetch returned a summary rather than verbatim code.

---

## Purpose
Implements a post-tool-use hook handler that keeps the Sense index fresh as Claude Code edits files.

## Mechanism
The handler:
1. Extracts file paths from tool input using `extractWrittenPath`
2. Validates that paths remain within the working directory
3. Excludes paths in `.sense/` directories
4. Checks if the file extension has an associated extractor
5. Filters paths against ignore patterns
6. Runs an incremental scan on modified files via `scan.RunIncremental`

## Key Details
- `postToolUseTimeout` is set to 4 seconds, leaving headroom within a 5-second external timeout
- Maps tool names ("Write", "Edit", "NotebookEdit") to their respective path fields (`FilePath` or `NotebookPath`)
- Discards output and warnings during scanning
- Embeddings disabled for incremental processing (deferred to MCP server)

## Significance
This is a critical piece of the architecture: after every Write/Edit by Claude Code, the index is incrementally updated so subsequent sense_graph/sense_search queries reflect the latest code state.
