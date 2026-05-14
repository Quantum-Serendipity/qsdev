---
source: https://github.com/openzim/sotoki/issues/394
retrieved: 2026-05-14
type: github-issue
---

# Issue #394: stackoverflow_en got (OOM?) killed during post sort

**Created:** 2026-03-13
**State:** closed

## Problem

The `sort` process got OOM killed during the SO build. Key details:

- `--buffer-size 160525965312b` = 149.5 GB allocated to GNU sort
- Sotoki allocates 90% of "available memory" for sort buffer
- Container was supposed to have only 80 GB memory assigned
- Host has 172 GB total RAM
- Bug: sotoki was reading host memory (172G) instead of container memory limit (80G)
- 90% of 172G = ~155G, close to the 149.5G seen

## Timeline from logs (2026-02-21)
- 13:56 - Sorted Badges by UserId
- 13:56 - Removed users headers
- 13:58 - Merged both sets, PROGRESS: 0.8% Step 1/8
- 14:03 - Removed posts headers
- 14:07 - sort process killed by SIGKILL (OOM)

Total time before crash: ~11 minutes into a sort of posts_nohead.xml

## Technical Stack Trace
Failed in: `merge_posts_with_answers_comments()` -> `create_sorted_posts()` -> `sort_dump_by_id()` -> `sort_dump_by_id_gnusort()`

## Key Insight
The SO build requires massive memory for sorting the posts XML dump. Even 80GB container memory may not be enough. The bug was that the memory detection code was reading `/proc/meminfo` (host) instead of cgroup memory limits (container).
