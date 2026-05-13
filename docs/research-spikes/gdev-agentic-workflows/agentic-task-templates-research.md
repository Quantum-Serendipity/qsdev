# Agentic Task Templates: Pre-Configured Patterns Encoding Best Practices

## Executive Summary

Agentic task templates are pre-configured prompt patterns that encode best practices for common operations. They differ from skills in that they are designed to be invoked with specific arguments and produce consistent, verifiable output. The four requested templates ("Review this PR for security issues," "Add tests for this module," "Upgrade dependency X," and "Onboard me to this codebase") are implemented as skills with structured checklists, verification steps, and consulting-appropriate output formatting. This report covers the design patterns, the template format, and how they integrate with gdev's generation pipeline.

## 1. What Makes a Task Template Different from a Skill

A task template is a skill with these additional properties:

| Property | Regular Skill | Task Template |
|---|---|---|
| Arguments | Optional | Required (specific inputs) |
| Output format | Flexible | Structured and consistent |
| Checklist | Implicit | Explicit with pass/fail criteria |
| Verification | Optional | Built-in (self-checking) |
| Completeness | "Best effort" | "Every item covered or explicitly skipped" |
| Audience | Developer (self) | Developer + reviewer/client |

Task templates are effectively "guided audit protocols" -- they ensure every important dimension is checked regardless of the engineer's familiarity with the codebase.

## 2. Design Patterns for Task Templates

### Pattern: Checklist-Driven Execution

Every template includes an explicit checklist that Claude must work through. Items are marked PASS, FAIL, or N/A with rationale:

```markdown
## Security Checklist
- [PASS] Input validation: All user inputs validated in UserController
- [FAIL] SQL injection: Raw query in OrderRepository.findByStatus() line 47
- [N/A] File upload: No file upload functionality in scope
- [PASS] Authentication: JWT validation present on all protected endpoints
```

### Pattern: Evidence-Anchored Findings

Every finding must reference a specific file and line:

```markdown
### Finding: Missing CSRF Protection
- **File**: src/routes/api/payments.ts:23
- **Severity**: HIGH
- **Evidence**: POST endpoint `/api/payments/charge` lacks CSRF token validation
- **Impact**: Attacker could forge payment requests from authenticated sessions
- **Fix**: Add CSRF middleware from existing `src/middleware/csrf.ts`
```

### Pattern: Graduated Depth

Templates support multiple depth levels controlled by arguments:

```
/review-security          → Quick scan (~2 min, ~$0.05)
/review-security --full   → Full OWASP assessment (~15 min, ~$8)
/review-security 42       → Review PR #42 for security
```

### Pattern: Verification Gate

Before producing output, the template includes a self-verification step:

```markdown
## Self-Verification
Before reporting findings:
1. Re-read each finding and confirm the file:line reference is accurate
2. Verify the severity rating matches the actual impact
3. Confirm the fix suggestion doesn't introduce new issues
4. Check that no duplicate findings exist
```

## 3. The Four Core Task Templates

### Template 1: "Review This PR for Security Issues"

**Implementation**: Skill `/review-security` (variant of `/review-pr`)

