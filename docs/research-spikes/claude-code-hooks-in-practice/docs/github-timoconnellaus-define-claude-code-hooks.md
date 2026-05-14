# define-claude-code-hooks - timoconnellaus/define-claude-code-hooks
- **Source**: https://github.com/timoconnellaus/define-claude-code-hooks
- **Retrieved**: 2026-03-27

## Overview
Type-safe hook definitions for Claude Code with automatic settings management. Enables TypeScript-based hook definitions in `.claude/hooks/` that automatically compile into settings files.

## Core Purpose
- TypeScript-based hook definitions with full type safety
- Automatic `.claude/settings.json` management
- Support for both project-wide and local-only configurations
- Pre-built utilities for common logging/control scenarios

## Hook File Organization
- `hooks.ts` → updates `.claude/settings.json` (project-level)
- `hooks.local.ts` → updates `.claude/settings.local.json` (local-only)

## Hook Types Supported
- PreToolUse (execute before tool runs, can block)
- PostToolUse (execute after tool completes)
- Notification (handle Claude messages)
- Stop (main agent completion)
- SubagentStop (subagent completion)

## Quick Installation
```bash
npx @timoaus/define-claude-code-hooks --init
```

Interactive command that selects project vs. local hooks, installs predefined hooks, adds package as dev dependency, creates `claude:hooks` npm script.

## API

### defineHooks
```typescript
export default defineHooks({
  PreToolUse: [
    {
      matcher: "Bash",
      handler: async (input) => {
        if (input.tool_input.command?.includes("grep")) {
          return {
            decision: "block",
            reason: "Use ripgrep (rg) instead of grep",
          };
        }
      },
    },
  ],
  Stop: [async (input) => { /* ... */ }],
});
```

### defineHook
```typescript
const preventEditingEnvFile = defineHook("PreToolUse", {
  matcher: "Write|Edit|MultiEdit",
  handler: async (input) => {
    const filePath = input.tool_input.file_path;
    if (filePath && filePath.endsWith(".env")) {
      return {
        decision: "block",
        reason: "Direct editing of .env files is not allowed for security reasons",
      };
    }
  },
});
```

## Predefined Hook Utilities

| Function | Purpose |
|----------|---------|
| `logPreToolUseEvents` | Log tools before execution |
| `logPostToolUseEvents` | Log tools after execution |
| `logStopEvents` | Log main agent completion |
| `logSubagentStopEvents` | Log subagent completion |
| `logNotificationEvents` | Log messages |
| `blockEnvFiles` | Prevent .env access |
| `announceStop` | TTS task completion |
| `announcePreToolUse` | TTS before tool use |
| `announcePostToolUse` | TTS after tool use |

## Hook Return Values
```typescript
interface HookOutput {
  continue?: boolean;
  stopReason?: string;
  suppressOutput?: boolean;
  decision?: "approve" | "block";  // PreToolUse specific
  reason?: string;
}
```

## Key Benefits
- Type Safety with autocomplete
- Automatic log rotation
- Error resilience (graceful handling without interrupting Claude)
- Regex-based tool matching
- Dual Configuration (project + local)
