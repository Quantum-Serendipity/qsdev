# Anthropic API: How Tool Use Works

- **Source URL**: https://platform.claude.com/docs/en/agents-and-tools/tool-use/how-tool-use-works
- **Retrieved**: 2026-05-14

## The Tool-Use Contract

"Tool use is a contract between your application and the model. You specify what operations are available and what shape their inputs and outputs take; Claude decides when and how to call them. The model never executes anything on its own."

## Where Tools Run

Three buckets:
1. **User-defined tools (client-executed)**: You write schema, execute code, return results
2. **Anthropic-schema tools (client-executed)**: Anthropic publishes schema (bash, text_editor, computer, memory), you execute
3. **Server-executed tools**: Anthropic runs code (web_search, web_fetch, code_execution, tool_search)

## The Agentic Loop

Client tools require application to drive a loop:
1. Send request with tools array and user message
2. Claude responds with stop_reason: "tool_use" and tool_use blocks
3. Execute each tool, format outputs as tool_result blocks
4. Send new request with original messages + assistant response + user message with tool_results
5. Repeat while stop_reason == "tool_use"

## Security-Relevant Observations

- "The model never executes anything on its own" - but the model DOES process tool result content as part of its context
- No documented trust boundary between tool_result content and regular conversation
- No mention of content validation, sanitization, or trust-level differentiation for tool results
- Tool results flow directly into the conversation context alongside user messages and system prompts
- The documentation treats tool results as purely functional data, with no security framing
