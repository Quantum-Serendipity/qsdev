# attach-guard: Claude Code Plugin for Package Supply Chain Security

- **Source URL**: https://dev.to/hammadtariq/i-built-a-claude-code-plugin-that-blocks-compromised-packages-before-installation-1o3l
- **Retrieved**: 2026-05-12
- **Note**: First known Claude Code plugin specifically targeting package install guardrails.

---

## Core Functionality

The plugin intercepts package installation commands before execution using Claude Code's PreToolUse hooks — a mechanism that "runs automatically on every matching tool call" and cannot be skipped by Claude.

## How It Works

When a user attempts installation (e.g., `npm install axios`), attach-guard:

1. Intercepts the command pre-execution
2. Evaluates the package via Socket.dev's supply chain API
3. Blocks unsafe packages or suggests safer alternatives
4. Supports npm, pip, Go, and Cargo

## Detection Capabilities

- Known malware and compromised packages
- Packages published within 48 hours (age gate)
- Low supply chain scores: below 50 = blocked, 50-70 = flagged

## Real-World Example

For `axios@1.14.1` (scoring 40/100), the tool blocks installation and rewrites the command to `axios@1.14.0` (71/100) using PreToolUse `updatedInput`.

## Installation

```
claude plugin marketplace add attach-dev/attach-guard
claude plugin install attach-guard@attach-dev
```

## Key Technical Details

- Uses PreToolUse hook with Bash matcher
- Pattern-matches package install commands across multiple package managers
- Calls Socket.dev API for real-time supply chain risk scoring
- Returns `updatedInput` to rewrite commands to safer package versions
- Cannot be skipped by the AI agent — fires on every matching tool call
