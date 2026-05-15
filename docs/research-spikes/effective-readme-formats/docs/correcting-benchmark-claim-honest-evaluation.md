# I Corrected My Own Benchmark Claim from 91.5% to 88%. Here's What Changed.

- **Source URL**: https://dev.to/mohankrishnaalavala/i-corrected-my-own-benchmark-claim-from-915-to-88-heres-what-changed-3i1
- **Retrieved**: 2026-05-15

---

## The Problem

Mohan Krishna Alavala published an initial benchmark claiming "91.5% fewer tokens than code-review-graph" for his context-router project. Upon reflection, he realized the comparison was fundamentally flawed: the two tools had been tested on different repositories, different tasks, and different inputs. "Both numbers came from real benchmark runs. It was also wrong in every way that matters."

## The Solution: Workload-Matched Testing

The corrected v4.4.4 release implements a single methodological rule: both tools must receive identical inputs:

- Same Git commit SHAs
- Same diff as input
- Same machine and environment
- Same task type (code review)

This yielded a revised headline: approximately 88% token reduction with 2 out of 3 rank-1 hits versus 0 out of 3 for the competitor -- numbers the author will "defend."

## Key Findings

On three Kubernetes commits:
- **Rank-1 accuracy**: 2/3 for context-router; 0/3 for code-review-graph
- **Token efficiency**: 406 total tokens versus 3,478
- **Honest limitations**: N=3 sample is small; both tools were equally confused by fixture noise

## Critical Lessons

**1. Transparency over perfection**: Rather than hiding failed cases, the report details exactly where both tools stumbled and why, making it more credible than a perfect score.

**2. Scale matters**: The investigation uncovered a silent regression -- an FTS5 query bug affecting repositories with 10,000+ symbols. Smaller test suites never exposed this problem.

**3. Benchmarking standards**: "If you read a tool benchmark and can't tell whether both systems saw the same input, treat the result as marketing."

## Broader Implications

Developer tools are frequently compared using marketing-friendly rather than scientifically sound methods. The willingness to publicly correct and re-run benchmarks -- rather than quietly updating them -- sets a meaningful standard for open-source software credibility.
