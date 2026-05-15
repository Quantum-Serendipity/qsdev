# reasoning-core ARCHITECTURE.md

- **Source**: https://github.com/jakubkrzysztofsikora/reasoning-core/blob/main/docs/ARCHITECTURE.md
- **Retrieved**: 2026-05-15
- **Note**: Content returned via WebFetch AI summary — may not be verbatim

---

## Core Design

The system implements a "two-brain" reasoning architecture inspired by Kahneman's fast/slow thinking model. **System 1** is a linguistic layer using Claude Code routed through Scaleway's Generative APIs (devstral-2-123b model), which generates code proposals. **System 2** is a mathematical layer — a local Python sidecar running a real Mamba state-space model via Tree-sitter AST parsing — that scores architectural impact and regression risk.

## Key Infrastructure

- **Port 8765**: S2 sidecar (HTTP, loopback-only)
- **Port 8787**: y-router for external API calls
- **Five hardening layers**: Pre-edit guards (L1), bash guards (L2), plan-document screening (L3), subagent controls (L4), and sidecar revival (L5)

## Language Support

Thirteen languages across two tiers: five "code languages" (Python, JavaScript, TypeScript, C#, SQL) with call-graph analysis, and eight "data languages" (Markdown, JSON, YAML, CSS, SCSS, HTML, Dockerfile, Vue) with embedding-only analysis.

## SSM Backbone

The design substitutes `state-spaces/mamba-130m-hf` (publicly available, Apache-2.0 licensed) because SlideMamba lacks a released checkpoint. This preserves architectural honesty — using a real pretrained SSM rather than mock weights — while remaining redistributable.

## Evaluation

The benchmark uses SWE-bench Verified (100 Python tasks, paired design) measuring regression rate reduction as the primary metric, with secondary metrics covering resolved rate, AST distance, complexity delta, and hook performance.
