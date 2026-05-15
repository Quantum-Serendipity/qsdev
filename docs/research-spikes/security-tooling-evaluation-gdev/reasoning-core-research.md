# reasoning-core (jakubkrzysztofsikora/reasoning-core) — Deep Dive Analysis

## Executive Summary

reasoning-core is a locally-run AI agent safety sidecar that scores code edits before LLM CLIs (Claude Code, Gemini, Copilot) execute them. It pairs a 130M-parameter Mamba state-space model with Tree-sitter AST parsing to compute structural risk vectors (cyclomatic complexity, fan-in/out, coupling, cohesion, churn, novelty) and gate edits that exceed per-file-kind thresholds. The system integrates with Claude Code via PreToolUse/PostToolUse hooks and an MCP server.

**Verdict for gdev integration**: Not recommended as a configuration option or default. Valuable as a concept/implementation source for specific patterns — particularly the hook architecture taxonomy, the multi-layer bypass threat model, and the shadow-mode calibration approach. The tool is too immature, too heavyweight, and too narrowly validated for production inclusion.

## 1. Architecture & Mechanisms

### System 1 + System 2 Design

reasoning-core implements a Kahneman-inspired dual-process architecture:

- **System 1 (Fast/Linguistic)**: The LLM (Claude, Gemini, etc.) proposes code edits as normal
- **System 2 (Deliberate/Structural)**: A local Python sidecar intercepts proposed edits via hooks, parses before/after ASTs using Tree-sitter, computes an 8-dimensional risk vector, and decides allow/block/warn

The key insight is scoring **deltas** rather than absolute complexity — a small safe edit to an inherently complex file should pass, while a structurally disruptive edit to a simple file should be flagged.

### Risk Vector (8 dimensions)

| Dimension | Source | What it measures |
|-----------|--------|-----------------|
| cyclomatic | AST branch counting | Complexity change |
| fan_in | Call graph in-degree | Dependency change |
| fan_out | Call graph out-degree | Coupling change |
| depth | DFS on call graph | Nesting/hierarchy change |
| churn | Line-set differencing | Volume of change |
| coupling | Edge count in call graph | Inter-module connectivity change |
| cohesion | Isolated-node ratio | Internal coherence change |
| novelty | Mamba embedding cosine distance | Semantic departure from project baseline |

Plus 3 session-level dimensions: session_centroid_drift, project_fan_in, project_coupling.

### Scoring Pipeline

1. Read file from disk (before state)
2. Apply proposed edit in-memory (after state)
3. Parse both with Tree-sitter, build call graphs
4. Compute risk vector deltas
5. Generate Mamba embeddings for before/after, compute AIS (architectural impact score) via cosine similarity and coherence_delta via chord distance
6. Compare against per-file-kind thresholds (source_code, test_code, plan_md, doc_md, config)
7. Flag regression if AIS below threshold, coherence_delta exceeds bounds, or individual risk dimensions breach ceilings
8. Return allow/block/warn with explanation

### Hook Layer Architecture (9 layers)

| Layer | Hook Point | Purpose |
|-------|-----------|---------|
| L1-L2 | PreToolUse (Edit/Write/MultiEdit) | SSM scoring, per-kind thresholds |
| L3-L4 | PreToolUse (Write for plans, Task) | Plan quality screening, subagent controls |
| L5-L6 | PreToolUse (Bash), PostToolUse (Edit) | Bash command safety, language fingerprinting |
| L7 | SessionStart | Session manifest and baseline registration |
| L8 | PreCompact | Context compaction guard |
| L9 | UserPromptSubmit + SessionStart | Session resume injection |

### Infrastructure

- FastAPI sidecar on localhost:8765 (loopback-only)
- ~250MB Mamba checkpoint download required
- Python 3.11+ with torch, transformers, tree-sitter, and 12+ tree-sitter grammar wheels
- MCP server for Copilot/Vibe integration (hybrid-reasoner)
- CLI tool (`rc`) for operator status, audit, and bypass management

## 2. What Problem Does It Solve?

reasoning-core addresses **structural regression during AI-assisted coding** — the tendency for LLM agents to make edits that individually look correct but cumulatively degrade codebase architecture. Specific problems:

