---
source: https://code.claude.com/docs/en/hooks
retrieved: 2026-05-12
note: Extracted section on "Hooks in skills and agents" only
---

# Hooks in Skills and Agents

Hooks can be defined directly in skills and subagents using frontmatter. These hooks are scoped to the component's lifecycle and only run when that component is active.

## Key Characteristics

- All hook events are supported
- Lifecycle scoping: Hooks only run when the skill or agent is active
- Automatic conversion: For subagents, Stop hooks are automatically converted to SubagentStop
- Automatic cleanup: Hooks are cleaned up when the component finishes

## Configuration Format

Hooks use the same configuration format as settings-based hooks but are scoped to the component's lifetime:

```yaml
---
name: secure-operations
description: Perform operations with security checks
hooks:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "./scripts/security-check.sh"
---
```

## Example: Skill with Security Validation

This skill defines a PreToolUse hook that runs a security validation script before each Bash command:

```yaml
---
name: secure-operations
description: Perform operations with security checks
hooks:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "./scripts/security-check.sh"
---
```

Agents use the same format in their YAML frontmatter as skills do.
