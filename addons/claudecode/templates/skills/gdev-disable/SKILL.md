---
name: gdev-disable
description: Disable a tool from the gdev project configuration.
disable-model-invocation: true
allowed-tools: Bash(gdev *) Read Grep Glob
arguments:
  - tool
---

# gdev disable

## Current Environment

!`gdev status --json 2>/dev/null || echo '{"tools": {}}'`

## Instructions

1. **Check arguments**: If no tool name was provided, show the list of currently enabled tools from the status output above and ask which one to disable. If a tool name was given, proceed.

2. **Verify tool status**: Check whether the tool is currently enabled. If it is not enabled, inform the user and stop.

3. **Check dependents**: Review the status output for any tools that depend on the tool being disabled. If dependents exist, warn the user that disabling this tool may affect them.

4. **Disable the tool**: Run `gdev disable $tool` where `$tool` is the tool name provided or selected.

5. **Verify**: Run `gdev status --json` to confirm the tool is now disabled.

6. **Report changes**: Summarize what was disabled, any files that were removed, and note any dependent tools that may need attention.
