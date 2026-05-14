---
name: write-adr
description: Generate an Architecture Decision Record (ADR) in MADR format.
disable-model-invocation: true
allowed-tools: Read Write Edit Grep Glob Bash(ls *) Bash(find *) Bash(git log *)
arguments: [decision-title]
argument-hint: "use-postgresql-over-mongodb"
---

# Write ADR

Generate an Architecture Decision Record (ADR) in MADR format.

## Step 1: Gather Context

1. Review recent commits for related changes: `git log --oneline -20`.
2. Look for configuration changes, new dependencies, or architectural shifts.
3. Check existing ADRs in `docs/adr/` for numbering and style conventions.

## Step 2: Interview User

Ask the user about:

1. **Problem statement**: What problem does this decision solve?
2. **Alternatives considered**: What other options were evaluated?
3. **Constraints**: What technical, business, or team constraints apply?
4. **Deciding factor**: What was the primary reason for choosing this option?

## Step 3: Generate ADR

Determine the next ADR number by examining existing files in `docs/adr/`.

Create the ADR at `docs/adr/NNNN-{decision-title}.md` using MADR format:

```markdown
# NNNN - {Title}

## Status

Accepted

## Date

{YYYY-MM-DD}

## Context

{Problem description and background}

## Decision

{What was decided and why}

## Consequences

### Positive
- {Benefits}

### Negative
- {Tradeoffs}

### Neutral
- {Other effects}

## Alternatives Considered

### {Alternative 1}
- Pros: ...
- Cons: ...
- Reason rejected: ...

### {Alternative 2}
- Pros: ...
- Cons: ...
- Reason rejected: ...
```

## Step 4: Verify Consistency

1. Ensure the ADR number is unique.
2. Verify the decision aligns with existing ADRs (no contradictions).
3. Confirm the file is well-formatted and complete.
