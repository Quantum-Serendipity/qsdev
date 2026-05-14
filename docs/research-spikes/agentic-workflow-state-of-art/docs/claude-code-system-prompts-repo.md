# Claude Code System Prompts Repository
- **Source**: https://github.com/Piebald-AI/claude-code-system-prompts
- **Retrieved**: 2026-03-15
- **Type**: Community reverse-engineering / open source repository

## Overview
Documents Claude Code's complete system prompt architecture as of version 2.1.76 (March 13, 2026). Provides a catalog of over 110 distinct prompt strings including conditional portions, tool descriptions, and utility functions.

## Key Architecture Insights

### Multiple System Prompts
Claude Code doesn't use a single monolithic system prompt. Instead employs:
- Conditional components added based on environment and configuration
- Tool descriptions for built-in utilities (Write, Bash, TodoWrite, etc.)
- Separate agent prompts for Explore and Plan sub-agents
- Utility function prompts for conversation compaction, CLAUDE.md generation, session titling

### Prompt Categories

**Agent Prompts (Sub-agents & Utilities)**:
- Explore (517 tokens) and Plan mode (685 tokens)
- Agent creation (1,110 tokens)
- CLAUDE.md generation (384 tokens)
- Status line setup (1,641 tokens)
- /batch (1,136 tokens), /security-review (2,607 tokens)
- 25+ specialized utility agents

**Data Prompts**:
- Agent SDK patterns and references (Python/TypeScript)
- Claude API references across 10+ languages
- GitHub Actions workflows, HTTP error codes

**System Prompt Components (50+)**:
- Output efficiency and tone/style guidelines
- Tool usage policies (prefer native tools over bash)
- Task execution philosophy (avoid over-engineering, minimize file creation)
- Auto mode and learning mode instructions
- Memory system and session continuation
- Security monitoring and malware analysis protocols

**System Reminders (~40)**:
- File modification notifications
- Hook system messages
- Plan mode activation variants
- Memory file references
- Token usage and USD budget tracking
- Diagnostic and linter output notifications

## Notable Details
- System reminders injected after tool calls to keep agent on track
- TodoWrite tool for structured task tracking
- Token counts exclude interpolated variables (±20 tokens typical variance)
- Updated within minutes of each Claude Code release
- Repository tracks changes across 126+ versions

## Supporting Tool: tweakcc
Enables customization of individual prompt sections as markdown files with patching capability.
