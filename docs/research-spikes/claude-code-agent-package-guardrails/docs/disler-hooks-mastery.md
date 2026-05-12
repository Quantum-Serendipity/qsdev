# disler/claude-code-hooks-mastery

- **Source URL**: https://github.com/disler/claude-code-hooks-mastery
- **Retrieved**: 2026-05-12
- **Note**: Community repository (3.7k stars) with 13 hook lifecycle examples and security patterns.

---

## Security Implementation: PreToolUse Command Blocking

```python
dangerous_patterns = [
    r'rm\s+.*-[rf]',        # rm -rf variants
    r'sudo\s+rm',           # sudo rm commands
    r'chmod\s+777',         # Dangerous permissions
    r'>\s*/etc/',           # Writing to system directories
]

for pattern in dangerous_patterns:
    if re.search(pattern, command, re.IGNORECASE):
        print(f"BLOCKED: {pattern} detected", file=sys.stderr)
        sys.exit(2)
```

## Architecture

Uses "UV single-file scripts" — each hook script in `.claude/hooks/` declares its own dependencies via inline metadata, keeping hooks isolated from project dependencies.

## Exit Codes

| Code | Behavior | Purpose |
|---|---|---|
| 0 | Success | Hook executed correctly |
| 2 | Blocking Error | stderr fed to Claude; blocks execution |
| Other | Non-blocking Error | stderr shown to user; continues |

## PostToolUse Validation

```python
if tool_name == "Write" and not tool_response.get("success"):
    output = {
        "decision": "block",
        "reason": "File write operation failed, check permissions"
    }
    print(json.dumps(output))
    sys.exit(0)
```

## Audit Trail

All hooks produce JSON logs in `logs/` directory for compliance tracking.

## Key Contribution

Demonstrates the full hook lifecycle with practical security patterns. Shows that hooks can be self-contained single-file scripts with inline dependency declarations, making them easy to distribute and maintain.
