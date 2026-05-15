---
name: qsdev-setup
description: Install missing prerequisites detected by qsdev doctor. Resolves FAIL and WARN items in the health check.
disable-model-invocation: true
allowed-tools: Bash(qsdev *) Read Grep Glob
---

# qsdev setup

## Current Environment

!`qsdev devenv doctor --json 2>/dev/null | jq '{checks: [.checks[] | select(.status != "PASS")]}' 2>/dev/null || qsdev devenv doctor --json 2>/dev/null || echo '{"installed": false}'`

## Instructions

1. **Show failing checks**: Present the FAIL and WARN items from the doctor output above. Explain what each issue means and how it will be resolved.

2. **Preview fixes**: Run `qsdev devenv setup --dry-run` to show what actions will be taken. Present the plan to the user.

3. **Get confirmation**: Ask the user to confirm before making changes.

4. **Run setup**: Execute `qsdev devenv setup` to install missing prerequisites and fix detected issues. Monitor for errors.

5. **Verify fixes**: Run `qsdev devenv doctor --json` again to confirm the issues are resolved. Report any remaining FAIL or WARN items.

6. **Report results**: Summarize what was installed or fixed, and note any issues that require manual intervention.
