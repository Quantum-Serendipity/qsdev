# ZacheryGlass/.claude User-Level Settings
- **Source**: https://github.com/ZacheryGlass/.claude/blob/master/settings.json
- **Retrieved**: 2026-03-27

## Overview
A real user-level Claude configuration with safety guards, commit validation, and emoji removal.

## Environment Variables
- `DISABLE_TELEMETRY`: "1"
- `CLAUDE_CODE_IDE_SKIP_AUTO_INSTALL`: "1"

## Model & Behavior
- Model: "sonnet"
- Effort Level: "high"
- Auto Updates: "latest" channel
- Skip dangerous mode prompts: enabled

## Permission Framework

### Allowed Operations (non-interactive)
File operations on GPU project directories, web search/fetch, read/write capabilities, and extensive bash commands (ls, grep, git operations, AWS CLI).

### Conditional Actions (require user approval)
Destructive operations (rm, mv, cp), network tools (curl, wget), elevated privileges (sudo), container/cloud tools (docker, kubectl, gcloud, az), and git push/pull.

## Hook Configurations

### PreToolUse Hooks

1. **Bash matcher** — Runs two Python guards:
   - `clean_commit_guard.py` (validates commits)
   - `github_issue_guard.py` (validates GitHub operations)

2. **Edit/Write matcher** — Executes `protect_claude_md.py`

3. **Git commit matcher** — Runs `clean_commit_guard.py`

4. **GitHub API matcher** (create/update issues, add comments) — Runs `github_issue_guard.py`

### PostToolUse Hooks
- **Edit/Write operations** — Runs `emoji_remover.py`

### Status Display
PowerShell script (`statusline.ps1`) provides real-time status information.

## Notable Patterns
- Safety through pre-execution validation
- Protecting CLAUDE.md from modification
- Removing emojis from Claude's output
- Commit message quality enforcement
- GitHub issue quality guard
