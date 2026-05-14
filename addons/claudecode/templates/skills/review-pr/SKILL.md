---
name: review-pr
description: Comprehensive PR review across security, performance, and code quality with structured findings.
disable-model-invocation: true
allowed-tools: Bash(git *) Bash(gh *) Read Grep Glob
arguments: [pr-number]
argument-hint: "123"
---

# Comprehensive PR Review

## 1. Gather PR Context

Fetch the PR metadata and diff:

```
gh pr view $0 --json title,body,additions,deletions,changedFiles 2>/dev/null || echo "PR not found"
```

```
gh pr diff $0 --name-only 2>/dev/null || echo "Cannot fetch diff"
```

Get the full diff for review:

```
gh pr diff $0 2>/dev/null || git diff main...HEAD
```

## 2. Review Dimensions

### Code Quality
- Readability and clarity of naming
- Consistent coding style with the rest of the codebase
- Appropriate abstraction level
- DRY principle adherence
- Function/method length and complexity

### Security
- Injection vulnerabilities (SQL, command, template)
- Input validation and sanitization
- Secrets or credentials in the diff
- Authentication and authorization correctness
- Dependency vulnerabilities (check for known CVEs)
- File permission appropriateness

### Performance
- N+1 query patterns
- Unnecessary allocations in hot paths
- Missing or incorrect indexing
- Unbounded growth (slices, maps, channels)
- Blocking operations in async contexts

### Testing
- New code has corresponding test coverage
- Edge cases and error paths are tested
- Tests are deterministic and isolated
- Mock/stub usage is appropriate

### Maintainability
- Documentation for public APIs
- Error messages are actionable
- Logging is sufficient for debugging
- Configuration is externalized appropriately

## 3. Synthesis

Produce a structured review with:

- **Verdict**: APPROVE, REQUEST CHANGES, or NEEDS DISCUSSION
- **Blocking issues**: Critical problems that must be fixed
- **Suggestions**: Non-blocking improvements
- **Praise**: Well-done aspects worth highlighting

Format findings as:
```
**[file:line]** <severity: critical/warning/nit> - <description>
```

Group by severity with critical issues first.
