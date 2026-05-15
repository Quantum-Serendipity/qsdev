# Sense bench/ Directory Listing
- **Source**: https://github.com/luuuc/sense/tree/main/bench
- **Retrieved**: 2026-05-15

---

## Directory Structure

```
bench/
├── improvement-loop/          # Self-tuning convergence-aware loop
├── lib/                        # Core scoring and evaluation modules
├── locked/                     # Immutable configuration files
├── results/                    # Output artifacts (reports, scores)
├── scenarios/                  # Test scenario definitions
│   └── held-out/              # Frozen test cases with reference grades
├── README.md
├── SCORING.md
├── bench.sh
├── end-goal.md
├── freeze-heldout.sh
├── judge.sh
├── report.sh
├── run.sh
└── score.sh
```

## Key Files

**Configuration & Docs:**
- README.md - Full usage guide
- SCORING.md - Formula and component definitions
- end-goal.md - Bench-readiness criteria
- locked/locked.yaml - Immutable axis weights and boundaries

**Execution Scripts:**
- bench.sh - Single wrapper (run -> score -> judge -> report)
- run.sh - Spawns Claude sessions, outputs transcript.json
- score.sh - Computes metrics, outputs scored.json
- judge.sh - LLM evaluation, outputs judged.json
- report.sh - Generates markdown/JSON/terminal reports
- freeze-heldout.sh - Locks held-out transcripts

**Python Libraries** (under lib/):
scorer, fairness, grounding, judge, reporter, audit modules, convergence evaluation, cost tracking, and validation tools

## Benchmark Scope

Six scenarios across repos: Flask, Gin, Axum, Discourse, Javalin, Next.js. Each runs 4 evaluation steps per tool, scored on "quality (55%), citation grounding (15%), efficiency (20%), keyword coverage (10%)."
