# Review Pull Request

Perform a structured, thorough review of the current pull request.

## Gather context

1. Identify the PR diff: `git diff main...HEAD` (or the appropriate base branch).
2. Read the PR description if available: `gh pr view --json title,body`.
3. List all changed files: `git diff main...HEAD --name-only`.
4. Check the total diff size: `git diff main...HEAD --stat`.

## Per-file review checklist

For each changed file, evaluate the following:

### Correctness
- Does the logic match the stated intent of the PR?
- Are there off-by-one errors, nil pointer risks, or unhandled edge cases?
- Are new branches in conditionals complete (no missing else/default)?

### Error handling
- Are errors checked and propagated with context (e.g., `fmt.Errorf("...: %w", err)`)?
- Are error messages descriptive enough for debugging?
- Is there appropriate fallback behavior for recoverable errors?

### Tests
- Do new functions have corresponding test coverage?
- Are edge cases and error paths tested, not just the happy path?
- Are tests deterministic and free of external dependencies?

### Performance
- Are there unnecessary allocations in hot paths?
- Could any O(n^2) or worse algorithms be replaced with more efficient approaches?
- Are database queries or I/O calls batched where possible?

### Security
- Is user input validated and sanitized before use?
- Are SQL queries parameterized?
- Are secrets or credentials absent from the diff?
- Are file permissions appropriate?

## Run automated checks

1. Execute the test suite: use the project's test command.
2. Run the linter: use the project's lint command.
3. If a type checker is available, run it (e.g., `tsc --noEmit`, `mypy`).
4. Report any failures with file and line references.

## Produce review summary

Structure the output as:

```
## Summary
<1-2 sentence overview of what the PR does>

## Findings
- **[file:line]** <severity: critical/warning/nit> — <description>
- ...

## Tests
<pass/fail summary, any gaps identified>

## Verdict
<APPROVE / REQUEST CHANGES — with rationale>
```

Reference specific file paths and line numbers for every finding. Group
findings by severity, with critical issues first.
