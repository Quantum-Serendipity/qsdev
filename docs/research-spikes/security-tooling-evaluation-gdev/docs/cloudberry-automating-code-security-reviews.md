<!-- Source: https://cloudberry.engineering/article/automating-code-security-reviews/ -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: Content returned via WebFetch AI summarization; may not be verbatim article text -->

# Automating Code Security Reviews

**Published:** May 14, 2026 | **Tags:** #ai #agents #security-reviews

---

## Overview

This article, originally published on Synthesia's blog, describes how Cloudberry Engineering built an autonomous agent system for conducting code security reviews. Rather than simply piping diffs into Claude, the team engineered a multi-phase pipeline that reduces false positives and improves consistency.

## The Core Problem

A basic approach—"pipe a diff into Claude, ask for security issues"—produces generic output and lacks codebase-specific knowledge. This generates noise: "a generic prompt produces generic output...you get an OWASP top-ten checklist applied to your code."

Two critical failure modes emerge:
- **False positives** that erode engineer trust
- **Run-to-run variance** that makes tuning impossible

## Three-Pillar Solution

### 1. Building an Attack Surface Map

Rather than implementing full taint analysis (too complex for self-serve), the team uses a two-step approach:

**Entry Point Enumeration:** Semgrep rules identify where untrusted input enters (HTTP handlers, GraphQL resolvers, websocket endpoints, CLI commands, queue consumers).

**Cartographer Phase:** Smaller Haiku subagents trace call graphs from each entry point to potential sinks (database, shell, filesystem, network, template), producing a "flat, factual map" without requiring extensive setup.

### 2. Dedicated Security Context

Separate `SECURITY.md` files from codegen context. Code-generation instructions tell agents to "trust existing abstractions"; security review requires the opposite mindset. Security context includes:
- Tenant models and isolation boundaries
- Blessed authorization primitives
- ID risk taxonomy
- Anti-false-positive patterns
- Historical vulnerability classes

This separation reduced variance and false positives significantly.

### 3. Six-Phase Validation Pipeline

```
┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌───────────┐
│  PREP    │──▶│   MAP    │──▶│   HUNT   │──▶│  DEDUP   │──▶│ VALIDATE │──▶│ AGGREGATE │
│  main    │   │  haiku   │   │  opus    │   │  sonnet  │   │  opus    │   │   main    │
└──────────┘   └──────────┘   └──────────┘   └──────────┘   └──────────┘   └───────────┘
```

**Step 1 (Prep):** Main agent resolves scope, detects language, summarizes architecture.

**Step 2 (Map):** Semgrep identifies entry points; parallel Haiku agents trace paths.

**Step 3 (Hunt):** Three parallel "Hunter" subagents search for injection, authorization, and business logic vulnerabilities using the attack surface map.

**Step 4 (Dedup):** Single Sonnet agent merges findings with identical root causes before expensive validation.

**Step 5 (Validate):** One subagent per deduplicated finding re-traces code, checks mitigations, assesses exploitability, and classifies as false positive or confirmed.

**Step 6 (Aggregate):** Discard false positives and low-impact findings; rank remainder and produce final report.

## Key Design Choices

**Shared Context Files:** Subagents read `.security/architecture.md` and `.security/attack-surface.md` themselves rather than receiving inlined context. This maintains a paper trail and keeps prompts lean.

**Right-Sized Models:** Expensive models (Opus) reserved for hunting and validation where judgment matters. Cheaper models (Haiku, Sonnet) for deterministic mapping and deduplication.

## Results

Across all review sizes:
- **Average cost:** $3.88 per review
- **Duration:** 9.4 minutes
- **Valid findings:** 1.566 average
- **Discarded findings:** ~60% of raw hunter output

**Cost per actionable finding:** roughly $2.70 for large reviews.

**Severity distribution:** Critical findings appear in roughly 1 of every 30 reviews; most output concentrated in medium and high severity bands.

## Deployment Strategy

Initially offered as self-serve tool, adoption was limited. Moved to CI/CD as non-blocking comments on pull requests. For merged PRs with identified issues, a follow-up agentic system opens fix proposals—treating post-merge patching as the honest approach given review latency.

## Replicable Principles

1. Encode attack surface deterministically using tech-stack-specific rules
2. Split pipeline into narrow phases; assign model size to task complexity, not vice versa
3. Build deduplication and validation before tuning hunters
4. Maintain separate security and codegen context layers
5. Benchmark every iteration on reference codebase: cost, duration, findings, variance

The authors emphasize rigorous measurement across multiple runs, stating: "No dimension is allowed to regress" without acknowledging vibes over evidence.