1. **Plan drift**: Agent wanders from the agreed implementation plan
2. **Architectural degradation**: Increasing complexity, coupling, reduced cohesion
3. **Token waste**: Agent retries and backs out of bad edits, consuming tokens
4. **Scope creep**: Agent modifies files outside the expected boundary
5. **Self-modification**: Agent disabling its own safety guardrails

The system claims 8.2% average token savings and 25% cost reduction on its evaluation suite (8 tasks, single codebase).

## 3. Maturity Assessment

### Critical Maturity Concerns

| Factor | Assessment | Detail |
|--------|-----------|--------|
| **Age** | 15 days old | Created 2026-04-30, 15 days before this evaluation |
| **Contributors** | 1 person | 165 commits from jakubkrzysztofsikora, 1 from mistral-vibe |
| **Stars** | 5 | Negligible community adoption |
| **Forks** | 0 | No external use |
| **Version** | 0.1.0 | Pre-release |
| **Evaluation** | Smoke test only | Only stub-mode eval completed (all zeros by design). No live Claude evaluation has been run. The headline "29% fewer tokens" claim appears in marketing but is not substantiated by the published eval data. |
| **Language support** | 5 code + 8 data | Full call-graph analysis for Python, JS, TS, C#, SQL only |
| **Platform** | macOS arm64 primary | requirements.txt comments say "macOS arm64 + Python 3.11+". No Linux/NixOS testing evidence. |

### Red Flags

1. **Unsubstantiated claims**: The README claims "up to ~29% fewer tokens" and "6 of 8 tasks won", but the only published eval (smoke-001) ran in stub mode producing all-zero deltas with verdict "inconclusive". There is no published live evaluation.

2. **Single-person project**: 165 commits from one author in 15 days suggests heavy AI-assisted development. The second "contributor" (mistral-vibe) appears to be an automated LLM commit.

3. **Heavy dependency chain**: PyTorch, transformers, accelerate, 12+ tree-sitter grammars, FastAPI, MCP SDK — this is a substantial Python ML stack that conflicts with gdev's "single binary zero prerequisites" principle.

4. **5-second latency**: p95 of ~5 seconds on CPU for each Mamba forward pass. CUDA/MLX optimization is listed as future work. This adds significant friction to every edit.

5. **98-second overhead per run**: The README acknowledges "~98 seconds overhead per run (planning before editing)".

6. **Fail-open default**: Shadow mode is default, meaning the system logs but does not actually block anything out of the box. This is sensible for calibration but means it provides no security value until manually promoted to enforcement mode.

## 4. Integration Fit Assessment for gdev

### Option A: As a Configuration Option (user enables it) — NOT RECOMMENDED

**Against:**
- Violates "single binary zero prerequisites" — requires Python 3.11+, PyTorch (~2GB), Mamba checkpoint (~250MB), 12+ tree-sitter wheels
- gdev is a Go CLI; bundling or orchestrating a Python ML sidecar contradicts the architecture
- Only 5 code languages supported vs. gdev's 27 ecosystems — most gdev users would get embedding-only scoring (no call graphs)
- 5-second latency per edit is hostile to developer experience
- No published evaluation data supports the claimed benefits
- 15-day-old project with 1 contributor — unacceptable supply chain risk for a security tool
- NixOS compatibility untested (requirements.txt targets macOS arm64)
- The fail-open/shadow-mode default means it provides no value without manual calibration per project

**For:**
- Conceptually interesting as a "second opinion" on code quality
- MIT licensed, no blocking IP concerns

### Option B: As a Default (always included) — STRONGLY NOT RECOMMENDED

Everything from Option A applies, plus:
- Mandatory ~2.5GB of Python ML dependencies for a Go CLI tool is absurd
- Forces users to run a Python sidecar process they may not want
- Default shadow mode provides literally zero security value

### Option C: As Concept/Implementation Inspiration — SELECTIVELY RECOMMENDED

Several patterns from reasoning-core are worth studying for gdev's own hook architecture:

#### Worth Borrowing