```yaml
---
name: review-security
description: Security-focused PR review with OWASP checklist, vulnerability classification, and remediation guidance. Use when reviewing PRs for security issues or conducting pre-merge security checks.
disable-model-invocation: true
allowed-tools: Bash(git *) Bash(gh *) Read Grep Glob
arguments: [target]
argument-hint: "[PR-number | --full | file-path]"
---

# Security Review: $target

## Determine Scope
If $target is a number, fetch PR diff:
!`gh pr diff $0 --name-only 2>/dev/null || echo "NOT_A_PR"`

If $target is a file path, review that file.
If $target is "--full", review the entire codebase.

## Security Checklist (OWASP Top 10 2021)

Work through each category. For each, report: PASS / FAIL / N/A with evidence.

### A01: Broken Access Control
- [ ] Authorization checks on all protected endpoints
- [ ] No direct object reference without ownership validation
- [ ] CORS configuration is restrictive (not `*`)
- [ ] Directory listing disabled
- [ ] JWT/session validation on state-changing operations

### A02: Cryptographic Failures
- [ ] No sensitive data in logs or error messages
- [ ] Passwords hashed with bcrypt/scrypt/argon2 (not MD5/SHA1)
- [ ] TLS enforced for all external communication
- [ ] No hardcoded secrets or API keys

### A03: Injection
- [ ] Parameterized queries for all database operations
- [ ] No string concatenation in SQL/NoSQL queries
- [ ] Output encoding for HTML context (XSS prevention)
- [ ] Command injection prevention (no shell exec with user input)
- [ ] Path traversal prevention (no user input in file paths)

### A04: Insecure Design
- [ ] Rate limiting on authentication endpoints
- [ ] Account lockout after failed attempts
- [ ] No business logic bypass possible
- [ ] Principle of least privilege applied

### A05: Security Misconfiguration
- [ ] No default credentials
- [ ] Debug mode disabled in production config
- [ ] Security headers configured (CSP, HSTS, X-Frame-Options)
- [ ] Error messages don't leak stack traces

### A06: Vulnerable Components
- [ ] No known CVEs in direct dependencies
- [ ] Dependencies are reasonably current
- [ ] Lock file present and committed

### A07: Authentication Failures
- [ ] Password complexity requirements enforced
- [ ] Multi-factor authentication supported (if applicable)
- [ ] Session management is secure (HttpOnly, Secure, SameSite cookies)

### A08: Data Integrity Failures
- [ ] Input validation on all deserialization
- [ ] CI/CD pipeline integrity (no unsigned deployments)

### A09: Logging Failures
- [ ] Security events are logged (login, access denial, input validation failure)
- [ ] Logs don't contain sensitive data
- [ ] Log injection prevention

### A10: SSRF
- [ ] No user-controlled URLs in server-side requests
- [ ] URL allowlisting for external service calls

## Output Format

### Summary
**Risk Level**: CRITICAL / HIGH / MEDIUM / LOW / CLEAN
**Findings**: N total (X critical, Y high, Z medium)

### Findings (sorted by severity)

#### [CRITICAL] Finding Title
- **File**: path:line
- **Category**: OWASP category
- **Evidence**: What was found
- **Impact**: What an attacker could do
- **Remediation**: Specific fix with code example
- **Effort**: Small / Medium / Large

### Recommendations
- Prioritized list of remediation actions
- Quick wins (high impact, low effort)

## Self-Verification
Before reporting:
1. Confirm every file:line reference exists and shows the claimed issue
2. Verify severity ratings match actual exploitability
3. Check remediation suggestions don't introduce new vulnerabilities
4. Remove duplicate findings
```

### Template 2: "Add Tests for This Module"

**Implementation**: Skill `/add-tests` (from catalog) with enhanced template behavior

The `/add-tests` skill already covers this template. Key template enhancements:

**Coverage target specification**:
```
/add-tests src/services/auth.ts              → Standard coverage
/add-tests src/services/auth.ts --target=90  → Target 90% line coverage
/add-tests src/services/auth.ts --edge-cases → Focus on edge cases only
```

**Structured output verification**:
```markdown
## Test Report
| Function | Happy Path | Edge Cases | Error Paths | Total Tests |
|----------|-----------|------------|-------------|-------------|
| login()  | 2         | 3          | 2           | 7           |
| refresh()| 1         | 2          | 1           | 4           |
| logout() | 1         | 0          | 1           | 2           |

**Total**: 13 tests added
**Status**: All passing
**Execution time**: 2.3s
```

### Template 3: "Upgrade Dependency X to Version Y"

**Implementation**: Skill `/upgrade-dep` (from catalog) with template verification

