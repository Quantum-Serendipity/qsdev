---
name: qsdev-onboard
description: Onboard an existing project to qsdev. Analyzes gaps in current configuration and merges qsdev settings non-destructively.
disable-model-invocation: true
allowed-tools: Bash(qsdev *) Read Grep Glob
argument-hint: "[--profile <name>]"
---

# qsdev onboard

## Current Environment

!`qsdev devenv doctor --json 2>/dev/null || echo '{"installed": false}'`

!`ls -la .claude/ devenv.nix devenv.yaml .envrc .mcp.json 2>/dev/null || echo 'no existing config files'`

!`qsdev detect --json 2>/dev/null || echo '{"ecosystems": []}'`

## Instructions

1. **Analyze existing configuration**: Review the doctor output and existing config files above. Identify what is already configured and what is missing.

2. **Present gap analysis**: Show the user:
   - What is already configured (existing devenv, Claude Code settings, hooks, etc.)
   - What is missing or could be improved
   - What qsdev would add or modify during onboarding

3. **Get confirmation**: Present the planned changes and ask the user to confirm before proceeding.

4. **Run onboarding**: Execute `qsdev init --merge --non-interactive` to merge qsdev configuration into the existing project without overwriting user customizations. Pass through any `--profile` argument.

5. **Verify results**: Run `qsdev devenv doctor --json` to confirm onboarding succeeded. Compare before and after states.

6. **Summarize**: Report what changed:
   - New files created
   - Existing files that were updated (merged)
   - Files that were left unchanged
   - Any warnings or manual steps needed
