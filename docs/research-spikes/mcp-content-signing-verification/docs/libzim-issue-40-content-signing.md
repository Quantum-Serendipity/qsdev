# libzim Issue #40 — Add Spec to Allow Content Signing

- **Source URL**: https://github.com/openzim/libzim/issues/40
- **Retrieved**: 2026-05-14
- **Data source**: GitHub API (gh api)

## Issue Status
**Open** — Created 2017-07-31, no assignee

## Issue Body
(Empty — title only: "Add spec to allow content signing")

## Discussion (from comments)

**@mgautierfr** (maintainer): "I'm suppose you are speaking about 'content signing' no?"

**@kelson42** (Kiwix lead): "Yes, this is not urgent, but at middle term I support we should have a solution to sign digitally the ZIM files."

**@mofosyne**: Proposed a 'community manifest' filled with hashes of important files from library.kiwix.org baked into the binary itself as a stepping stone. Rationale: would provide verification UX that could later be replaced with proper crypto. Also raised the sneakernet use case — sharing ZIM files offline in places with blocked/slow/expensive internet.

**@kelson42**: Skeptical — "if they download from library.kiwix.org, the authenticity is anyway secure via https, so what you propose seems to me a very imperfect hack not adding much value."

**@mofosyne** (rebuttal): Pointed out offline/sneakernet sharing scenarios where HTTPS download provenance is unavailable.

## Key Takeaways

1. The Kiwix team acknowledges the need for content signing but considers it non-urgent
2. Issue has been open since July 2017 — nearly 9 years with no implementation
3. The lead maintainer's mental model is HTTPS-centric (download provenance), not offline-first
4. Community member raised the offline/sneakernet case — the exact scenario relevant to our research
5. No concrete technical proposal or spec has been drafted
6. Cross-referenced from issue #614 comment: "if at some point we need to sign content, see #40, linking to openssl will be necessary"
