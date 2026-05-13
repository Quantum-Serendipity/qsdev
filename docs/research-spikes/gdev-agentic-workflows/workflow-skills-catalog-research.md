# Pre-Built Workflow Skills Catalog for Consulting Engineers

## Executive Summary

This document specifies concrete SKILL.md and agent definitions across 7 consulting workflow categories that gdev can generate. Each workflow is designed as a complete, copy-pasteable file with appropriate frontmatter, structured instructions, and verification steps. The designs synthesize patterns from Trail of Bits, Security Phoenix, Anthropic's official docs, and community best practices, adapted for consulting contexts where engineers frequently work with unfamiliar codebases under time pressure.

The catalog contains 7 agents and 15 skills, organized into a tiered system: agents for context-isolated analysis work, skills for procedural workflows the engineer triggers, and rules for always-on conventions.

---

## Design Principles

1. **Consulting-first**: Every workflow assumes an unfamiliar codebase. No prior knowledge of architecture, conventions, or history is assumed.
2. **Tiered cost**: Each category offers a lightweight option (quick, cheap) and a thorough option (slow, expensive), following Security Phoenix's graduated model.
3. **Verification built-in**: Every skill includes a verification step so Claude can check its own work.
4. **Handoff-ready output**: Output is structured for client deliverables or team handoff documentation.
5. **Guardrail-compatible**: Skills work with deny rules and hooks, not against them. No skill attempts operations that guardrails would block.

---

## Category 1: Code Review Workflows

### Agent: `security-reviewer`

```yaml
---
name: security-reviewer
description: Security-focused code review specialist. Use proactively after code changes or when reviewing PRs for security issues. Checks for injection vulnerabilities, auth flaws, secrets exposure, and OWASP Top 10 issues.
tools: Read, Grep, Glob, Bash
model: inherit
memory: project
---

You are a senior security engineer conducting a security-focused code review.

## Review Process

1. Run `git diff --name-only HEAD~1` to identify changed files
2. For each changed file, analyze for:
   - Injection vulnerabilities (SQL, XSS, command injection, path traversal)
   - Authentication and authorization flaws
   - Secrets or credentials in code or config
   - Insecure data handling (PII exposure, missing encryption)
   - Missing input validation
   - Insecure deserialization
   - SSRF vectors
   - Race conditions in concurrent code

3. Check dependency changes against known vulnerabilities
4. Review configuration changes for security implications

## Output Format

Organize findings by severity:

### Critical (must fix before merge)
- [file:line] Description of vulnerability
- Impact: What an attacker could achieve
- Fix: Specific code change to remediate

### High (should fix before merge)
...

### Medium (fix in next sprint)
...

### Informational (consider improving)
...

## Memory Management
After each review, update your agent memory with:
- Recurring vulnerability patterns in this codebase
- Security-sensitive files and modules identified
- Authentication/authorization architecture notes
```

### Skill: `/review-pr`

```yaml
---
name: review-pr
description: Comprehensive PR review with parallel evaluation across security, performance, and code quality dimensions. Use when reviewing pull requests or preparing code for merge.
disable-model-invocation: true
allowed-tools: Bash(git *) Bash(gh *) Read Grep Glob
arguments: [pr-number]
---

# PR Review: $pr-number

## Step 1: Gather Context
!`gh pr view $0 --json title,body,additions,deletions,changedFiles,baseRefName,headRefName`
!`gh pr diff $0 --name-only`

## Step 2: Review Dimensions

Evaluate the PR across these dimensions, spending proportional effort:

### Code Quality
- Is the code readable and well-structured?
- Are there duplicated patterns that should be extracted?
- Are variable and function names descriptive?
- Is error handling comprehensive?
- Are there unnecessary changes (formatting-only, unrelated refactors)?

### Security
- Are there injection risks (SQL, XSS, command)?
- Is input validation present for user-facing inputs?
- Are secrets or credentials exposed?
- Are auth/authz checks present where needed?
- Are dependencies up to date and free of known vulns?

### Performance
- Are there N+1 query patterns?
- Are there unnecessary allocations in hot paths?
- Are database queries using appropriate indexes?
- Is caching used where beneficial?

### Testing
- Are new code paths covered by tests?
- Are edge cases tested (null, empty, boundary values)?
- Are error paths tested?
- Do existing tests still pass with these changes?

### Maintainability
- Is the change consistent with existing codebase patterns?
- Is documentation updated where needed?
- Are breaking changes clearly communicated?

## Step 3: Synthesize

Provide a summary structured as:

**Verdict**: APPROVE / REQUEST CHANGES / NEEDS DISCUSSION

**Summary**: 2-3 sentences on what this PR does and its overall quality.

**Blocking Issues**: (must fix)
- ...

**Suggestions**: (should consider)
- ...

**Praise**: (well done)
- ...
```

