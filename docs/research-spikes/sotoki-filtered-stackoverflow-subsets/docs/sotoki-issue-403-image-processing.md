---
source: https://github.com/openzim/sotoki/issues/403
retrieved: 2026-05-14
type: github-issue
---

# Issue #403: stackoverflow.com still fails while processing images

**Created:** 2026-04-05
**State:** closed

## Problem
Image processing was "damned slow and then went (mostly?) completely stuck."

This indicates that even after fixing the OOM sort issue (#394), the SO build hit another bottleneck: image processing at scale. With millions of questions containing images, the image download/processing pipeline becomes a critical bottleneck.
