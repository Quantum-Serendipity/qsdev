# Docker Layer Caching in CI Pipelines Cut Build Times by 70% — Netdata

- **Source**: https://www.netdata.cloud/academy/docker-layer-caching/
- **Retrieved**: 2026-03-20

## Headline Claim
"Docker Layer Caching in CI Pipelines Cut Build Times by 70%" — this is the title metric.

## Supporting Data
- Implementing smart caching strategies can "slash your container build times, often by 70% or more"
- Subsequent CI runs after first build show "dramatic speed-up" with remote cache backends
- Multi-stage builds "dramatically reduce final docker image size"

## Limitations
Functions as a practical implementation guide rather than benchmarking report. Does not provide:
- Specific project measurements
- Application size comparisons
- Build environment specifications
- Quantified time reductions beyond the headline 70% claim

## Relevance
Docker layer caching claims similar percentage improvements (70%+) to what Nix ecosystem companies claim for Nix caching. This is important context: **conventional Docker pipelines with optimized caching can achieve comparable percentage reductions.** The comparison is not "Nix with cache vs. Docker without cache" but should be "Nix with cache vs. Docker with optimized caching."
