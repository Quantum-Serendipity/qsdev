# Tasks: Sotoki Filtered Stack Overflow Subsets

## Phase 1: Sotoki Capabilities & Data Architecture

### Pending

### Active

### Completed

- **P1-T1: Sotoki CLI, configuration, and tag-filtering capability**
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-14
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: Sotoki does NOT support tag filtering natively. The `--tags` flag is ZIM metadata only. GitHub issue #287 (open since Jul 2023, no response) requests this exact feature. The `--without-unanswered` filter pattern shows exactly where tag filtering would slot in. Adding it estimated at 225-325 lines across 5-6 files (2-4 days). Full dump must still be downloaded regardless. Report: sotoki-cli-filtering-research.md

- **P1-T2: Stack Exchange data dump structure and tag architecture**
  - Priority: high
  - Estimate: small
  - Started: 2026-05-14
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: Tags are denormalized angle-bracket strings on questions only. Synonyms pre-resolved at write time. Cascade problem well-defined (5 tiers). Pre-filtering feasible via two-pass SAX streaming, ~200 MB memory for popular tags. No existing tool produces filtered XML dumps. SEDE not viable (50K row limit). Report: se-data-dump-structure-research.md

- **P1-T5: Cross-tag link integrity analysis**
  - Priority: medium
  - Estimate: small
  - Started: 2026-05-14
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: Three link pathways analyzed: (1) Sidebar "Linked" links from PostLinks.xml bypass rewriter and produce hard-broken links to non-existent ZIM entries. (2) Inline body links are gracefully degraded by the rewriter (href removed, text preserved). (3) Duplicate markers not fully confirmed in templates. Estimated 30-40% of sidebar links broken for broad tags, 50-70% for narrow tags. Recommended strategy: filter broken sidebar links during build + include duplicate targets for chain integrity. ~50 lines of implementation. Report: cross-tag-link-integrity-research.md

- **P1-T4: Estimate per-ecosystem slice sizes**
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-14
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: ZIM sizes range from 250 MB (Rust) to 13 GB (JS/TS full). Small ecosystems trivially fit. Large ecosystems (Python ~10 GB, JS ~13 GB) fit on workstations but exceed 2-3 GB target without temporal/quality/no-images filtering. DevOps ~2 GB validates calibration against Ask Ubuntu (2.6 GB). Report: ecosystem-slice-sizes-research.md

- **P1-T6: Alternative approaches if native filtering is absent**
  - Priority: medium
  - Estimate: small
  - Started: 2026-05-14
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: Decision matrix: Option B (fork sotoki) 8.2/10, Option A (pre-filter XML) 6.7/10, Option C (post-process ZIM) 1.6/10 eliminated. Key finding: maintainers want tag filtering — rgaudin provided implementation sketch, kelson42 endorsed "one ZIM per language." CONTRIBUTING.md welcomes external PRs. Report: approach-comparison-research.md

- **P1-T3: Existing pre-built ZIM ecosystem and build infrastructure**
  - Priority: high
  - Estimate: small
  - Started: 2026-05-14
  - Completed: 2026-05-14
  - Outcome: success
  - Notes: Zimfarm is ~6 servers, 4000+ recipes. Full SO ZIM stale since Nov 2023 due to cascading failures (OOM at 80+ GB RAM, Cloudflare throttling, Unicode edge cases). All other SE sites current. No tag-filtered ZIMs exist anywhere. Self-hosted filtered builds feasible on modest hardware (~16 GB RAM, ~100 GB disk, hours not weeks). Report: kiwix-build-infrastructure-research.md

## Phase 2: Experimental Validation (pending Phase 1 findings)

### Pending

### Active

### Completed
