# Claude Code: Connect Claude Code to tools via MCP

- **Source**: https://code.claude.com/docs/en/mcp
- **Retrieved**: 2026-05-14

## Key Findings for Provenance Research

### _meta Annotations

Claude Code recognizes specific vendor-namespaced _meta fields on MCP tool results:

- `anthropic/maxResultSizeChars`: Allows individual MCP tools to return results larger than the default persist-to-disk threshold. Claude Code raises that tool's threshold to the annotated value, up to a hard ceiling of 500,000 characters. Set in the tool's `tools/list` response entry.
- `anthropic/alwaysLoad`: Marks individual tools as always-loaded when set to `true` in the tool's _meta object.

MCP reserves the `_meta` object on requests, responses, and tool results as a vendor-namespaced extension point for implementation-specific metadata that travels with the payload without polluting the protocol schema.

Enforcement is in the harness, not the model — the LLM sees preserved bytes, not the annotation.

### Trust Model

- Claude Code prompts for approval before using project-scoped servers from .mcp.json files
- "Verify you trust each server before connecting it. Servers that fetch external content can expose you to prompt injection risk."
- No per-response trust evaluation — trust is binary at server level (approved or not)
- No metadata-based trust differentiation for tool responses

### Result Handling

- Claude Code displays a warning when MCP tool output exceeds 10,000 tokens
- MAX_MCP_OUTPUT_TOKENS environment variable controls the limit
- Results exceeding the threshold are persisted to disk and replaced with a file reference
- The LLM sees the content directly, not any _meta annotations

### Security

- Server scope hierarchy: Local > Project > User > Plugin > claude.ai connectors
- OAuth 2.0 support for remote server authentication
- Project-scoped servers require explicit user approval
- No content integrity verification on responses