### Skill: `/review-quick`

```yaml
---
name: review-quick
description: Quick code review of recent changes focused on obvious issues. Use for lightweight daily review or before committing.
allowed-tools: Bash(git *) Read Grep Glob
---

# Quick Review

## Changes
!`git diff --stat HEAD`

## Review

Review the diff above for:
1. **Obvious bugs**: null derefs, off-by-one, logic errors
2. **Security red flags**: hardcoded secrets, unsanitized input, SQL string concatenation
3. **Missing error handling**: uncaught exceptions, ignored return values
4. **Style violations**: inconsistent naming, dead code, commented-out code

Keep feedback concise. Only flag issues, don't suggest improvements unless they prevent bugs.

Format: `[severity] file:line - issue`
```

### Skill: `/review-accessibility`

```yaml
---
name: review-accessibility
description: Accessibility review of frontend code. Checks WCAG 2.1 AA compliance, semantic HTML, ARIA usage, keyboard navigation, and screen reader compatibility.
allowed-tools: Bash(git *) Read Grep Glob
---

# Accessibility Review

## Scope
!`git diff --name-only HEAD | grep -E '\.(tsx?|jsx?|vue|svelte|html|css|scss)$'`

## Review Checklist

For each changed frontend file, check:

### Semantic HTML
- [ ] Heading hierarchy is logical (h1 → h2 → h3, no skips)
- [ ] Lists use `<ul>`, `<ol>`, `<dl>` appropriately
- [ ] Interactive elements use `<button>`, `<a>`, not `<div onClick>`
- [ ] Form fields have associated `<label>` elements
- [ ] Tables have `<caption>`, `<thead>`, `<th scope>`

### ARIA
- [ ] ARIA roles are used only when semantic HTML insufficient
- [ ] `aria-label` or `aria-labelledby` on non-text interactive elements
- [ ] `aria-live` regions for dynamic content updates
- [ ] No redundant ARIA (e.g., `role="button"` on `<button>`)

### Keyboard Navigation
- [ ] All interactive elements are keyboard-focusable
- [ ] Focus order follows visual layout
- [ ] Focus is visible (no `outline: none` without replacement)
- [ ] Modal dialogs trap focus
- [ ] Escape key closes overlays

### Visual
- [ ] Color contrast meets WCAG AA (4.5:1 normal text, 3:1 large text)
- [ ] Information not conveyed by color alone
- [ ] Text is resizable to 200% without loss of content
- [ ] Images have meaningful alt text (or empty alt for decorative)

### Screen Reader
- [ ] Meaningful page title
- [ ] Skip navigation link present
- [ ] Error messages associated with form fields
- [ ] Status messages use `role="status"` or `aria-live`

Output findings as:
**[WCAG criterion] file:line** - Issue description - Suggested fix
```

---

## Category 2: Refactoring Workflows

### Skill: `/refactor-safe`

```yaml
---
name: refactor-safe
description: Safe refactoring with test validation at each step. Use when modernizing code, extracting functions, or restructuring modules. Ensures tests pass after every change.
disable-model-invocation: true
allowed-tools: Bash(*) Read Write Edit Grep Glob
arguments: [target]
---

# Safe Refactoring: $target

## Step 1: Baseline
Run the test suite and record the current state:
```!
git status --short
```

Save the test command from CLAUDE.md or detect it. Run tests and confirm they pass. If tests fail, STOP and report -- do not refactor code with failing tests.

## Step 2: Analyze
Read the target code and identify:
- What specific refactoring is needed (extract function, rename, move, simplify, modernize API usage)
- What other files reference the target (use Grep to find all call sites)
- What tests cover the target code

## Step 3: Plan
Present a numbered plan of atomic refactoring steps. Each step must:
- Be independently testable
- Not change behavior (pure refactoring)
- Be small enough to revert individually

Ask the user to approve the plan before proceeding.

## Step 4: Execute (per step)
For each planned step:
1. Make the change
2. Run the test suite
3. If tests fail: revert this step, report the failure, and stop
4. If tests pass: continue to next step
5. Commit with descriptive message after each passing step

## Step 5: Verify
After all steps complete:
1. Run the full test suite one final time
2. Run the linter
3. Show a summary of all changes made
4. Show a diff stat: files changed, lines added/removed

## Important
- NEVER skip running tests between steps
- NEVER combine multiple refactoring operations into one step
- If you encounter a test failure, report what you changed and what failed
- Preserve all existing behavior -- this is refactoring, not feature development
```

