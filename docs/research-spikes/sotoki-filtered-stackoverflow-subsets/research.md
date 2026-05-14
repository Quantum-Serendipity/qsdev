# Research Summary: Sotoki Filtered Stack Overflow Subsets

## Overview

This spike investigated whether sotoki (the openZIM Stack Exchange-to-Kiwix-ZIM converter) can build tag-filtered Stack Overflow subsets small enough for local developer workstations. The parent spike (`gdev-local-docs-mcp`) identified the 74 GB full SO ZIM as impractical and proposed tag-filtered slices at ~2-3 GB per ecosystem.

**Bottom line**: Tag-filtered SO ZIMs are viable and the recommended path is contributing tag filtering directly to sotoki upstream — the maintainers actively want this feature.

## Topics

### 1. Sotoki Tag Filtering Capability — [Complete](sotoki-cli-filtering-research.md)
Sotoki does **not** support tag-based content filtering. The `--tags` CLI flag sets ZIM catalog metadata, not content filters. The only content filter is `--without-unanswered`. GitHub issue #287 (open Jul 2023) requests tag filtering; the maintainer (rgaudin) provided a 4-point implementation sketch and called it "quite easy." The Kiwix founder (kelson42) endorsed "one ZIM per mainstream programming language."

### 2. Stack Exchange Data Dump Structure — [Complete](se-data-dump-structure-research.md)
Tags are denormalized angle-bracket strings on questions only (`<python><django>`). Synonyms are pre-resolved at write time, so filtering by canonical tag name captures all posts. The cascade problem (questions → answers → comments/votes/history → users → badges) is well-defined and requires ~200 MB memory for a popular tag like `python`. No existing tool produces tag-filtered XML dumps.

### 3. Kiwix Build Infrastructure — [Complete](kiwix-build-infrastructure-research.md)
The Kiwix zimfarm runs ~6 servers with 4,000+ recipes. The full SO ZIM has been stuck at Nov 2023 (75 GB) due to cascading failures at SO's extreme scale: OOM kills needing 80+ GB RAM, Cloudflare throttling on 4.1M images, and Unicode edge cases. All other SE sites (~500 ZIMs) are current. No tag-filtered ZIMs exist anywhere. Self-hosted filtered builds are feasible on modest hardware (~16 GB RAM, ~100 GB disk, hours not weeks).

### 4. Per-Ecosystem ZIM Size Estimates — [Complete](ecosystem-slice-sizes-research.md)
Tag counts from the SE API (2026-05-14) and calibration against existing SE site ZIMs yield:

| Ecosystem | Unique Questions | ZIM Size Est. | Verdict |
|-----------|-----------------|---------------|---------|
| Rust | ~44,600 | ~250 MB | Trivial |
| Go | ~75,200 | ~400 MB | Trivial |
| DevOps (core) | ~354,000 | ~2 GB | Trivial |
| C#/.NET | ~1,814,000 | ~8 GB | Ideal |
| Java/Kotlin (no Android) | ~2,025,000 | ~9 GB | Ideal |
| Python (full) | ~2,335,000 | ~10 GB | Acceptable |
| JS/TS (full) | ~2,970,000 | ~13 GB | Acceptable |

The "2-3 GB per ecosystem" target is achievable natively for small/medium ecosystems and for large ecosystems with temporal (post-2018+), quality (score ≥ 1), or no-images filtering. Combined filters bring Python from ~10 GB to ~2-3 GB.

### 5. Cross-Tag Link Integrity — [Complete](cross-tag-link-integrity-research.md)
Three link pathways with different failure modes:
- **Inline body links**: Already handled gracefully — sotoki's rewriter removes `href` when target is missing (text preserved, link becomes unclickable)
- **Sidebar "Linked" links**: Bypass rewriter, produce hard 404s in Kiwix. Estimated 30-40% broken for broad tags, 50-70% for narrow tags
- **Duplicate markers**: Chain integrity matters for UX

Fix: filter broken sidebar links during build (~20 LOC) + include duplicate chain targets (~30 LOC). Fits within the overall implementation estimate.

### 6. Implementation Approach Comparison — [Complete](approach-comparison-research.md)
Decision matrix scored three approaches:

| Approach | Score | Verdict |
|----------|-------|---------|
| **B: Fork sotoki, add `--include-tags`/`--exclude-tags`** | 8.2/10 | **Recommended** |
| A: Pre-filter XML dump, feed to unmodified sotoki | 6.7/10 | Fallback |
| C: Post-process full 75 GB ZIM | 1.6/10 | Eliminated |

Option B wins because: (a) maintainers want it and provided implementation guidance, (b) sotoki's pipeline already handles cascade effects, (c) single tool invocation, (d) clear upstream PR path. Option C is eliminated — ZIM tools cannot content-filter and the full SO build requires 80+ GB RAM and 2+ weeks.

## Open Questions

1. **Will the upstream PR be accepted?** The maintainers endorsed the feature but nobody has implemented it in 3 years. A well-crafted PR has strong odds but isn't guaranteed. Fallback to Option A (pre-filter XML) if rejected.

2. **Image download throttling for large ecosystem slices**: Even a filtered build for Python (~2.3M questions) will need to download millions of images from `i.sstatic.net`. Cloudflare throttling at ~10 req/s means days of image download. The `--without-images` flag is a practical mitigation.

3. **Temporal/quality filtering**: Sotoki doesn't currently support filtering by date or score. If these prove necessary for large ecosystems, they'd need the same fork-and-add approach. The maintainer sketch only covers tag filtering.

4. **CI/CD automation**: Building filtered ZIMs on a quarterly cadence requires infrastructure. A GitHub Actions workflow downloading the ~67 GB compressed SO dump, running the filter, and uploading the result needs either large runners or self-hosted infrastructure.

5. **Distribution**: How does gdev discover and download the right ecosystem ZIM? Options: host on a CDN, use GitHub Releases (2 GB limit), BitTorrent, or piggyback on Kiwix's library if the upstream PR lands.

## Design Decision: Shared User-Level ZIM Storage

ZIM files are static, read-only, and ecosystem-scoped — not project-scoped. An engineer working on 5 Python repos should have **one** copy of the Python SO ZIM, not five. Storage must be at the user level, not per-devenv:

- **Linux**: `$XDG_DATA_HOME/qsdev/zim/` (defaults to `~/.local/share/qsdev/zim/`)
- **macOS**: `~/Library/Application Support/qsdev/zim/`
- **Windows**: `%LOCALAPPDATA%\qsdev\zim\`

**Flow**: `gdev init` detects ecosystem from project files (`pyproject.toml` → python, `Cargo.toml` → rust, `package.json` → javascript) → checks shared store for the ecosystem ZIM → downloads on first use or quarterly refresh → configures local MCP server to serve from the shared path. ZIM is read-only and safe for concurrent access across multiple devenvs.

## Conclusions

1. **Tag-filtered SO ZIMs solve the 74 GB problem.** The full SO ZIM is unmaintainable (stale since 2023). Per-ecosystem slices at 250 MB to 13 GB are buildable on modest hardware in hours, not weeks.

2. **The recommended path is contributing `--include-tags`/`--exclude-tags` to sotoki upstream.** The maintainers want this feature, provided an implementation sketch, and welcome external PRs. Implementation is ~100 lines of filtering logic plus ~100-200 lines of tests, following the existing `--without-unanswered` pattern.

3. **The "2-3 GB per ecosystem" target from the parent spike is partially validated.** It holds natively for Rust (250 MB), Go (400 MB), and DevOps (~2 GB). For Python (~10 GB) and JS/TS (~13 GB), it requires secondary filtering (temporal, quality, no-images) or ecosystem sub-splitting.

4. **Cross-tag link integrity is manageable.** Inline body links already degrade gracefully. Sidebar broken links need ~50 LOC of additional filtering, which fits within the implementation scope.

5. **The full SE dump must still be downloaded** regardless of approach (~67 GB compressed). Filtering reduces only the ZIM output size and build resource requirements, not the input data size. This is the irreducible cost.

6. **No experimental validation was performed.** Phase 1 research is complete; actually building a tag-filtered ZIM (Phase 2) would require forking sotoki, implementing the feature, and running a test build. This is implementation work, not research.