1. **Hook layer taxonomy (L1-L9)**: reasoning-core's systematic classification of hook layers by concern (edit gating, bash safety, plan screening, subagent controls, session state) maps well to gdev Phase 32's managed hook policy. gdev already has destructive-prevention and credential-scan hooks; the taxonomy could inform future hook additions.

2. **Bypass threat model**: The 6-vector threat model (bash writes, sidecar kill, settings manipulation, hook modification, failure scenarios, subagent exploitation) is excellent threat modeling. gdev Phase 32 covers destructive commands and credential scanning but does not explicitly model the "agent disabling its own guardrails" attack vector. The guard-file locking and sidecar revival patterns are worth adapting.

3. **Shadow mode calibration**: Deploying hooks in logging-only mode before enforcement is a good operational pattern. gdev could offer `--shadow` mode for Phase 32 hooks, letting teams calibrate before enforcement. This reduces the "hooks that break my workflow" rejection risk.

4. **Per-file-kind thresholds**: Different enforcement rules for source code vs. test code vs. config vs. documentation. gdev's hook architecture could adopt this — a destructive-prevention hook might be stricter for production configs than for test fixtures.

5. **Audit trail design**: JSONL append-only logs with decision metadata (allowed/blocked/shadow_blocked, latency, risk vectors) per session. gdev Phase 32 already plans JSONL audit trails; reasoning-core's schema offers a mature reference.

6. **Escape hatch patterns**: Magic comments (`# rc:bypass-next`), CLI bypass (`rc bypass-next`), per-session overrides, kill switches. gdev needs similar escape hatches for its managed hooks — the `GDEV_HOOK_BYPASS` approach could mirror this.

#### Not Worth Borrowing

1. **Mamba SSM scoring**: The ML-based edit scoring is the core novelty but also the core risk. It adds massive dependency weight, 5-second latency, and has no validated effectiveness. gdev's security model is built on deterministic rules (deny patterns, vulnerability scanning, lockfile enforcement), not probabilistic ML scoring.

2. **MCP server for edit validation**: gdev already uses MCP for specific tools (Socket.dev). Adding an MCP-based edit validator would duplicate the hook-based approach that Claude Code natively supports and that gdev Phase 4/32 already implement.

3. **Call-graph analysis for gating**: Interesting in theory but limited to 5 languages, and gdev covers 27 ecosystems. Building partial call-graph analysis that only works for some users is worse than no call-graph analysis.

## 5. Comparison to Alternatives

### vs. gdev's Existing Hook Architecture (Phase 4 + Phase 32)

gdev already implements the valuable parts of reasoning-core's approach through a different mechanism:

| Capability | reasoning-core | gdev (planned) |
|-----------|---------------|----------------|
| Edit gating | ML-based (Mamba SSM + AST) | Rule-based (deny patterns, credential scan) |
| Bash safety | Regex pattern matching | Regex pattern matching (same approach) |
| Subagent controls | Regex screening of Task prompts | Not explicitly planned (gap worth addressing) |
| Session state | Manifest + resume injection | Not explicitly planned |
| Audit trail | JSONL per-session | JSONL per-session (Phase 32) |
| Escape hatches | Magic comments, CLI bypass | Not yet designed (Phase 32 gap) |
| Dependencies | Python + PyTorch + tree-sitter | Shell scripts (zero extra deps) |
| Latency | 5s per edit (CPU) | <50ms target (Phase 32) |

### vs. Pre-commit Hooks

Pre-commit hooks run after the agent finishes, before commit. reasoning-core runs during editing, intercepting each tool use. gdev Phase 5 already deploys pre-commit hooks (ripsecrets, lockfile audit). The in-flight interception is reasoning-core's differentiator, but gdev Phase 32 already achieves this via Claude Code's PreToolUse hooks without the ML overhead.

### vs. Claude Code Native Permissions

Claude Code's built-in permission system (allow/deny/ask rules in settings.json) provides deterministic, zero-latency gating. gdev Phase 4 generates these. reasoning-core adds probabilistic scoring on top, which could theoretically catch subtler issues but at 5s latency and unproven effectiveness.

### vs. Semgrep/CodeQL (Static Analysis)

