# Claude Code Source Leak: Security Architecture Analysis

- **Source URL**: https://claudefa.st/blog/guide/mechanics/claude-code-source-leak
- **Retrieved**: 2026-05-14
- **Note**: Based on third-party analysis of leaked source; some claims may be inaccurate.

## Tool Processing & Context Structure

- **40+ registered tools** with base tool definition spanning 29,000 lines
- **Query engine alone: 46,000 lines**, suggesting sophisticated tool invocation and result processing

## Security Checkpoints

"bashSecurity.ts contains 23 numbered security checks that gate every shell command Claude Code executes." Multi-layer permission verification, not a single deny mechanism.

## Prompt Caching & Context Management

- **14 tracked cache-break vectors**: system actively monitors conditions that could invalidate prompt cache
- **5 context compaction strategies**: tool results may undergo multiple compression passes

## Distillation Defense

- Anti-distillation system injects "decoy tool definitions into responses" to poison competitor training
- Server returns "only cryptographically signed summaries rather than full reasoning chains"

## Key Gap

The leaked source analysis does NOT detail:
- How tool results are framed/tagged when inserted into context
- Whether MCP tool results receive any special trust markers
- The exact system prompt structure for tool result handling
- Internal trust differentiation between tool_result and user text
