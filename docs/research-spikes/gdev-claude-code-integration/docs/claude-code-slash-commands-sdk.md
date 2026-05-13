---
source: https://code.claude.com/docs/en/agent-sdk/slash-commands
retrieved: 2026-05-12
---

# Slash Commands in the SDK

Commands are stored in designated directories based on scope:
- Project commands: .claude/commands/ (legacy; prefer .claude/skills/)
- Personal commands: ~/.claude/commands/ (legacy; prefer ~/.claude/skills/)

## File Format
- Filename (without .md) becomes the command name
- File content defines what the command does
- Optional YAML frontmatter provides configuration

## Key Features

### Arguments and Placeholders
- $ARGUMENTS for all args
- $1, $2 etc for positional
- argument-hint frontmatter for autocomplete hints

### Bash Command Execution
- !`command` syntax runs before content sent to Claude
- Output replaces placeholder in content

### File References
- @filename includes file contents

### Organization with Namespacing
Subdirectories appear in description but don't affect command name:
```
.claude/commands/
├── frontend/
│   ├── component.md      # Creates /component
│   └── style-check.md    # Creates /style-check
├── backend/
│   ├── api-test.md       # Creates /api-test
│   └── db-migrate.md     # Creates /db-migrate
└── review.md             # Creates /review
```

## Note on Skills vs Commands
.claude/commands/ is legacy format. .claude/skills/<name>/SKILL.md is recommended as it supports:
- Directory for supporting files
- Frontmatter to control invocation
- Autonomous invocation by Claude
- All the same slash-command features
