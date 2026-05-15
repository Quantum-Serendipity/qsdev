# reasoning-core s2_core.py (System 2 Sidecar Core)

- **Source**: https://github.com/jakubkrzysztofsikora/reasoning-core/blob/main/src/s2_core.py
- **Retrieved**: 2026-05-15
- **Note**: Content returned via WebFetch AI summary — may not be verbatim

---

## Core Components

### Data Structures
The file defines `ParseResult` and `ImpactReport` dataclasses. The latter tracks architectural impact scores, coherence deltas, risk vectors across 11 dimensions ("cyclomatic", "fan_in", "fan_out", "depth", "churn", "coupling", "cohesion", "novelty", "session_centroid_drift", "project_fan_in", "project_coupling"), and regression detection.

### Parsing & Graph Building
Functions like `parse_source()` leverage tree-sitter for multi-language AST parsing (Python, JavaScript, TypeScript, C#, SQL). The `build_call_graph()` function constructs call graphs using language-specific traversals — for example, Python looks for "call" nodes, while JavaScript searches "call_expression" nodes.

### Risk Quantification
The `_compute_risk_vector()` function measures deltas: cyclomatic complexity from branch counts, fan-in/out from graph degree analysis, structural depth via DFS, churn via line-set differences, coupling from edge counts, and cohesion from isolated-node ratios.

### Scoring Engine
`score_change()` compares before/after source via a Mamba-backed embedding backbone, computing architectural impact score (AIS) from cosine similarity and coherence_delta via chord distance on normalized embeddings. It flags regressions when AIS drops below threshold, coherence_delta exceeds bounds, or individual risk dimensions breach ceilings — with per-file-kind thresholds (source_code, test_code, plan_md, doc_md, config).

### HTTP Service
A FastAPI application exposes `/health`, `/score`, `/metrics`, and `/baseline` endpoints on loopback 127.0.0.1:8765, with latency ring buffering and session-based baseline registries for drift tracking.

### Key Insight
The system prioritizes *delta metrics* over absolute file complexity to avoid false positives on inherently complex files receiving small, safe edits.
