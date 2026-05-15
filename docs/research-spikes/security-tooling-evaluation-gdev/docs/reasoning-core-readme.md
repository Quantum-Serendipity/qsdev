# reasoning-core README

- **Source**: https://github.com/jakubkrzysztofsikora/reasoning-core/blob/main/README.md
- **Retrieved**: 2026-05-15
- **Note**: Content returned via WebFetch AI summary — may not be verbatim

---

## Overview
**reasoning-core** is a locally-run AI agent safety system that scores code edits before Claude (or other LLM CLIs) execute them. It uses a 130M-parameter Mamba SSM model running on your machine to detect structural regressions and plan drift.

## Key Features

### Performance Claims
- **"Up to ~29% fewer tokens per task"** on cache-heavy operations
- 8.2% average token savings across an 8-task evaluation
- Better plan quality (+0.32 on 1–5 scale) and implementation focus
- Agents stay within promised file boundaries (+0.23 improvement)

### Architecture: System 1 + System 2
The system pairs Claude's linguistic reasoning (System 1) with a specialized structural reasoner (System 2):
- **Fast stage:** LLM proposes an edit
- **Deliberate stage:** Local SSM scorer evaluates an 8-dimensional risk vector (cyclomatic complexity, fan-in/out, depth, churn, coupling, cohesion, novelty)
- **Gating:** Blocks edits exceeding per-file-kind thresholds; suggests repairs

## Installation (6 Steps)

1. Clone and create Python venv
2. Download Mamba checkpoint (~250MB)
3. Start the FastAPI sidecar
4. Activate direnv for repo-scoped environment
5. Launch Claude from the repo
6. (Optional) Promote globally across all projects

Default operates in **shadow mode** (logs decisions without blocking), allowing calibration before enforcement.

## Multi-CLI Support
- **Claude Code:** Native hook integration via `.claude/settings.json`
- **Gemini CLI:** Hook-compatible adapter layer
- **GitHub Copilot / Mistral Vibe:** MCP server approach with post-turn audit

## Hook Layers (9 Total)
| Layer | Purpose |
|-------|---------|
| L1–L2 | Source-code edit gating (SSM scoring, per-kind thresholds) |
| L3–L4 | Plan quality + task prompt screening |
| L5–L6 | Bash command safety + language fingerprinting |
| L7–L9 | Session state, compaction, resume injection |

## Risk Vector Scoring
Eight normalized dimensions drive decision-making:
- **Structural dims:** cyclomatic, fan_in/out, depth, coupling, cohesion
- **Semantic dims:** churn, novelty (via Mamba pooled embedding)

Block threshold per file kind (source_code, test_code, plan_md, doc_md, config).

## Evaluation Results (Iteration 1)
- **6 of 8 tasks won** against vanilla Claude (decision rule: gates → quality → cost)
- **−25% suite cost** ($91.50 → $68.51, token-normalized)
- **−23% wall clock** (feedback loops avoided)
- Implementation quality tied; plan quality initially declined (iter-2 fixes shipped)

## Calibration & Hardening
- **Mahalanobis gate** for multivariate outlier detection
- **Grounding eval:** Cohen κ ≥0.7 against cross-family judges (devstral, llama-3.3-70b, mistral-small)
- **Golden set:** Pinned regression cases
- **OOD detector:** Flags novel patterns not in training corpus

## Roadmap Highlights
- Shipped: Shadow-mode, plan-time scoring, mock detector, generative repair head, Mahalanobis calibration
- In progress: Synthesize-Check-Refine loop (auto-repair on block), hybrid symbolic rules
- Future: CUDA/MLX kernels (p95 ~5s → ~50ms), pre-commit integration, Mamba-3 watch

## Configuration
- **Shadow mode:** `RC_SHADOW_MODE=1` (default)
- **Thresholds:** `S2_AIS_THRESHOLD`, `S2_COHERENCE_THRESHOLD`, `S2_RISK_DIM_THRESHOLD`
- **Generative repair:** `RC_REASONER_BACKEND` (mlx/remote), Qwen/Scaleway API key support
- **Escape hatches:** Magic comments (`# rc:bypass-next`), `rc bypass-next` CLI, per-session overrides

## Notable Limitations
- **Cost:** ~98 seconds overhead per run (planning before editing)
- **Code readability:** No improvement (gate ensures conformance, not aesthetics)
- **Single codebase eval:** Results may vary across projects
- **CPU latency:** 5s p95 on Mamba forward pass (CUDA not yet integrated)

## License & Credits
MIT license. Built on Mamba (Gu & Dao), Tree-sitter, Model Context Protocol, and FastMCP.
