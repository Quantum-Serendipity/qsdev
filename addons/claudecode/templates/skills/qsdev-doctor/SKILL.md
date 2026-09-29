---
name: qsdev-doctor
description: Run qsdev health diagnostics and analyze results.
allowed-tools: Bash(qsdev *) Read Grep Glob
---

# qsdev doctor

## Current Environment

!`qsdev devenv doctor --json 2>/dev/null || echo "ERROR: 'qsdev devenv doctor --json' exited with status $?. Any output above may be partial; do not treat missing data as empty."`

## Instructions

1. **Analyze results**: Review the doctor output above. If it reports an ERROR instead of results, say so (qsdev may not be installed) rather than guessing. Categorize each check by status: PASS, WARN, or FAIL.

2. **Report issues**: For any FAIL or WARN items:
   - Explain what the check verifies
   - Describe the impact of the failure
   - Provide the specific fix or remediation steps

3. **Overall assessment**: Summarize the health of the development environment:
   - Total checks passed vs failed
   - Critical issues that block development
   - Warnings that should be addressed

4. **Suggest next steps**: If there are failing checks, suggest running `/qsdev-setup` to automatically resolve them. If everything passes, confirm the environment is healthy.
