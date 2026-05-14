# Hacker News: "We've moved our complete CI/CD pipeline to Nix"

- **Source**: https://news.ycombinator.com/item?id=36001690
- **Retrieved**: 2026-03-20

## Summary

Discussion thread about moving a complete CI/CD pipeline to Nix (Go, Rust, and other projects).

## Performance Data

**No specific CI/CD build time metrics or before/after measurements were shared in the comments.**

The discussion focuses on technical implementation details rather than performance outcomes:

- **dindresto** selected cargo2nix for Rust projects because it "allows incremental builds, meaning that dependency builds can be shared between projects" — no quantified improvement given
- **nix2container** chosen over built-in Docker tools to "improve layer caching" — no metrics provided

## Gaps

The thread lacks:
- Build time comparisons (pre-Nix vs. post-Nix)
- Caching hit rates or metrics
- Negative experiences or regressions
- Concrete timing measurements

The conversation remains theoretical rather than empirical regarding performance outcomes.
