# mcph Credential Handling
- **Source**: https://raw.githubusercontent.com/YawLabs/mcph/main/src/credentials.ts
- **Retrieved**: 2026-05-14

## Heuristic Detection for Missing Credentials

Scans stderr output for specific patterns when a server fails to start:
- "missing [env] variable [NAME]"
- "[NAME] is required/not set/missing/empty/undefined"
- "[NAME] must be set"
- "please set [env] variable [NAME]"

## Filtering Rules

- Only ALL_CAPS identifiers (minimum 3 characters) treated as credentials
- Blocklist of common system env vars (PATH, HOME, SHELL, NODE_ENV, etc.) excluded
- Prevents ordinary English words from triggering false positives

## Workflow

When a local upstream fails:
1. Scans error message against patterns
2. Validates captured names meet ALL_CAPS format
3. Filters against ignored set
4. Returns detected credential names for MCP elicitation (prompts user)
