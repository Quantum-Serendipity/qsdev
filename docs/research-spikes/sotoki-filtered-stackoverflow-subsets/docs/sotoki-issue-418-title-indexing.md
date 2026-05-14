---
source: https://github.com/openzim/sotoki/issues/418
retrieved: 2026-05-14
type: github-issue
---

# Issue #418: stackoverflow.com fails with libzim TitleIndexingError

**Created:** 2026-05-04
**State:** closed

## Problem
The SO build failed because a question had a title full of Unicode control characters (RTL/LTR marks). The question was SO #71417749 with title containing dozens of ‏ and ‎ characters.

libzim's Creator threw a TitleIndexingError which aborted the ZIM creation.

## Fix
Strip control characters from post titles before adding to ZIM.

## Significance
This shows the Kiwix team is ACTIVELY trying to build a new SO ZIM as of May 2026, but keeps hitting edge cases due to SO's massive and messy dataset. Each build attempt takes days and reveals new failure modes.
