---
name: incident-debug
description: Systematic production incident debugging with hypothesis-test-conclude methodology.
disable-model-invocation: true
allowed-tools: Bash(*) Read Grep Glob
arguments: [symptom]
argument-hint: "API returning 500 errors on /users endpoint"
---

# Incident Debug

Systematic production incident debugging using hypothesis-test-conclude methodology.

**IMPORTANT**: Do not make any production changes without explicit user approval.

## Step 1: Triage

1. **When did it start?** Check deployment logs, recent releases, and timestamps.
2. **Blast radius**: How many users/services are affected? Is it total or partial?
3. **Recent changes**: What was deployed or changed in the last 24-48 hours?
4. **Error patterns**: Examine logs, error rates, and monitoring dashboards.

## Step 2: Hypothesize

Generate 3-5 ranked hypotheses for the root cause:

1. Most likely cause based on symptoms and recent changes.
2. Second most likely cause.
3. Infrastructure-related possibility.
4. Data-related possibility.
5. External dependency failure possibility.

For each hypothesis, define:
- What evidence would confirm it
- What evidence would refute it
- How to test safely

## Step 3: Test Each Hypothesis

For each hypothesis, systematically gather evidence:

1. Execute the defined test procedure.
2. Record result as: **CONFIRMED**, **REFUTED**, or **INCONCLUSIVE**.
3. If inconclusive, identify what additional data is needed.
4. Move to the next hypothesis if refuted.

## Step 4: Root Cause

Once a hypothesis is confirmed:

1. Document the root cause clearly.
2. Explain the chain of events that led to the incident.
3. Identify any contributing factors.

## Step 5: Remediation Options

Present three tiers of remediation:

1. **Immediate**: Quick fix to restore service (hotfix, rollback, config change).
2. **Short-term**: Proper fix addressing the root cause (days).
3. **Long-term**: Systemic improvements to prevent recurrence (weeks).

## Step 6: Verification

After the fix is applied:

1. Confirm the original symptom is resolved.
2. Verify no new issues were introduced.
3. Monitor for recurrence over a defined period.
