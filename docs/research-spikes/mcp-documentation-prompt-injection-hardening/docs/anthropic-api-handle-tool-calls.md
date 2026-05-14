# Anthropic API: Handle Tool Calls

- **Source URL**: https://platform.claude.com/docs/en/agents-and-tools/tool-use/handle-tool-calls
- **Retrieved**: 2026-05-14

## Tool Result Message Structure

Tool results are sent as content blocks within `user` role messages:

```json
{
  "role": "user",
  "content": [
    {
      "type": "tool_result",
      "tool_use_id": "toolu_01A09q90qw90lq917835lq9",
      "content": "15 degrees"
    }
  ]
}
```

Content can be:
- A plain string: `"content": "15 degrees"`
- Nested content blocks: `"content": [{"type": "text", "text": "15 degrees"}]`
- Document blocks: `"content": [{"type": "document", "source": {"type": "text", "media_type": "text/plain", "data": "15 degrees"}}]`
- Images: base64-encoded image content blocks
- Omitted entirely for empty results

## Formatting Requirements

- Tool result blocks MUST immediately follow corresponding tool_use blocks
- In user messages, tool_result blocks must come FIRST before any text
- Text after tool results is allowed; text before causes 400 error

## Error Handling

Tool errors are returned with `is_error: true`:
```json
{
  "type": "tool_result",
  "tool_use_id": "toolu_01A09q90qw90lq917835lq9",
  "content": "ConnectionError: the weather service API is not available (HTTP 500)",
  "is_error": true
}
```

## Security-Relevant Observations

1. Tool results are embedded in `user` role messages, not a separate `tool` role
2. No documented content sanitization or escaping requirements for tool_result content
3. No documented framing/tagging that marks tool results as "untrusted" vs user text
4. The content field accepts arbitrary strings - no structural separation between data and instructions
5. No mention of trust hierarchy differences between tool_result content and regular user text
