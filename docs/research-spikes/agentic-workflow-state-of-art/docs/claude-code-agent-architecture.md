# Claude Code: Agent Architecture and Multi-Agent Patterns

- **Source URLs**:
  - https://code.claude.com/docs/en/sub-agents
  - https://code.claude.com/docs/en/how-claude-code-works
  - https://www.zenml.io/llmops-database/claude-code-agent-architecture-single-threaded-master-loop-for-autonomous-coding
  - https://blog.promptlayer.com/claude-code-behind-the-scenes-of-the-master-agent-loop/
  - https://platform.claude.com/docs/en/agent-sdk/agent-loop
- **Retrieved**: 2026-03-15
- **Note**: Content compiled from multiple search results.

## Core Philosophy

"A simple, single-threaded master loop combined with disciplined tools and planning delivers controllable autonomy."

Deliberate choice for simplicity and debuggability over complex multi-agent coordination patterns.

## Single-Threaded Master Loop

### Architecture
Internally codenamed "nO". Implements a classic while-loop pattern:
1. Continue execution as long as the model's responses include tool calls
2. When Claude produces plain text without tool invocations, the loop naturally terminates
3. Returns control to the user

### Design Decisions
- Single main thread with one flat message history
- Explicitly avoids threaded conversations or multiple competing agent personas
- Prevents unpredictable behaviors in production environments
- TODO-based planning and diff-based workflows

## Tool Architecture

Tools are what make Claude Code agentic. With tools, Claude can:
- Read code
- Edit files
- Run commands
- Search the web
- Interact with external services

Each tool use returns information that feeds back into the loop, informing Claude's next decision.

### Built-in Tools
Core tools include file read/write/edit, bash execution, web search/fetch, grep, glob (file search), and the Agent/Task tool.

## Sub-Agent Architecture

### dispatch_agent (Task Agent / I2A)
For tasks requiring exploration or alternative approaches, Claude invokes sub-agents through the `dispatch_agent` tool.

### Depth Limitation
**Subagents cannot spawn other subagents** — core architectural constraint preventing recursive explosion/infinite nesting.

### Parallelism
Task tool delegates work to parallel sub-agents, running up to **7 agents simultaneously**. But strictly limited to a single level of depth.

### Sub-Agent Configuration
Each subagent is a Claude instance with:
- Custom system prompt
- Its own context window
- Own expertise and tools (customizable)
- Defined scope of what it knows and when to invoke

## Extension Framework

### Skills (October 2025)
Organized folders of instructions, scripts, and resources. Claude discovers and loads dynamically. Mechanism for packaging repeatable expertise into reusable modules.

### MCP (Model Context Protocol)
How Claude Code plugs external systems into the gather/act/verify loop. Instead of hardcoding integrations, the agent discovers and calls MCP-provided tools on demand.

### Hooks
Run in the application process, not inside the agent's context window (don't consume context). Can short-circuit the loop: a PreToolUse hook that rejects a tool call prevents execution, and Claude receives the rejection message instead.

## Agent Teams (Early 2026)

Anthropic shipped experimental "Swarms" feature — a team lead agent plans and delegates to specialized agents rather than writing code itself. Available behind feature flags.

## Code Review Multi-Agent (March 2026)

Multi-agent system that dispatches AI teams to analyze every pull request. Available in research preview for Team and Enterprise users.

## Strengths
- Simplicity and debuggability
- Single-threaded loop is predictable and transparent
- Hooks system for customization without context consumption
- MCP for extensibility
- Skills for reusable expertise
- Depth-limited sub-agents prevent runaway costs

## Weaknesses
- Single level of sub-agent depth limits complex decomposition
- Sub-agents cannot spawn their own sub-agents (user-requested feature)
- Limited parallel orchestration compared to dedicated frameworks
- No built-in state persistence across sessions (relies on files)
