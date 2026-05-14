# Anthropic API: Tool Use with Claude - Overview

- **Source URL**: https://platform.claude.com/docs/en/agents-and-tools/tool-use/overview
- **Retrieved**: 2026-05-14

## How Tool Use Works

Tools differ by where code executes:
- **Client tools** (user-defined + Anthropic-schema like bash, text_editor): Claude responds with `stop_reason: "tool_use"` and `tool_use` blocks. Your code executes and sends back `tool_result`.
- **Server tools** (web_search, code_execution, web_fetch, tool_search): Run on Anthropic infrastructure; results returned directly.

## Tool Result Structure

Tool results are sent as `tool_result` content blocks within a `user` role message:
- `tool_use_id`: Links to the original tool_use request
- `content`: String, list of content blocks (text, image, document), or omitted
- `is_error`: Optional boolean for error signaling

**Critical structural requirement**: Tool result blocks must immediately follow their corresponding tool_use blocks. In user messages containing tool results, tool_result blocks must come FIRST in content array before any text.

## Key Architectural Detail

"Unlike APIs that separate tool use or use special roles like `tool` or `function`, the Claude API integrates tools directly into the `user` and `assistant` message structure. Messages contain arrays of text, image, tool_use, and tool_result blocks. `user` messages include client content and `tool_result`, while `assistant` messages contain AI-generated content and `tool_use`."

This means tool results are embedded within the `user` message role, not in a separate `tool` role.

## Pricing Note

When tools are used, Anthropic automatically includes "a special system prompt for the model which enables tool use" (346 tokens for current models).