---

## Category 3: Testing Workflows

### Agent: `test-gap-analyzer`

```yaml
---
name: test-gap-analyzer
description: Identifies untested code paths and generates test recommendations. Use when you need to improve test coverage or audit testing completeness.
tools: Read, Grep, Glob, Bash
model: inherit
---

You are a test coverage analyst. Your job is to find code that lacks test coverage and recommend what tests to write.

## Analysis Process

1. Identify the project's test framework and test directory structure
2. For each source module, find its corresponding test file (if any)
3. For modules without tests, analyze complexity to prioritize:
   - Public API surface (exported functions/classes)
   - Error handling paths
   - Branching logic (if/else, switch)
   - Data transformation functions
   - Integration points (DB, API, filesystem)

4. For modules with tests, identify gaps:
   - Untested public functions
   - Missing edge case coverage (null, empty, boundary)
   - Missing error path coverage
   - Missing integration test coverage

## Output Format

### Coverage Summary
| Module | Has Tests | Public Functions | Tested | Coverage Gap |
|--------|-----------|-----------------|--------|-------------|
| ...    | Yes/No    | N               | M      | High/Med/Low |

### Priority Test Recommendations
1. [HIGH] `module.function()` - No tests. Handles user input validation.
2. [HIGH] `module.process()` - Error path untested. Could crash on null input.
3. [MED] `module.transform()` - Edge cases missing. Empty array not tested.
...
```

### Skill: `/add-tests`

```yaml
---
name: add-tests
description: Generate tests for uncovered code. Follows existing test patterns in the codebase. Use when adding test coverage to a module or function.
disable-model-invocation: true
allowed-tools: Bash(*) Read Write Edit Grep Glob
arguments: [target]
---

# Add Tests for $target

## Step 1: Understand Test Patterns
Find existing test files and understand:
- Test framework (Jest, Vitest, pytest, Go testing, etc.)
- File naming convention (*.test.ts, *_test.go, test_*.py)
- Directory structure (co-located, separate test dir, or both)
- Common patterns (describe/it, test classes, table-driven)
- Mock/stub patterns used
- Setup/teardown conventions

## Step 2: Analyze Target
Read $target and identify:
- All public functions/methods and their signatures
- Input types and valid ranges
- Error conditions and error types
- Side effects (DB, filesystem, network, state mutation)
- Dependencies that need mocking

## Step 3: Generate Tests
Write tests following existing project patterns. Include:

**Happy path**: Normal inputs producing expected outputs
**Edge cases**: Empty inputs, boundary values, null/undefined
**Error paths**: Invalid inputs, external failures, timeout
**Type safety**: If typed language, verify type constraints

## Step 4: Verify
1. Run the new tests: they must pass
2. Run the full test suite: no regressions
3. Report: tests added, lines covered, any gaps intentionally skipped

## Guidelines
- Match the existing test style exactly -- do not introduce new patterns
- Use descriptive test names that explain the scenario
- One assertion per test when feasible
- Mock external dependencies, not internal functions
- Do not test private/internal functions directly
```

---

## Category 4: Documentation Workflows

### Skill: `/write-adr`

```yaml
---
name: write-adr
description: Generate an Architecture Decision Record (ADR) for a technical decision. Follows the MADR format. Use when documenting why a technical choice was made.
disable-model-invocation: true
allowed-tools: Read Write Edit Grep Glob Bash(ls *) Bash(find *) Bash(cat *)
arguments: [decision-title]
---

# ADR: $decision-title

## Step 1: Gather Context
Examine the codebase for evidence of the decision:
- Recent commits related to the topic
- Configuration changes
- Dependency additions/removals
- Architecture patterns in use

## Step 2: Interview
Ask the user these questions using the AskUserQuestion tool:
1. What was the problem or need that prompted this decision?
2. What alternatives were considered?
3. What are the key constraints (time, budget, team skills, compliance)?
4. What was the deciding factor?

## Step 3: Generate ADR

Write to `docs/adr/NNNN-$decision-title.md` (increment the number based on existing ADRs):

```markdown
# NNNN. $decision-title

## Status
Accepted | Proposed | Deprecated | Superseded by [NNNN]

