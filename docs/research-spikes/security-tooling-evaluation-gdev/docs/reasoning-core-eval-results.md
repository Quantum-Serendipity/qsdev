# reasoning-core EVAL_RESULTS.md (smoke-001)

- **Source**: https://github.com/jakubkrzysztofsikora/reasoning-core/blob/main/docs/EVAL_RESULTS.md
- **Retrieved**: 2026-05-15
- **Note**: Content returned via WebFetch AI summary — may not be verbatim

---

## Overview
First end-to-end exercise of the reasoning-core evaluation toolkit. Status: toolkit functions properly, though signal from live Claude testing remains pending.

## Summary Table
| Field | Value |
|---|---|
| run_id | smoke-001 |
| commit | 503f7e1 |
| invocation | RC_LIVE=1 RC_EVAL_STUB_CLAUDE=1 python3 eval/run_suite.py --n 2 |
| tasks | 2 (psf__requests-2317, django__django-13710) |
| arms | vanilla, treatment |
| pairs completed | 2 / 2 (100%) |
| errored | 0 |
| timed out | 0 |
| total runs | 4 |
| audit events captured | 311 |
| verdict | inconclusive (stub mode — by design) |
| wall clock | ~75 s |

## Headline Metrics Table
| Metric | n | Mean Delta (treatment - vanilla) | 95% CI | p (Wilcoxon) | p (Holm) |
|---|---:|---:|---|---:|---:|
| resolved_rate | 2 | 0.0000 | [0.000, 0.000] | nan | nan |
| regression_rate | 2 | 0.0000 | [0.000, 0.000] | nan | nan |
| ast_edit_distance | 2 | 0.0000 | [0.000, 0.000] | nan | nan |
| cyclomatic_delta | 2 | 0.0000 | [0.000, 0.000] | nan | nan |
| fan_in_delta | 2 | 0.0000 | [0.000, 0.000] | nan | nan |
| fan_out_delta | 2 | 0.0000 | [0.000, 0.000] | nan | nan |
| wall_clock_s | 2 | -0.50 | [-1.000, -0.500] | 1.000 | 1.000 |
| tokens_in | 2 | 0.0000 | [0.000, 0.000] | nan | nan |
| tokens_out | 2 | 0.0000 | [0.000, 0.000] | nan | nan |
| novelty_drift | 2 | 0.0000 | [0.000, 0.000] | nan | nan |

The stub implementation returns identical recorded patches for both arms, producing zero deltas by design.

## Decision Criteria Results
| Criterion | Result |
|---|---|
| regression_rate_drop_geq_0.15_holm_p_lt_0.05 | FAIL (no signal) |
| resolved_rate_no_worse_than_-5pp | PASS |
| latency_ratio_leq_1.5 | PASS |
| **verdict** | **inconclusive** |

This represents expected stub-mode behavior. Genuine signal requires running with `RC_LIVE=1` while disabling stub mode.

## Validation Checkpoints

Confirmed working: dataset loader, per-arm settings.json replacement, per-task JSON output, audit event correlation, report generation (markdown + JSON), decision logic.

Unvalidated: actual Claude performance differential, Mamba SSM scoring effectiveness, hook performance overhead, CI runner performance on full cold-cache scenarios.

## Next Steps
| Phase | Est. Duration | Cost |
|---|---|---|
| Live Claude eval (n=2 paired) | ~25 min | ~$11 |
| Stub eval n=5 via GH Actions | ~30-45 min | ~$26 |
| Full eval per design spec (n=100) | ~9 hours | ~$528 |
