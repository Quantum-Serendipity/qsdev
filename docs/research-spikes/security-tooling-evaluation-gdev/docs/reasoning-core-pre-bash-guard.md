# reasoning-core pre_bash_guard.py

- **Source**: https://github.com/jakubkrzysztofsikora/reasoning-core/blob/main/src/hooks/pre_bash_guard.py
- **Retrieved**: 2026-05-15
- **Note**: Content returned via WebFetch AI summary — may not be verbatim

---

## Purpose
Blocks shell commands that would rewrite source files inside the project, kill the sidecar process, or modify guard configuration.

## Exit Codes
- 0: command allowed
- 2: command blocked
- 0: also returned on malformed input

## Design Philosophy
"Conservative: when in doubt, allow — this hook is precision, not recall."

## Major Data Structures

- `GUARDED_PATH_FRAGMENTS`: Files that cannot be modified (settings.json, hook scripts, core modules)
- `GUARDED_PROCESS_TOKENS`: Process names protected from termination
- `HARD_DENY_PATTERNS`: Regex patterns for absolute rejections (18+ patterns covering bypass attempts)
- `SRC_WRITE_PATTERNS`: Detects shell redirections/writes targeting source files
- `SAFE_LEADING_TOKENS`: Commands allowed without deeper inspection (git, pytest, npm, etc.)

## Core Functions

- `_read_payload()`: Parses stdin input from the Claude adapter
- `_command_first_token()`: Extracts the base command name
- `_hard_deny_reason()`: Checks against absolute-block patterns
- `_src_write_match()`: Detects source file write attempts
- `_manifest_disallowed_extension()`: Enforces language-family constraints
- `screen_command()`: Main screening logic implementing layered defense (L3, A, B, C, D)
- `main()`: Entry point that logs decisions to audit_log

The implementation uses only stdlib imports and aims for execution under 5 seconds.