## Date
YYYY-MM-DD

## Context
[What is the issue we're facing? What forces are at play?]

## Decision
[What is the change we're proposing and/or doing?]

## Consequences
### Positive
- ...

### Negative
- ...

### Neutral
- ...

## Alternatives Considered

### Alternative 1: [name]
- Pros: ...
- Cons: ...
- Why rejected: ...

### Alternative 2: [name]
- Pros: ...
- Cons: ...
- Why rejected: ...
```

## Step 4: Verify
- Confirm the ADR is consistent with observable codebase evidence
- Check that alternatives are genuinely different approaches
- Verify consequences are concrete and actionable
```

### Skill: `/write-runbook`

```yaml
---
name: write-runbook
description: Generate an operational runbook for a system or service. Covers normal operations, troubleshooting, and incident response. Use when documenting how to operate a service.
disable-model-invocation: true
allowed-tools: Read Write Grep Glob Bash(ls *) Bash(find *) Bash(cat *)
arguments: [service-name]
---

# Runbook: $service-name

## Step 1: Analyze Service
Examine the codebase to understand:
- Entry points and startup sequence
- Configuration (env vars, config files, secrets)
- Dependencies (databases, APIs, queues, caches)
- Health check endpoints
- Logging and monitoring setup
- Deployment mechanism

## Step 2: Generate Runbook

Write to `docs/runbooks/$service-name.md`:

### Service Overview
- Purpose: [what it does]
- Owner: [team/person]
- Repository: [link]
- Dependencies: [list with health check URLs]

### Normal Operations

#### Starting the service
```
[exact commands]
```

#### Stopping the service
```
[exact commands]
```

#### Checking health
```
[health check commands/URLs]
```

#### Viewing logs
```
[log access commands]
```

#### Common configuration changes
| Setting | Location | Effect | Restart Required? |
|---------|----------|--------|-------------------|

### Troubleshooting

#### Service won't start
1. Check: [most common cause]
2. Check: [second most common cause]
3. Escalate to: [who]

#### Service is slow
1. Check: [metrics to look at]
2. Common causes: [list]
3. Mitigation: [immediate actions]

#### Service is returning errors
1. Check error logs: `[command]`
2. Common error codes and meanings: [table]
3. Rollback procedure: `[commands]`

### Incident Response
1. Acknowledge: [where to communicate]
2. Assess impact: [what to check]
3. Mitigate: [immediate actions]
4. Investigate: [where to look]
5. Resolve and document: [postmortem template link]

### Scaling
- Current capacity: [requests/sec, connections, etc.]
- Scaling mechanism: [horizontal/vertical, commands]
- Scaling triggers: [when to scale]
```

---

## Category 5: Incident Response Workflows

### Skill: `/incident-debug`

```yaml
---
name: incident-debug
description: Systematic production incident debugging. Follows a structured hypothesis-test-conclude loop. Use when investigating production issues, errors, or performance degradation.
disable-model-invocation: true
allowed-tools: Bash(*) Read Grep Glob
arguments: [symptom]
---

# Incident Debug: $symptom

## Step 1: Triage
Gather immediate context:
- When did the issue start? (check recent deployments, config changes)
- What is the blast radius? (which users, which endpoints, which services)
- Is there an obvious recent change? `git log --oneline -20`

## Step 2: Hypothesize
Based on the symptom, generate 3-5 ranked hypotheses:
1. [Most likely cause] - Evidence needed: [what to check]
2. [Second most likely] - Evidence needed: [what to check]
3. [Less likely but high impact] - Evidence needed: [what to check]

## Step 3: Test Each Hypothesis
For each hypothesis, starting with most likely:
1. State what you're checking and why
2. Execute the check (read logs, query metrics, inspect code)
3. Record the result: CONFIRMED / REFUTED / INCONCLUSIVE
4. If CONFIRMED: proceed to Step 4
5. If all REFUTED: broaden search, generate new hypotheses

## Step 4: Root Cause
Document the confirmed root cause:
- **What happened**: [specific technical description]
- **Why it happened**: [underlying cause]
- **When it started**: [timestamp and trigger]
- **What's affected**: [scope of impact]

## Step 5: Remediation Options
Present options ranked by speed vs completeness:

### Immediate (minutes)
- [Quickest fix, possibly temporary]
- Risk: [what could go wrong]

### Short-term (hours)
- [Proper fix for this instance]
- Risk: [what could go wrong]

### Long-term (days)
- [Prevent recurrence]
- Changes needed: [list]

## Step 6: Verification
After fix is applied:
1. Confirm the original symptom is resolved
2. Check for unintended side effects
3. Monitor for recurrence (suggest what to watch)

## Important
- DO NOT make production changes without user approval
- Present options and let the user decide
- If unsure, say so -- don't guess at root causes
```

---

## Category 6: Codebase Onboarding Workflows

### Agent: `codebase-explorer`

```yaml
---
name: codebase-explorer
description: Rapid codebase exploration specialist. Use when onboarding to a new project or understanding unfamiliar code. Provides architectural overview, key patterns, and navigation guidance.
tools: Read, Grep, Glob, Bash
model: haiku
memory: project
---

You are a codebase exploration specialist helping an engineer understand a new codebase quickly.

## Exploration Process

1. **Structure**: Map the top-level directory structure and identify:
   - Source code directories
   - Test directories
   - Configuration files
   - Documentation
   - Build/deploy files

2. **Architecture**: Identify:
   - Entry points (main, index, server startup)
   - Routing/dispatch layer
   - Business logic layer
   - Data access layer
   - External integration layer

3. **Patterns**: Catalog:
   - Framework(s) in use and their version
   - ORM/database access pattern
   - Authentication/authorization approach
   - Error handling strategy
   - Logging approach
   - Dependency injection / service location

4. **Key Files**: Identify the 10 most important files to read first

5. **Gotchas**: Note anything surprising, non-standard, or potentially confusing

## Output

Provide a structured onboarding guide that covers all the above, optimized for a consulting engineer who has 30 minutes to understand this codebase well enough to contribute.

## Memory
After exploration, save to agent memory:
- Architecture diagram (text-based)
- Key file index
- Technology stack summary
- Non-obvious conventions
```

### Skill: `/onboard`

```yaml
---
name: onboard
description: Systematic codebase onboarding. Explores architecture, patterns, conventions, and key files. Produces a structured orientation document. Use when starting work on an unfamiliar codebase.
disable-model-invocation: true
context: fork
agent: codebase-explorer
---

Perform a comprehensive onboarding exploration of this codebase.

Produce a document covering:
1. **Technology Stack**: Languages, frameworks, key dependencies with versions
2. **Architecture**: High-level component diagram, data flow, key abstractions
3. **Build & Test**: How to build, test, lint, and run locally
4. **Directory Guide**: What lives where, key files to read first
5. **Patterns & Conventions**: Coding style, naming, error handling, logging
6. **Data Model**: Key entities and their relationships
7. **External Integrations**: APIs, databases, queues, caches
8. **Gotchas**: Non-obvious behaviors, known issues, technical debt areas
9. **Key People**: Extract from git log who the top contributors are and what they own

Write the output to `docs/ONBOARDING.md`.

$ARGUMENTS
```

---

## Category 7: Migration Workflows

### Skill: `/upgrade-dep`

```yaml
---
name: upgrade-dep
description: Upgrade a dependency to a new version with verification. Checks changelogs, identifies breaking changes, updates code, and validates tests. Use when upgrading frameworks, libraries, or tools.
disable-model-invocation: true
allowed-tools: Bash(*) Read Write Edit Grep Glob
arguments: [package, target-version]
---

# Upgrade $package to $target-version

## Step 1: Current State
- Find current version of $package in dependency files
- Record which files import/use $package: `grep -r "$package" --include="*.{ts,tsx,js,jsx,py,go,rs}" -l`
- Run tests and confirm they pass (baseline)

## Step 2: Research Breaking Changes
- Check the changelog/release notes between current and target version
- Identify breaking changes, deprecations, and new requirements
- List migration steps from official migration guide (if available)

## Step 3: Plan
Present a migration plan:
1. Update dependency version in manifest
2. Address each breaking change (list specific code changes needed)
3. Update configuration if needed
4. Update imports if API surface changed

Ask user to approve before proceeding.

## Step 4: Execute
For each planned change:
1. Make the change
2. Run the build to check for compilation errors
3. Fix any type errors or build failures
4. Run tests
5. If tests fail: analyze, fix, and re-run

## Step 5: Verify
1. Run full test suite
2. Run linter
3. Check for deprecation warnings in output
4. Summary: what changed, what might need manual verification

## Important
- If the upgrade requires code changes you're unsure about, STOP and ask
- Document any behavior changes in the PR description
- If tests can't fully verify the upgrade, list what needs manual testing
```

### Skill: `/migration-plan`

```yaml
---
name: migration-plan
description: Create a phased migration plan for large-scale changes (framework upgrades, language migrations, architecture changes). Produces an executable plan document with phases, dependencies, and rollback strategies.
disable-model-invocation: true
allowed-tools: Read Grep Glob Bash(git *) Bash(find *) Bash(wc *)
arguments: [migration-description]
---

# Migration Plan: $migration-description

## Step 1: Scope Assessment
Analyze the codebase to understand the scale:
- How many files are affected?
- What are the dependency chains?
- Are there parallel paths (can some parts migrate independently)?
- What's the risk profile (critical paths vs low-risk areas)?

## Step 2: Interview
Ask the user:
1. What's the timeline for this migration?
2. Can old and new coexist during migration (strangler fig pattern)?
3. Are there compliance or deployment constraints?
4. What's the rollback strategy if something goes wrong?

## Step 3: Generate Migration Plan

Write to `docs/migrations/$migration-description.md`:

### Overview
- **Goal**: [what the end state looks like]
- **Current State**: [what exists today]
- **Estimated Effort**: [based on codebase analysis]
- **Risk Level**: High / Medium / Low

### Phase 1: Preparation (no behavior change)
- [ ] Set up parallel infrastructure/tooling
- [ ] Add compatibility layers / adapters
- [ ] Ensure test coverage of affected areas
- Rollback: None needed (no changes to production code)

### Phase 2: Incremental Migration
- [ ] Migrate [low-risk area first]
- [ ] Migrate [next area]
- [ ] ...
- Rollback: Revert to Phase 1 state

### Phase 3: Cleanup
- [ ] Remove compatibility layers
- [ ] Remove old code
- [ ] Update documentation
- Rollback: Keep compatibility layers, revert removals

### Dependencies
[Diagram of what must happen before what]

### Risks
| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|

### Verification per Phase
[What tests/checks confirm each phase succeeded]
```

---

## Summary: Complete Catalog

### Agents (7)
| Agent | Model | Tools | Memory | Purpose |
|-------|-------|-------|--------|---------|
| security-reviewer | inherit | Read, Grep, Glob, Bash | project | Security-focused code review |
| test-gap-analyzer | inherit | Read, Grep, Glob, Bash | - | Find untested code paths |
| codebase-explorer | haiku | Read, Grep, Glob, Bash | project | Rapid codebase understanding |
| performance-analyzer | inherit | Read, Grep, Glob, Bash | - | Performance analysis |
| accessibility-reviewer | inherit | Read, Grep, Glob, Bash | - | WCAG compliance review |
| documentation-auditor | haiku | Read, Grep, Glob, Bash | - | Find undocumented code |
| incident-investigator | inherit | Read, Grep, Glob, Bash | project | Production issue investigation |

### Skills (15)
| Skill | Category | Auto-invoke | Context | Purpose |
|-------|----------|-------------|---------|---------|
| /review-pr | Code Review | No | main | Comprehensive PR review |
| /review-quick | Code Review | Yes | main | Quick change review |
| /review-accessibility | Code Review | No | main | WCAG accessibility review |
| /refactor-safe | Refactoring | No | main | Test-validated refactoring |
| /add-tests | Testing | No | main | Generate tests for uncovered code |
| /write-adr | Documentation | No | main | Architecture Decision Records |
| /write-runbook | Documentation | No | main | Operational runbooks |
| /write-api-docs | Documentation | No | main | API documentation |
| /incident-debug | Incident Response | No | main | Systematic debugging |
| /onboard | Onboarding | No | fork | Codebase orientation |
| /upgrade-dep | Migration | No | main | Dependency upgrade |
| /migration-plan | Migration | No | main | Large-scale migration planning |
| /handoff-doc | Documentation | No | main | Client handoff documentation |
| /compliance-check | Review | No | fork | Compliance requirements verification |
| /estimate-effort | Planning | No | main | Effort estimation for tasks |

## Depth Checklist

- [x] Underlying mechanism explained (SKILL.md format, agent format, frontmatter fields, invocation patterns)
- [x] Key tradeoffs identified (agent vs skill, auto-invoke vs manual, context cost, model selection)
- [x] Compared to alternatives (Trail of Bits patterns, Security Phoenix graduated tiers, Cursor rules)
- [x] Failure modes described (context bloat from skill descriptions, misfiring auto-invoke, test failures during refactoring)
- [x] Concrete examples found (complete SKILL.md files, agent definitions, ready to embed in gdev)
- [x] Standalone-readable (each skill is copy-pasteable)
