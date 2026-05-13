# agent-postmortem-skill — README.md
# Source: https://github.com/plus8bit/agent-postmortem-skill/blob/develop/README.md
# Retrieved: 2026-05-12

# agent-postmortem-skill

Stop letting AI agents lie to you.

`agent-postmortem-skill` is an open-source verification skill that forces coding agents to prove work with evidence before they claim a task is complete.

## Why This Exists

AI coding agents often report "done" while one or more of these are still true:
- Files were not actually changed as requested.
- Tests/build were never run.
- Commands failed but the agent still moved on.
- The final summary sounds confident but has no proof.

This skill turns "trust me" into "show me."

## Value Proposition

- Catches fake-done states before they hit your branch.
- Standardizes completion quality across humans and agents.
- Produces a portable postmortem artifact you can review, share, and audit.
- Works with any coding agent that can run shell commands and read git state.

## How It Works (Under the Hood)

The skill enforces a strict completion pipeline:

1. Intent Snapshot  
   Capture the exact requested outcome and success criteria.

2. Evidence Collection  
   Collect hard signals: `git status`, `git diff`, command outputs, and exit codes.

3. Verification Check  
   Compare claimed work vs actual evidence. If evidence is missing or failing, the task is not complete.

4. Postmortem Output  
   Write a final report with verdict, proof, unresolved risks, and next actions.

## Quickstart

```bash
git clone https://github.com/plus8bit/agent-postmortem-skill.git
cd agent-postmortem-skill
```

Copy `SKILL.md` into your agent skill directory (example paths):

```bash
mkdir -p ~/.claude/skills/agent-postmortem
cp SKILL.md ~/.claude/skills/agent-postmortem/SKILL.md
```

Use it in your coding flow:

```bash
# Pseudoflow, depends on your agent runtime:
# 1) Load SKILL.md
# 2) Run your implementation task
# 3) Require postmortem before "done"
```

## Example Output

```markdown
# Agent Postmortem Report

## Task
Refactor auth middleware and add regression tests for expired token handling.

## Intent Snapshot
- Expected outcome: middleware rejects expired tokens with 401.
- Required checks: npm run build, npm test.

## Evidence Collection
- git status: 3 files modified
- git diff: src/middleware/auth.ts, src/middleware/auth.test.ts, docs/llms.txt
- command: npm run build
  - exit_code: 0
- command: npm test
  - exit_code: 0

## Verification
- Claimed refactor exists in diff: PASS
- Claimed tests added: PASS
- Required commands succeeded: PASS

## Verdict
VERIFIED DONE

## Residual Risks
- No load testing performed.

## Next Actions
- Optional: run e2e auth flow in CI preview environment.
```

## What This Project Is Not

- Not another coding agent.
- Not a replacement for CI.
- Not a generic "quality checklist."

It is a focused lie detector for agent completion claims.

## License

MIT
