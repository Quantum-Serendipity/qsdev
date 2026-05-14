# Claude Code --add-dir CLAUDE.md Loading
- **Source**: https://github.com/anthropics/claude-code/issues/21138
- **Retrieved**: 2026-05-14

## Files Loaded

When `CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1` is enabled, Claude Code loads:
- `CLAUDE.md` files from additional directories
- `CLAUDE.local.md` files from additional directories

These are loaded alongside project memories from the current working directory chain.

## Environment Variable Control

- `CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1`: Enables loading CLAUDE.md files from additional directories
- When not set or `0`: Additional directories only extend file access, do NOT contribute to memory
- Opt-in design for backward compatibility

## How It Works

### Standard Memory Loading (Always Active)
- Starting in the cwd, recurses up to (but not including) root `/`
- Loads any CLAUDE.md or CLAUDE.local.md files found

### Additional Directory Memory Loading (Opt-in)
- Additional directories specified via `--add-dir`, `/add-dir`, or `additionalDirectories` setting
- CLAUDE.md/CLAUDE.local.md files from these directories are loaded
- Files follow the same permission rules as the original working directory

## Configuration Methods
- CLI: `--add-dir <path>`
- Interactive command: `/add-dir`
- Settings: `additionalDirectories` configuration

## Feature Introduction
Version: Claude Code 2.1.20

## Note
Feature was undocumented in official docs as of the issue report.
