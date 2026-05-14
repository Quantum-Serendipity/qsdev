# Effective Context Engineering for AI Agents (Anthropic, 2025)

- **Source URLs**:
  - https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents
  - https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents
  - https://code.claude.com/docs/en/how-claude-code-works
  - https://engineering.atspotify.com/2025/11/context-engineering-background-coding-agents-part-2
- **Retrieved**: 2026-03-15
- **Note**: Compiled from web search results on Anthropic's context engineering and agentic prompting guidance.

---

## Context Engineering Definition
"The delicate art and science of filling the context window with just the right information for the next step." A shift from prompt engineering (clever wordsmithing) to context engineering (rigorous software architecture).

## Three Core Patterns

### 1. Server-Side Compaction
Summarizes earlier parts of conversation, enabling long-running conversations beyond context limits. Available in beta for Claude Opus 4.6 and Sonnet 4.6.

### 2. Structured Note-Taking (Agentic Memory)
Agent regularly writes notes persisted outside the context window (to files). Notes get pulled back into context at later times, providing persistent memory with minimal overhead.

### 3. Just-in-Time Retrieval
Rather than pre-loading all data, agents maintain lightweight identifiers (file paths, stored queries, web links) and dynamically load data using tools at runtime. Claude Code uses this for complex operations over large codebases without loading everything into context.

## CLAUDE.md as Hybrid Context
Claude Code employs a hybrid model:
- CLAUDE.md files are naively dropped into context up front (always available)
- Primitives like glob and grep allow just-in-time context loading (on demand)

## Prompting for Agentic Workflows
- Describe the end state, leave room for the model to figure out how to get there
- Use initializer pattern: first session sets up environment (init.sh, progress files)
- Subsequent sessions ask for incremental progress with structured updates
- Claude Code works through: gather context -> take action -> verify results

## Key Insight
Context engineering is about curating the smallest high-signal set of tokens the model sees at each step — improving accuracy, avoiding context rot, and enabling reliable multi-turn behavior.
