# Claude Format Hook - ryanlewis/claude-format-hook
- **Source**: https://github.com/ryanlewis/claude-format-hook
- **Retrieved**: 2026-03-27

## Overview
Auto-format code when Claude Code edits files. Supports JavaScript/TypeScript, Python, Go, Kotlin, and Markdown with popular formatters like Biome, Ruff, and Prettier.

## Supported Formatters

| Language | Formatters | Extensions |
|----------|-----------|-----------|
| JavaScript/TypeScript | Biome (with Prettier fallback) | `.js`, `.jsx`, `.ts`, `.tsx` |
| Python | Ruff | `.py` |
| Markdown | Prettier | `.md` |
| Go | goimports + go fmt | `.go` |
| Kotlin | ktlint (with ktfmt fallback) | `.kt`, `.kts` |

## Hooked Events
- **Event**: PostToolUse
- **Matcher**: `Edit|MultiEdit|Write`
- **Type**: command
- **Command**: Runs `format-code.sh` script

## Configuration
Hook activates through `~/.claude/settings.json`. Configuration uses a PostToolUse hook that executes a shell script whenever matching operations complete.

## Key Characteristics
- **Graceful degradation**: Missing formatters are silently skipped without interrupting Claude's workflow
- **Background, non-blocking operation**
- **File type detection by extension**: Only attempts formatting if corresponding tool is installed and accessible via system PATH
