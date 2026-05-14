---
name: gdev-onboard
description: Onboard an existing project to gdev. Analyzes gaps in current configuration and merges gdev settings non-destructively.
disable-model-invocation: true
allowed-tools: Bash(gdev *) Read Grep Glob
argument-hint: "[--profile <name>]"
---

# gdev onboard

## Current Environment

!`gdev devenv doctor --json 2>/dev/null || echo '{"installed": false}'`

!`ls -la .claude/ devenv.nix devenv.yaml .envrc .mcp.json 2>/dev/null || echo 'no existing config files'`

!`gdev detect --json 2>/dev/null || echo '{"ecosystems": []}'`

## Instructions

1. **Analyze existing configuration**: Review the doctor output and existing config files above. Identify what is already configured and what is missing.

2. **Present gap analysis**: Show the user:
   - What is already configured (existing devenv, Claude Code settings, hooks, etc.)
   - What is missing or could be improved
   - What gdev would add or modify during onboarding

3. **Get confirmation**: Present the planned changes and ask the user to confirm before proceeding.

4. **Run onboarding**: Execute `gdev init --merge --non-interactive` to merge gdev configuration into the existing project without overwriting user customizations. Pass through any `--profile` argument.

5. **Verify results**: Run `gdev devenv doctor --json` to confirm onboarding succeeded. Compare before and after states.

6. **Summarize**: Report what changed:
   - New files created
   - Existing files that were updated (merged)
   - Files that were left unchanged
   - Any warnings or manual steps needed
