# Cloudberry Engineering: Automating Code Security Reviews — Deep Dive Analysis

## Source

- **Article**: [Automating Code Security Reviews](https://cloudberry.engineering/article/automating-code-security-reviews/)
- **Published**: 2026-05-14 (originally on Synthesia's blog)
- **Saved to**: `docs/cloudberry-automating-code-security-reviews.md`

---

## 1. Core Concepts & Architecture

### The Problem

Naive AI security review ("pipe a diff into Claude, ask for security issues") produces two failure modes that kill adoption:

1. **False positives** — generic OWASP checklist output erodes engineer trust, causing teams to ignore findings
2. **Run-to-run variance** — non-deterministic output makes tuning and benchmarking impossible

### The Three-Pillar Solution

**Pillar 1: Attack Surface Mapping (Deterministic)**

Uses Semgrep rules to enumerate entry points (HTTP handlers, GraphQL resolvers, websocket endpoints, CLI commands, queue consumers), then deploys cheap Haiku subagents as "cartographers" to trace call graphs from entry points to sinks (database, shell, filesystem, network, template). The output is a "flat, factual map" — deterministic rather than judgment-dependent. This avoids full taint analysis (too complex to set up per-project) while still grounding the review in actual code paths.

**Pillar 2: Dedicated Security Context (Separated)**

Maintains separate `.security/architecture.md` and `.security/attack-surface.md` files distinct from codegen context (CLAUDE.md). Codegen context says "trust existing abstractions"; security review context says "distrust everything." Security context includes tenant models, isolation boundaries, blessed authorization primitives, ID risk taxonomy, anti-false-positive patterns, and historical vulnerability classes. This separation is cited as the key factor in reducing both variance and false positives.

**Pillar 3: Six-Phase Validation Pipeline**

```
PREP → MAP → HUNT → DEDUP → VALIDATE → AGGREGATE
main    haiku   opus    sonnet   opus       main
```

- **Prep**: Scope resolution, language detection, architecture summary
- **Map**: Semgrep entry point enumeration + parallel Haiku cartographer agents
- **Hunt**: Three parallel Opus "hunter" subagents search for injection, authorization, and business logic vulnerabilities using the attack surface map
- **Dedup**: Single Sonnet agent merges findings with identical root causes (cheap model, deterministic task)
- **Validate**: One Opus subagent per deduplicated finding re-traces code, checks mitigations, assesses exploitability, classifies as false positive or confirmed
- **Aggregate**: Discard false positives and low-impact findings; rank remainder; produce final report

### Key Design Choices

- **Shared context files on disk** rather than inlined context in prompts — maintains paper trail, keeps prompts lean
- **Right-sized models**: Opus for judgment (hunting, validation), Haiku for deterministic mapping, Sonnet for deduplication
- **~60% of raw findings discarded** through dedup + validation — the pipeline's value is noise reduction, not finding generation

### Results

| Metric | Value |
|---|---|
| Average cost per review | $3.88 |
| Average duration | 9.4 minutes |
| Average valid findings | 1.566 |
| Raw findings discarded | ~60% |
| Cost per actionable finding (large PRs) | ~$2.70 |
| Critical finding frequency | ~1 per 30 reviews |

### Deployment Strategy

Self-serve adoption was low. Moving to CI/CD as non-blocking PR comments solved adoption. For merged PRs with identified issues, a follow-up agentic system opens fix proposals — treating post-merge patching as the honest approach given review latency (~9.4 min).

---

## 2. Mapping to gdev Architecture

### 2a. What gdev Already Has (Overlap)

gdev's existing security architecture already covers several of Cloudberry's concerns, but through different mechanisms:

| Cloudberry Concept | gdev Equivalent | Coverage |
|---|---|---|
| Dedicated security context | `.claude/rules/security-rules.md` + CLAUDE.md security section (Phase 4) | **Partial** — gdev separates security rules from codegen, but doesn't maintain attack surface maps or anti-false-positive patterns |
| Entry point enumeration via Semgrep | Phase 5 Unit 4.4 references Semgrep in CI workflows; Phase 12 enables Semgrep as a toggleable tool | **Partial** — Semgrep is available but not used for attack surface mapping |
| Multi-agent pipeline | Phase 14 Unit 14.2 deploys `security-reviewer` agent | **Minimal** — single agent, no multi-phase pipeline, no dedup/validation stages |
| Non-blocking CI integration | Phase 5 Unit 4.4 generates CI vulnerability scanning workflows | **Different focus** — gdev CI scans dependencies (OSV, Socket.dev), not code-level security reviews |
| Right-sized model selection | Phase 14 Unit 14.4 has model-aware context budgets (Sonnet vs Opus) | **Architectural match** — gdev already thinks in terms of model-appropriate task assignment |
| False positive reduction | Not explicitly addressed in current gdev phases | **Gap** |

### 2b. Integration Assessment

#### Option A: Configuration Option (User Enables It)

**What it would look like**: `gdev enable cloudberry-security-review` or a wizard toggle. gdev generates the `.security/` directory with project-specific context files, deploys a multi-phase security review skill that orchestrates the pipeline, and configures Semgrep rules for entry point detection.

**Feasibility**: High. This fits gdev's "every tool is individually toggleable" principle and the Phase 12 lifecycle system. The `.security/` directory and its contents are project-specific and user-maintained (like CLAUDE.md's user section).

**Value**: High for teams with significant security review needs. The pipeline approach (map → hunt → dedup → validate) is materially better than the current single-pass `security-reviewer` agent in Phase 14.

**Complexity**: Medium. Requires:
- Template for `.security/architecture.md` with guided prompts
- Template for `.security/attack-surface.md` (initially empty, populated by first run)
- A multi-phase orchestration skill (or agent chain)
- Semgrep rule sets for entry point detection per ecosystem
- Model selection logic (which phases use which model)

**Recommendation**: **Yes, as an opt-in configuration option.** This is the strongest integration path. gdev already has the ecosystem detection, Semgrep integration, and model-aware generation to support this. The `.security/` directory pattern is a natural extension of gdev's file generation model.

#### Option B: Default Pattern (Always Included)

**What it would look like**: Every `gdev init` generates `.security/` templates and the multi-phase review pipeline is the default `security-reviewer` agent.

**Feasibility**: Low. The pipeline requires Semgrep (not always installed), costs ~$3.88/run (not acceptable as an always-on default), and requires project-specific security context that can't be auto-generated.

**Recommendation**: **No.** The cost, setup burden, and complexity make this unsuitable as an always-on default. It violates gdev's "zero prerequisites" principle (Semgrep dependency) and would generate noise for projects without meaningful security context.

#### Option C: Concept/Implementation Inspiration (Borrow Patterns)

Several patterns from the article are directly borrowable regardless of whether the full pipeline is implemented:

1. **Separated security context pattern** — Already partially present in gdev. Enhancement: generate a `.security/` or `.claude/security/` directory with `architecture.md` and `known-patterns.md` templates during `gdev init`, alongside the existing security rules. This is cheap to implement and immediately improves the existing `security-reviewer` agent.

2. **Anti-false-positive patterns** — The article's insight that security context should include "what NOT to flag" is missing from gdev's current security rules. Enhancement: add an anti-false-positive section to `security-rules.md` template (e.g., "This project uses parameterized queries via [ORM] — do not flag SQL injection in repository layer" or "Auth is handled by [middleware] — do not flag missing auth checks in route handlers").

3. **Dedup-before-validate principle** — The pipeline spends cheap compute (Sonnet) to deduplicate before expensive compute (Opus validation). This principle applies to any gdev agent that produces findings lists. Enhancement: add a dedup instruction block to the `security-reviewer` agent prompt.

4. **Right-sized model assignment** — gdev already does this (Phase 14.4). The article validates the approach with concrete cost data.

5. **Shared context files on disk vs. inlined context** — gdev already does this (`.claude/rules/`, `@`-imports). The article validates the approach.

6. **Benchmarking protocol** — "No dimension is allowed to regress" across cost, duration, findings, and variance. This is a process principle gdev could document in its security review skill as a self-evaluation checklist.

**Recommendation**: **Yes, borrow patterns 1-3 and 6.** These are low-cost, high-value improvements to gdev's existing security infrastructure.

---

## 3. Detailed Integration Recommendations

### Recommendation 1: Security Context Directory (Borrow + Config Option)

**As a concept borrow (low cost, immediate):**
- Add `.security/architecture.md` template to the claudecode addon's embedded file library
- Template contains guided prompts: "Describe your tenant model:", "List authorization primitives:", "Describe data classification:", "List known vulnerability patterns:"
- Generated during `gdev init` when Claude Code addon is enabled, as an empty-but-structured file
- Referenced via `@.security/architecture.md` in the `security-reviewer` agent's prompt
- This is a documentation file that improves all downstream security review quality

**As a configuration option (medium cost, Phase 14+):**
- `gdev enable security-context` generates the `.security/` directory
- Includes `architecture.md`, `attack-surface.md`, `known-patterns.md`
- The `security-reviewer` agent is upgraded to read these files
- First invocation offers to help populate the files interactively

### Recommendation 2: Multi-Phase Security Review Skill (Config Option)

**As a new skill in Phase 14's catalog:**
- `/security-review-deep` — orchestrates a Cloudberry-style pipeline
- Requires: Semgrep installed, `.security/architecture.md` populated
- Steps: (1) Semgrep entry point scan, (2) call graph tracing via subagent, (3) parallel hunt with specialized prompts (injection, auth, business logic), (4) dedup, (5) validate, (6) aggregate
- Gate: `gdev enable deep-security-review` adds Semgrep to prerequisites
- Cost warning in skill description: "~$3-5 per review, recommended for pre-release or high-risk changes"

**Tradeoff**: This is significantly more complex than the current single-pass `security-reviewer` agent. The value is proportional to codebase complexity — for small projects or low-risk changes, the existing agent is sufficient. The deep review is for CI integration on high-value repos.

### Recommendation 3: Anti-False-Positive Patterns in Security Rules (Concept Borrow)

**Enhance the existing `security-rules.md` template (Phase 4 Unit 3.4):**

Add a section:
```markdown
## Known Safe Patterns (Anti-False-Positives)
<!-- Add patterns that security reviewers should NOT flag -->
<!-- Examples: -->
<!-- - SQL queries use parameterized queries via [ORM name] — do not flag SQL injection in the data layer -->
<!-- - Authentication is handled by [middleware name] at the router level — do not flag missing auth in handlers -->
<!-- - User input is sanitized by [library name] before rendering — do not flag XSS in template files -->
```

This is zero-cost (template text) and immediately improves review quality by reducing noise.

### Recommendation 4: Benchmarking Protocol for Security Reviews (Concept Borrow)

**Add to the `security-reviewer` agent or `/security-review` skill:**

A self-evaluation block that tracks:
- Number of findings produced
- Number classified as false positive (by user feedback)
- Review duration
- Model used

This creates a feedback loop for tuning. gdev could store this in `.security/review-metrics.jsonl` (append-only log), enabling teams to track their false positive rate over time.

---

## 4. Tradeoffs, Limitations, and Failure Modes

### Tradeoffs

| Factor | Cloudberry Approach | Impact on gdev |
|---|---|---|
| **Cost** | $3.88/review average | Significant for CI on every PR. Must be opt-in, not default. |
| **Latency** | 9.4 min average | Too slow for blocking CI. Non-blocking comments are the right deployment. |
| **Setup burden** | Requires `.security/` context files to be populated | Effective reviews require project-specific context that can't be auto-generated. |
| **Semgrep dependency** | Required for entry point mapping | Adds a prerequisite. gdev principle: "zero prerequisites." Must be gated behind the feature toggle. |
| **Model dependency** | Assumes access to Opus, Sonnet, and Haiku | Not all teams have access to all model tiers. Need fallback behavior. |

### Limitations

1. **Security context is manually maintained** — if `.security/architecture.md` drifts from reality, reviews produce false positives from stale context. No automatic staleness detection.
2. **Entry point detection is language-dependent** — Semgrep rules must exist for each ecosystem's entry point patterns. Cloudberry built rules for their stack; gdev would need rules for 27 ecosystems (or start with Tier 1 only).
3. **Pipeline orchestration complexity** — Six phases with different models, parallel execution, and inter-phase data passing is complex to implement as a Claude Code skill. The current skill/agent system doesn't natively support multi-phase orchestration with model switching.
4. **Business logic vulnerabilities** — The article acknowledges that the "business logic" hunter is the weakest link. AI struggles with domain-specific logic vulnerabilities without deep context.
5. **Cost unpredictability** — While average is $3.88, variance likely correlates with PR size and codebase complexity. Large refactors could cost significantly more.

### Failure Modes

1. **Empty security context** — If `.security/architecture.md` is never populated, the pipeline falls back to generic review (the exact problem it's designed to solve). Mitigation: gdev skill prompts for context on first run.
2. **Semgrep rule gaps** — If entry point patterns don't cover the project's framework, the attack surface map is incomplete. Mitigation: allow custom Semgrep rules in `.security/semgrep-rules/`.
3. **Model tier unavailability** — If Opus isn't available, hunting and validation degrade. Mitigation: configurable model mapping with sensible fallbacks (Sonnet for all phases as minimum viable).
4. **Overfitting to Cloudberry's stack** — The pipeline was tuned for Synthesia's codebase. Different architectures (microservices vs monolith, different frameworks) may have different optimal configurations.
5. **Review latency causing ignore behavior** — 9.4 min is long. If engineers merge before review completes, findings arrive post-merge. The article addresses this with follow-up fix proposals, but this requires additional infrastructure.

---

## 5. Comparison to Alternatives

### vs. gdev's Current `security-reviewer` Agent (Phase 14)

| Dimension | Current Agent | Cloudberry Pipeline |
|---|---|---|
| Architecture | Single-pass, single agent | Six-phase, multi-model pipeline |
| False positive handling | None — relies on agent judgment | Explicit dedup + validation stages |
| Context | Reads changed files, generic security prompt | Attack surface map + project-specific security context |
| Cost | ~$0.10-0.50 (single agent invocation) | ~$3.88 average |
| Setup | Zero (deployed by gdev) | Requires `.security/` context + Semgrep |
| Finding quality | Generic, OWASP-checklist-level | Project-specific, map-grounded |

**Assessment**: The current agent is a reasonable default (cheap, zero-setup). The Cloudberry pipeline is a premium option for teams that need higher-quality reviews. They serve different points on the cost/quality spectrum. Both should exist in gdev.

### vs. CodeQL / GitHub Advanced Security

| Dimension | Cloudberry Pipeline | CodeQL |
|---|---|---|
| Type | AI agent pipeline | Static analysis engine |
| Setup | Moderate (security context files) | High (query packs, CI integration) |
| False positives | ~40% raw findings survive (60% discarded) | Varies by query pack, often high |
| Business logic | Attempts it (weakest area) | Cannot detect (outside scope) |
| Cost | ~$3.88/review | Free for public repos, GHAS license for private |
| Adaptability | Learns from security context files | Requires custom queries |

**Assessment**: Complementary, not competing. CodeQL/GHAS provides deterministic static analysis. The Cloudberry approach adds AI judgment for business logic and context-aware review. gdev could recommend both.

### vs. Semgrep Pro / Semgrep Supply Chain

| Dimension | Cloudberry Pipeline | Semgrep Pro |
|---|---|---|
| Type | AI-augmented review pipeline using Semgrep as a component | Rule-based static analysis platform |
| Role of Semgrep | Entry point detection only (component) | Full analysis engine |
| Business logic | Yes (AI hunting phase) | No (rules only) |
| Custom rules | Via `.security/` context | Via .semgrep/ rules |
| Cost | ~$3.88/review + Semgrep (free tier) | Semgrep Pro license |

**Assessment**: The Cloudberry approach uses Semgrep as a building block, not a replacement. gdev already plans Semgrep integration (Phase 12). The Cloudberry pattern extends Semgrep's value by using its output as input to AI-driven analysis.

---

## 6. Replicable Principles (Applicable Beyond Security Review)

The article surfaces five engineering principles that apply to any AI agent pipeline, not just security review:

1. **Encode deterministic steps deterministically** — Use tools (Semgrep) for what tools do well (pattern matching), AI for what AI does well (judgment). Don't ask AI to enumerate entry points when grep/Semgrep can do it deterministically.

2. **Split pipeline into narrow phases with model-appropriate sizing** — Cheap models for cheap tasks, expensive models for judgment tasks. This principle directly validates gdev's existing Phase 14.4 model-aware generation.

3. **Build dedup and validation before tuning generation** — Reducing false positives (output quality) is more valuable than increasing true positives (output quantity). This applies to any agent that produces lists of findings/recommendations.

4. **Separate adversarial and constructive context** — Security review requires distrust; code generation requires trust. These mindsets conflict. Separate their context/instructions.

5. **Benchmark ruthlessly** — Cost, duration, findings count, variance. No dimension regresses. This is a process discipline, not a technical feature.

---

## Depth Checklist

- [x] Underlying mechanism explained — six-phase pipeline, three-pillar architecture, model sizing rationale
- [x] Key tradeoffs and limitations identified — cost, latency, setup burden, context maintenance, model dependency
- [x] Compared to alternatives — vs. current gdev agent, vs. CodeQL, vs. Semgrep Pro
- [x] Failure modes described — empty context, Semgrep gaps, model unavailability, overfitting, latency-induced ignore
- [x] Concrete examples found — $3.88 cost, 9.4 min duration, 60% discard rate, CI deployment strategy
- [x] Report is standalone-readable — sufficient for integration decisions without reading the original article
