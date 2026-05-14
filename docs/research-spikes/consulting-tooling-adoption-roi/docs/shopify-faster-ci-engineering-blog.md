# Keeping Developers Happy with a Fast CI — Shopify Engineering

- **Source**: https://shopify.engineering/faster-shopify-ci
- **Retrieved**: 2026-03-20

## Overall Achievement
Test Infrastructure team reduced p95 CI build time from **45 minutes to 18 minutes** (60% reduction).

## Key Performance Metrics

### Docker Container Start Time
- Before: 90 seconds (p95), sometimes up to 2 minutes
- After: 25 seconds (p95)
- Improvement: ~3.6x faster

### Dependency Building (Rails Monolith)
- Before: ~5 minutes
- After: ~3 minutes
- Improvement: 40% reduction

### Test Selection Optimization
- Builds avoiding full test suite: 45% -> 60%+
- Test stability improvement: 88% -> 97%
- p95 reduction from this change: 10 minutes (44 min -> 34 min)

## Three Focus Areas (% of CI time)
1. **Preparing Agents** (31% of CI time) — disk I/O bottlenecks, increased disk size and write speed
2. **Building Dependencies** (37% of CI time) — MD5 hash caching for database/assets, parallelized dependency steps
3. **Running Tests** — selective test running based on code changes, 170,000+ tests in Shopify Core

## Nix Relevance
**No Nix-related improvements mentioned.** This article is about Shopify's conventional CI optimization (Docker, caching, test selection) — NOT about their Nix adoption. Their Nix adoption is covered in separate NixCon talks and focuses on dev environments, not CI pipeline optimization.