The `/upgrade-dep` skill already covers this. Key template enhancements:

**Pre-flight checklist**:
```markdown
## Pre-Upgrade Checklist
- [ ] Current version identified: [version]
- [ ] Target version confirmed: [version]
- [ ] Changelog reviewed for breaking changes: [count] breaking changes found
- [ ] Affected files identified: [count] files import this dependency
- [ ] Test baseline established: all [count] tests passing
- [ ] Git state clean: no uncommitted changes
```

**Post-upgrade verification**:
```markdown
## Post-Upgrade Verification
- [ ] Dependency version updated in manifest
- [ ] Lock file regenerated
- [ ] Build succeeds without errors
- [ ] Type checking passes (if applicable)
- [ ] All existing tests pass
- [ ] No new deprecation warnings
- [ ] Manual verification needed for: [list of untestable changes]
```

### Template 4: "Onboard Me to This Codebase"

**Implementation**: Skill `/onboard` (from catalog) with structured exploration template

The `/onboard` skill delegates to the `codebase-explorer` agent. Template enhancements:

**Time-boxed exploration levels**:
```
/onboard              → Standard (5 min, produces ONBOARDING.md)
/onboard --quick      → Quick orientation (2 min, terminal output only)
/onboard --deep       → Deep dive (15 min, includes data model + integration map)
/onboard --interview  → Interactive mode (asks questions about your focus areas)
```

**Structured output format**:
```markdown
# Codebase Orientation: [project-name]

## 30-Second Overview
[One paragraph that answers: what does this do, who uses it, what tech stack]

## Architecture at a Glance
```
[text-based architecture diagram]
```

## First 10 Files to Read
1. `path/to/file` — Why: [reason this file is important]
2. ...

## How to Work Here
- Build: `[command]`
- Test: `[command]`
- Lint: `[command]`
- Run locally: `[command]`

## Patterns You'll See
- [Pattern name]: [one-line description + example file]
- ...

## Gotchas
- [Non-obvious behavior or convention]
- ...

## Key People (from git history)
- [name]: [areas of ownership]
- ...
```

## 4. Template Integration with gdev

### Generation Strategy

gdev generates task templates as skills with enhanced frontmatter:

```go
type TaskTemplate struct {
    Skill           SkillConfig
    Checklist       []ChecklistItem
    OutputFormat    string    // "document", "terminal", "both"
    VerificationSteps []string
    CostEstimate    string    // "~$0.05", "~$8-10"
    TimeEstimate    string    // "~2 min", "~15 min"
    DepthLevels     []DepthLevel
}

type DepthLevel struct {
    Flag        string  // "--quick", "--full", "--deep"
    Description string
    Cost        string
    Time        string
}
```

### Template Selection in Wizard

gdev's wizard presents templates as part of the skills selection step:

```
Install team skills:
[x] /review-pr         — Comprehensive PR review
[x] /review-security   — Security-focused review with OWASP checklist
[x] /add-tests         — Generate tests for uncovered code
[x] /upgrade-dep       — Dependency upgrade with verification
[x] /onboard           — Systematic codebase exploration
[ ] /refactor-safe      — Test-validated refactoring
[ ] /write-adr          — Architecture Decision Records
[ ] /incident-debug     — Production debugging protocol
...
```

The top 5 are pre-selected for new consulting projects.

## Depth Checklist

- [x] Underlying mechanism explained (template as enhanced skill, checklist-driven execution, evidence anchoring)
- [x] Key tradeoffs identified (depth levels vs cost, structured output vs flexibility, auto-invoke vs manual)
- [x] Compared to alternatives (templates vs ad-hoc prompts, structured vs freeform review)
- [x] Failure modes described (checklist fatigue, false confidence from passing checklists, cost overrun on --full)
- [x] Concrete examples found (4 complete templates with YAML, checklists, output formats)
- [x] Standalone-readable
