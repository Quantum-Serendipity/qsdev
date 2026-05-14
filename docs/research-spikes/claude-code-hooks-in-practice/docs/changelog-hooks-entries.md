# Claude Code Changelog — Hook-Related Entries
- **Source**: https://github.com/anthropics/claude-code/blob/main/CHANGELOG.md
- **Retrieved**: 2026-03-27

## New Hook Events (chronological, recent first)
- v2.1.85: CronCreate (scheduled tasks)
- v2.1.84: TaskCreated, WorktreeCreate HTTP support
- v2.1.83: CwdChanged, FileChanged (reactive environment management)
- v2.1.78: StopFailure (API error turn endings)
- v2.1.76: Elicitation, ElicitationResult, PostCompact

## Hook Fixes
- v2.1.85: PreToolUse can satisfy AskUserQuestion with updatedInput; conditional `if` field added for hooks
- v2.1.83: Fixed uninstalled plugin hooks continuing to fire; fixed plugin hooks blocking prompt when plugin dir deleted
- v2.1.78: SECURITY FIX — PreToolUse "allow" was bypassing deny permission rules including enterprise managed settings
- v2.1.77: Fixed SessionEnd hooks not firing on /resume session switch
- v2.1.75: Suppressed async hook completion messages by default

## Breaking Changes Affecting Hooks
- v2.1.85: Conditional `if` field — new syntax but backward-compatible
- v2.1.82: Plugin options manifest.userConfig now externally available
- v2.1.78: Security fix changed behavior of PreToolUse allow + permission rules interaction

## Key Observation
New hook events added in nearly every minor version (2.1.76, 2.1.78, 2.1.83, 2.1.84, 2.1.85). The API surface is actively expanding. 13 releases in 3 weeks of March 2026 alone.