Traditional static analysis tools detect known vulnerability patterns in code. reasoning-core detects structural regression (complexity increase, coupling increase). These are orthogonal concerns. gdev could integrate Semgrep (via pre-commit or hook) for vulnerability detection without reasoning-core's ML overhead.

## 6. Failure Modes & Limitations

1. **False positives on refactoring**: Any edit that intentionally changes architecture (splitting modules, extracting functions) will trigger high risk vector scores. The per-file-kind thresholds help but don't solve this.

2. **Fail-open by default**: Shadow mode means the system provides no protection until manually promoted. Users who forget to switch to enforcement get false sense of security.

3. **Cold start**: ~250MB model download + first inference warmup. Not suitable for CI/CD or ephemeral environments.

4. **Language coverage gaps**: Only 5 languages get call-graph analysis. Go, Rust, Java, Ruby, PHP, and 15+ other gdev ecosystems get embedding-only scoring, which is the weakest signal.

5. **Single-codebase validation**: All evaluation data is from one project. Cross-project generalization is unknown.

6. **Threshold calibration burden**: Per-file-kind thresholds need tuning per project. No published calibration guidance exists.

7. **Supply chain risk**: The tool itself downloads ML models from Hugging Face at runtime. For a security tool, this is an ironic attack surface (model poisoning, dependency confusion).

## 7. Recommendations for gdev

### Do Not Integrate (as tool or dependency)

reasoning-core is too immature (15 days, 1 contributor, no validated eval), too heavyweight (PyTorch + ML model), and too narrowly scoped (5 languages) for inclusion in gdev's toolchain. It violates gdev's "single binary zero prerequisites" and "curate don't reinvent" principles.

### Borrow These Patterns

1. **Add subagent guard to Phase 32**: gdev's hook architecture does not explicitly guard against the Task tool being used to spawn subagents that bypass parent hooks. Adapt reasoning-core's L4 regex screening concept — a PreToolUse hook on the Task matcher that screens prompts for mutation verbs targeting protected paths.

2. **Design escape hatches for Phase 32 hooks**: reasoning-core's magic comment, CLI bypass, and per-session override patterns should inform gdev's hook bypass design. `gdev hook bypass-next` or `GDEV_HOOK_BYPASS=1` environment variable for controlled overrides.

3. **Add shadow mode to Phase 32 hooks**: Offer `--shadow` mode for all managed hooks, logging what would have been blocked without actually blocking. This enables calibration and reduces adoption friction.

4. **Reference the bypass threat model**: Document gdev's equivalent of reasoning-core's 6-vector threat model. Particularly: can Claude modify `.claude/settings.json` to remove hooks? Can it `pkill` guard processes? Can it use bash to write files that bypass Edit/Write hooks? gdev Phase 32 partially addresses these but should explicitly enumerate and mitigate each vector.

5. **Audit log schema reference**: Use reasoning-core's JSONL audit schema as a starting point for gdev Phase 32's session audit trail — decision type, latency, matched pattern, tool input hash, and bypass reason are good fields to include.

## Sources

All source documents saved to `docs/`:
- `reasoning-core-readme.md` — Project README with architecture overview and claims
- `reasoning-core-architecture.md` — ARCHITECTURE.md with design details
- `reasoning-core-s2-core.md` — Core scoring engine source analysis
- `reasoning-core-ssm-backbone.md` — ML backbone and embedder loader analysis
- `reasoning-core-claude-settings.md` — Hook configuration (settings.json)
- `reasoning-core-hardening.md` — Bypass threat model and defense layers
- `reasoning-core-eval-results.md` — Evaluation results (smoke test only)
- `reasoning-core-pyproject-toml.md` — Build configuration and dependencies
- `reasoning-core-requirements.md` — Full dependency list
- `reasoning-core-pre-edit-guard.md` — Edit gating hook implementation
- `reasoning-core-pre-bash-guard.md` — Bash command safety hook implementation
- `reasoning-core-rc-cli.md` — Operator CLI tool
- `reasoning-core-enable-in-repo.md` — Repository enablement script
- `reasoning-core-commit-history.md` — Commit history and contributor stats
