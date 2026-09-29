---
name: qsdev-onboard
description: Onboard an existing project to qsdev. Analyzes gaps in current configuration and merges qsdev settings non-destructively.
disable-model-invocation: true
allowed-tools: Bash(qsdev *) Read Grep Glob
argument-hint: "[--profile <name>]"
---

# qsdev onboard

## Current Environment

!`qsdev devenv doctor --json 2>/dev/null || echo "ERROR: 'qsdev devenv doctor --json' exited with status $?. Any output above may be partial; do not treat missing data as empty."`

!`ls -la .claude/ devenv.nix devenv.yaml .envrc .mcp.json 2>/dev/null || echo 'no existing config files'`

!`ls -a`

## Instructions

1. **Analyze existing configuration**: Review the doctor output and existing config files above. Identify what is already configured and what is missing.

2. **Present gap analysis**: Show the user:
   - What is already configured (existing devenv, Claude Code settings, hooks, etc.)
   - What is missing or could be improved
   - What qsdev would add or modify during onboarding

3. **Get confirmation**: Present the planned changes and ask the user to confirm before proceeding.

4. **Run onboarding**: Execute `qsdev init --yes --merge` to merge qsdev configuration into the existing project. `--merge` merges into CLAUDE.md, `.claude/settings.json` and `.mcp.json`, and keeps an existing devenv.nix and .envrc (the generated devenv.nix is written to `devenv.nix.new` for a manual merge). Never use `--force` here: it overwrites existing files. Pass through any `--profile` argument.

5. **Verify results**: Run `qsdev devenv doctor --json` to confirm onboarding succeeded. Compare before and after states.

6. **Summarize**: Report what changed:
   - New files created
   - Existing files that were updated (merged)
   - Files that were left unchanged
   - Any warnings or manual steps needed
