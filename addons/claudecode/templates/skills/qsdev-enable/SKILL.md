---
name: qsdev-enable
description: Enable a tool in the qsdev project configuration.
disable-model-invocation: true
allowed-tools: Bash(qsdev *) Read Grep Glob
arguments:
  - tool
---

# qsdev enable

## Current Environment

!`qsdev status --json 2>/dev/null || echo '{"tools": {}}'`

!`qsdev list --json 2>/dev/null || echo '{"available": []}'`

## Instructions

1. **Check arguments**: If no tool name was provided, show the list of available tools from the output above organized by category and ask which one to enable. If a tool name was given, proceed.

2. **Verify tool status**: Check whether the tool is already enabled. If it is, inform the user and stop.

3. **Check availability**: Verify the tool exists in the available tools list. If not found, report the error and list similar tool names.

4. **Enable the tool**: Run `qsdev enable $tool` where `$tool` is the tool name provided or selected.

5. **Verify**: Run `qsdev status --json` to confirm the tool is now enabled. Check for any warnings.

6. **Report changes**: Summarize what was enabled and any files that were created or modified as a result.
