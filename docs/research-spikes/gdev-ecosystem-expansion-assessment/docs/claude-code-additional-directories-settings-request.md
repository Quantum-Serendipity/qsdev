# Feature Request: Configure Additional Directories via Settings Files
- **Source**: https://github.com/anthropics/claude-code/issues/3146
- **Retrieved**: 2026-05-14

## What's Being Requested

Ability to configure additional directories in Claude Code's settings files rather than only through CLI flags or slash commands.

## Proposed Solution: settings.json Configuration

```json
{
  "additionalDirectories": [
    "../backend",
    {
      "path": "../shared-components",
      "readClaudeMd": true
    },
    {
      "path": "~/company/templates",
      "readClaudeMd": false
    }
  ]
}
```

## Key Feature: CLAUDE.md Control

- `readClaudeMd: true` - automatically reads CLAUDE.md from that directory
- `readClaudeMd: false` - ignores CLAUDE.md for that directory
- Default behavior matches current `--add-dir` (don't read CLAUDE.md)

## Primary Use Cases

1. Monorepo development with multiple services
2. Shared libraries and design systems
3. Legacy system references
4. Team knowledge sharing across directories
