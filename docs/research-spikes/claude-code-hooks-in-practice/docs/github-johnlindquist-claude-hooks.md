# Claude Hooks - johnlindquist/claude-hooks
- **Source**: https://github.com/johnlindquist/claude-hooks
- **Retrieved**: 2026-03-27

## Overview
"Claude-hooks gives you a powerful, TypeScript-based way to customize Claude Code's behavior." It enables developers to write hooks using TypeScript with full type safety and auto-completion for accessing strongly-typed payload data.

## Hook Types & Events

The system supports four primary hook event types:

1. **PreToolUse** - Fires before Claude executes a tool
2. **PostToolUse** - Fires after a tool completes execution
3. **Notification** - Triggered for notification events
4. **Stop** - Fires when a session ends

## Handler Structure

Each hook receives a typed payload and returns appropriate responses. Examples:

**PreToolUse Handler**: `async function preToolUse(payload: PreToolUsePayload): Promise<HookResponse>` - checks tool names and inspects tool inputs before execution.

**PostToolUse Handler**: `async function postToolUse(payload: PostToolUsePayload): Promise<void>` - reacts to completed tool operations, accessing results and success status.

## Key Features

- Full TypeScript type definitions for all payload structures
- Auto-completion support in code editors
- Access to tool names, inputs, and execution results
- Ability to conditionally allow/block operations via return actions
- Support for async/await patterns and npm package integration

## Generated Configuration

The CLI creates:
- `.claude/settings.json` (hook configuration)
- `.claude/hooks/index.ts` (customizable handlers)
- `.claude/hooks/lib.ts` (type definitions)
- `.claude/hooks/session.ts` (optional session utilities)

Session logs save to system temp directory under `claude-hooks-sessions/`
