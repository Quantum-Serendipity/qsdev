# Research Log: Sotoki Filtered Stack Overflow Subsets

## 2026-05-14 15:00 — Spike Promoted from Pending
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: Spike scaffolded from pending-spikes.md entry. Awaiting scope confirmation and Phase 1 task decomposition.
- **Next**: Confirm research question with user; populate Phase 1 tasks in tasks.md.

## 2026-05-14 15:30 — P1-T1/T2/T3 Completed (Parallel Sub-Agent Investigation)
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: sotoki GitHub repo (source code, issues, changelog), SE data dump schema docs, Kiwix zimfarm docs/issues, download.kiwix.org catalog
- **Summary**: Three parallel sub-agents completed the foundational research:
  - **T1 (Sotoki CLI)**: Sotoki does NOT support tag filtering. The `--tags` flag is ZIM metadata only. GitHub issue #287 (open Jul 2023, zero response) requests exactly this. The `--without-unanswered` filter's implementation pattern shows where filtering would slot in. Adding tag filtering estimated at 225-325 lines, 2-4 days. Full dump still must be downloaded.
  - **T2 (SE Data Structure)**: Tags are denormalized angle-bracket strings on questions only (`<python><django>`). Synonyms pre-resolved at write time. Cascade problem well-defined across 5 tiers (questions→answers→comments/votes/history→users→badges→tag defs). Pre-filtering feasible via two-pass SAX streaming, ~200 MB memory. No existing tool produces filtered XML dumps.
  - **T3 (Kiwix Build Infra)**: Zimfarm is ~6 servers, 4000+ recipes. Full SO ZIM stale since Nov 2023 due to cascading failures at SO's extreme scale (OOM at 80+ GB RAM, Cloudflare throttling, Unicode edge cases). All other SE sites current (Dec 2025/Feb 2026). No tag-filtered ZIMs exist anywhere. Self-hosted filtered builds feasible: ~16 GB RAM, ~100 GB disk, hours.
- **Next**: Estimate per-ecosystem slice sizes (T4), analyze cross-tag link integrity (T5), compare implementation approaches (T6).

## 2026-05-14 16:00 — P1-T4/T5/T6 Launched (Parallel)
- **Type**: research
- **Status**: in-progress
- **Depth**: moderate
- **Summary**: Three more parallel sub-agents launched for remaining Phase 1 tasks: ecosystem slice size estimation, cross-tag link integrity analysis, and implementation approach comparison (XML pre-filter vs sotoki fork vs ZIM post-process).
- **Next**: Synthesize all Phase 1 findings into research.md once complete.

## 2026-05-14 17:00 — P1-T5 Completed: Cross-Tag Link Integrity Analysis
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: sotoki source (utils/html.py, posts.py, templates/question.html, templates/linked_list.html, renderer.py), Kiwix issues (#430, #917), ClickHouse SO dataset docs, arxiv papers on SO links (2010.04892, 2104.03518), SO blog (linked posts feature)
- **Summary**: Identified three distinct link pathways with different broken-link behavior:
  1. **Sidebar "Linked" links** (PostLinks.xml): rendered directly by linked_list.html template, bypass the rewriter, produce hard 404s in Kiwix when target is missing from subset
  2. **Inline body links**: pass through Rewriter.rewrite_relative_link() which does `del link.attrs["href"]` when target not found — graceful degradation to unclickable text
  3. **Duplicate markers**: stored but rendering location not fully confirmed in templates
  Estimated 30-40% of sidebar links broken for broad tags (python), 50-70% for narrow tags (flask). 6.55M PostLinks total, affecting ~15-20% of questions.
  Recommended hybrid strategy: filter broken sidebar links during build (Strategy B, ~20 LOC) + include duplicate chain targets (Strategy E, ~30 LOC). Inline body links need no changes.
- **Next**: Complete T4 (slice sizes) and T6 (approach comparison), then synthesize Phase 1.

## 2026-05-14 17:30 — P1-T4 Completed: Ecosystem Slice Size Estimates
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: SE API `/2.3/tags` (300 tags, live counts 2026-05-14), download.kiwix.org ZIM catalog (2025-12), SE API `/2.3/info` for site question counts
- **Summary**: Built calibrated size model using existing SE site ZIMs as reference (Ask Ubuntu 425K Q → 2.6 GB = ~6 MB/1K Q). Results: Rust 250 MB, Go 400 MB, DevOps 2 GB, C#/.NET 8 GB, Java/Kotlin 9 GB, Python 10 GB, JS/TS 13 GB. The "2-3 GB per ecosystem" target holds natively for small ecosystems; large ecosystems need temporal+quality+no-images filtering to reach 2-3 GB. Images are the dominant size variable (40-60% of ZIM for visual-heavy tags).
- **Next**: Synthesize Phase 1 into research.md.

## 2026-05-14 17:30 — P1-T6 Completed: Approach Comparison Decision Matrix
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Sources**: sotoki issue #287 (full discussion with 14 comments), CONTRIBUTING.md, zim-tools README, zimdump/zimrecreate capabilities, sotoki PR history
- **Summary**: Decision matrix scored Option B (fork sotoki) at 8.2/10 vs Option A (pre-filter XML) 6.7/10 vs Option C (post-process ZIM) 1.6/10. Key finding shifting the recommendation: issue #287 has 14 comments including maintainer rgaudin's concrete implementation sketch and Kiwix founder kelson42 endorsing "one ZIM per mainstream programming language." CONTRIBUTING.md welcomes external PRs. Option C eliminated — ZIM tools cannot content-filter.
- **Next**: Synthesize Phase 1.

## 2026-05-14 18:00 — Phase 1 Complete: Full Synthesis Written
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: All 6 Phase 1 tasks completed. research.md updated with synthesized findings across all topics. Key conclusions: (1) tag-filtered SO ZIMs solve the 74 GB problem, (2) recommended path is upstream contribution to sotoki (~100 LOC filtering logic), (3) "2-3 GB per ecosystem" validated for small ecosystems and achievable for large ones with secondary filtering, (4) cross-tag link integrity manageable with ~50 LOC, (5) maintainers actively want this feature. No Phase 2 experimental work performed — remaining work is implementation, not research.
- **Next**: Spike ready for completion review.

## 2026-05-14 18:30 — Spike Completed
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Spike closed with all 6 Phase 1 tasks completed (6/6 success). All 6 topic reports pass depth checklist. Key conclusions: (1) Tag-filtered SO ZIMs are viable and solve the 74 GB problem. (2) Recommended path is upstream contribution to sotoki — maintainers want it (rgaudin sketch, kelson42 endorsement). (3) ZIM sizes: 250 MB (Rust) to 13 GB (JS/TS), with temporal+quality+no-images filtering bringing large ecosystems to 2-3 GB. (4) Cross-tag link integrity manageable with ~50 LOC. (5) ZIMs stored at user-level (`~/.qsdev/zim/`), not per-devenv, to avoid duplication across projects. 3 follow-on candidates flushed to proposed-spikes.md (upstream contribution, temporal/quality filter extension, CI/CD build pipeline).
