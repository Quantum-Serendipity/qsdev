---
name: qsdev-init
description: Initialize a new project with qsdev. Detects ecosystem, generates security-hardened devenv configs, Claude Code settings, and pre-commit hooks.
disable-model-invocation: true
allowed-tools: Bash(qsdev *) Read Grep Glob
argument-hint: "[--profile <name>] [--yes]"
---

# qsdev init

## Current Environment

!`qsdev devenv doctor --json 2>/dev/null || echo '{"installed": false}'`

!`ls -la`

!`qsdev detect --json 2>/dev/null || echo '{"ecosystems": []}'`

## Instructions

1. **Check prerequisites**: Verify qsdev is installed from the doctor output above. If `installed` is false, tell the user to install qsdev first and stop.

2. **Review detected ecosystems**: Present the ecosystems detected above to the user. Confirm that the detection looks correct and ask if any adjustments are needed.

3. **Dry run first**: Run `qsdev init --dry-run --json` to preview what will be generated. If the user provided `--profile`, pass it through. Present the planned changes to the user:
   - Files that will be created or modified
   - Ecosystems that will be configured
   - Security tools that will be enabled

4. **Get confirmation**: Ask the user to confirm before proceeding. If the user passed `--yes` as an argument, skip confirmation.

5. **Run init**: Execute `qsdev init` with any arguments the user provided. Monitor the output for errors.

6. **Verify results**: Run `qsdev devenv doctor --json` to confirm the initialization succeeded. Check for any FAIL or WARN items in the results.

7. **Summarize**: Report what was configured:
   - Languages/ecosystems enabled
   - Security tools activated
   - Files created or modified
   - Any warnings or issues that need attention
