<!-- Source: https://code.claude.com/docs/en/features-overview -->
<!-- Retrieved: 2026-05-14 -->

# Extend Claude Code - Features Overview

## Extension Layer

Extensions plug into different parts of the agentic loop:
- CLAUDE.md: persistent context every session
- Skills: reusable knowledge and invocable workflows
- MCP: connects to external services and tools
- Subagents: run own loops in isolated context, returning summaries
- Agent teams: coordinate multiple independent sessions
- Hooks: fire on lifecycle events
- Plugins: package and distribute features

## MCP Server Handling

MCP servers override by name with priority: local > project > user.

Tool search is on by default, so idle MCP tools consume minimal context. Full JSON schemas stay deferred until Claude needs a specific tool.

MCP connections can fail silently mid-session. If a server disconnects, its tools disappear without warning.

## How Features Layer

- CLAUDE.md files are additive: all levels contribute content simultaneously
- Skills override by name: managed > user > project
- MCP servers override by name: local > project > user
- Hooks merge: all registered hooks fire for matching events

## Context Costs

| Feature         | When it loads             | What loads                                    | Context cost |
|:----------------|:--------------------------|:----------------------------------------------|:-------------|
| CLAUDE.md       | Session start             | Full content                                  | Every request |
| Skills          | Session start + when used | Descriptions at start, full content when used | Low |
| MCP servers     | Session start             | Tool names; full schemas on demand            | Low until used |
| Subagents       | When spawned              | Fresh context with specified skills           | Isolated |
| Hooks           | On trigger                | Nothing (runs externally)                     | Zero unless returns output |

## Combining Features

| Pattern                | How it works |
|:-----------------------|:-------------|
| Skill + MCP            | MCP provides connection; skill teaches how to use it |
| Skill + Subagent       | Skill spawns subagents for parallel work |
| CLAUDE.md + Skills     | CLAUDE.md holds always-on rules; skills hold reference material |
| Hook + MCP             | Hook triggers external actions through MCP |
