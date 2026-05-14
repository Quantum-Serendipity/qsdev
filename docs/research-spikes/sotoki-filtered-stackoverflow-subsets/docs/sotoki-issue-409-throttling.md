---
source: https://github.com/openzim/sotoki/issues/409
retrieved: 2026-05-14
type: github-issue-with-comments
---

# Issue #409: Caching does not really help much regarding upstream throttling

**Created:** 2026-04-21
**State:** open
**Author:** @benoit74

## Problem Statement

Even with S3 image caching, the scraper still needs to perform a HEAD request per image to check the ETag (cache freshness). This counts against the throttling budget.

Key numbers:
- i.sstatic.net hosts 99+% of stackoverflow.com images
- Throttle budget: ~10 requests/second
- Total images: ~4M
- Time estimate: ~5 days JUST to download all images
- HEAD requests return 429 (too many requests) when throttled

## Discussion

### @kelson42 (2026-05-06)
"The cache is not only, and actually not primarily, there to improve situation around throttling."

### @rgaudin (2026-05-06)
The S3 cache had two roles:
1. Saving optimized media for reuse (predominant in YouTube scraper)
2. Saving network requests to not overwhelm source servers

"Today, CPU resources are abundant in ZF workers but network has become a bottleneck partly due to the rise of LLM-scrapers."

Proposed optimization: use S3 cache without checking ETag for some images (e.g., user profile images).

### @benoit74 (2026-05-06)
Confirmed the cache doesn't help with throttling at all. Whether it's a problem worth solving is still open.

## Key Insights

- **Network is now the bottleneck** for large ZIM builds, not CPU
- LLM scrapers have increased general web scraping pressure, tightening rate limits
- SO image download alone takes ~5 days at sustainable rate
- The full SO build pipeline is: dump download -> XML processing -> sort -> post rendering -> image download -> ZIM creation
- Each phase can take days for SO specifically
