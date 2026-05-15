# Sense internal/hook/pre_tool_use.go
- **Source**: https://raw.githubusercontent.com/luuuc/sense/main/internal/hook/pre_tool_use.go
- **Retrieved**: 2026-05-15
- **Note**: WebFetch returned a summary rather than verbatim code.

---

## Key Components

**Data Structures:**
- `preToolUseInput` struct unmarshals JSON tool requests with fields for tool name, pattern, command, regex, subagent type, prompt, and description.

**Global Variables:**
- Maps and slices defining exploration agents ("deep-explore", "Explore") and phrases indicating codebase analysis intent
- List of supported code file extensions (.go, .py, .ts, .js, .rb, .rs, .java, .kt, .scala, .cs, .php)

**Main Handler Functions:**
- `handlePreToolUse()` dispatches to specific handlers based on tool type (Agent, Bash, Grep, Glob)
- `handleAgent()` detects exploration queries and suggests Sense MCP tools
- `handleBash()`, `handleGrep()`, `handleGlob()` detect symbol-matching patterns and redirect to specialized Sense tools
- Functions return nudge notifications recommending appropriate Sense tools

**Utility Functions:**
- `extractPattern()` and `extractBashPattern()` parse tool inputs to identify search patterns
- `isSymbolShaped()` validates whether strings appear to be code symbols (rejects regex characters, paths, file extensions)
- `hasExplorationKeyword()` performs case-insensitive phrase matching
- `isExplorationCommand()` identifies file-reading commands (find, cat, head, etc.)
- `needsValue()` determines if CLI flags require arguments

## Mechanism
The system functions as an intelligent interceptor: when Claude Code is about to use grep/glob/bash to search for code symbols, the pre-tool-use hook detects this and nudges the AI to use Sense's indexed tools instead. This reduces token waste from manual searching.
