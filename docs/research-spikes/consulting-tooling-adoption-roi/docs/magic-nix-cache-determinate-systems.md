# Introducing the Magic Nix Cache — Determinate Systems

- **Source**: https://determinate.systems/blog/magic-nix-cache/
- **Retrieved**: 2026-03-20

## Performance Claims

Primary claim: "we're confident that a wide variety of Nix projects can reduce Nix-related build times in Actions by 30-50%"

This is described as occurring "with no configuration changes beyond the single line of YAML."

## Absence of Benchmark Data

The article provides **no specific benchmark data, before/after comparisons, or concrete performance metrics**:
- No actual build time measurements
- No test case results
- No project-specific performance examples
- No graphs, charts, or detailed statistics
- No time savings in seconds or minutes

## Context

The performance improvement estimate is framed as a confidence statement rather than backed by published benchmarks. The author notes that caching "can cut build times by orders of magnitude (depending on the context)" but attributes this to Nix's general caching capabilities, not specifically to Magic Nix Cache performance data.

The article emphasizes ease of use and lack of setup requirements as primary benefits rather than quantified performance gains.

## Critical Note

The Garnix CI benchmarks (see garnix-nix-ci-benchmarks.md) found that magic-nix-cache provided "no apparent benefit" compared to parallel GitHub Actions without caching, directly contradicting this 30-50% claim. The Japanese binary cache comparison (see nix-binary-cache-tools-comparison-github-actions.md) similarly found magic-nix-cache-action performed at 100-148% of uncached baseline (i.e., no improvement or worse).
