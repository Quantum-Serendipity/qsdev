---
name: gdev-update
description: Update gdev-managed configuration files to the latest templates and settings.
disable-model-invocation: true
allowed-tools: Bash(gdev *) Read Grep Glob
---

# gdev update

## Current Environment

!`gdev status --json 2>/dev/null || echo '{"tools": {}}'`

!`git log --oneline -5 2>/dev/null || echo 'not a git repo'`

## Instructions

1. **Show current state**: Present the current gdev status and recent git changes from the output above.

2. **Preview changes**: Run `gdev init --update --dry-run --json` to preview what will be updated. Present:
   - Files that will be modified
   - New settings or tools that will be added
   - Deprecated settings that will be removed

3. **Get confirmation**: Ask the user to confirm before applying the update.

4. **Run update**: Execute `gdev init --update` to apply the configuration updates.

5. **Verify results**: Run `gdev devenv doctor --json` to confirm the update succeeded and no new issues were introduced.

6. **Summarize**: Report what was updated, any new features or tools that were added, and any breaking changes or manual steps needed.
