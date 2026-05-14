# Kiwix Build Infrastructure & Stack Overflow ZIM Status

## Executive Summary

Kiwix operates a semi-decentralized build farm called **Zimfarm** that runs 4,000+ recipes across ~6 servers, producing ZIM files for 100+ languages. All Stack Exchange sites except english Stack Overflow receive regular updates (most recently Dec 2025 / Feb 2026). The full SO ZIM has been stuck at Nov 2023 (75 GB) because each build attempt hits a new failure mode -- OOM kills during XML sorting, Cloudflare throttling during image downloads (4.1M images at ~10 req/s = 5 days), and edge-case data corruption. The Kiwix team is actively trying to fix this as of May 2026 but has not yet produced a successful build. No community-built tag-filtered SO ZIMs exist. Sotoki has no native tag-filtering capability. The only tool that offered tag-based content filtering was **Seekoff**, which is archived and unmaintained since Dec 2022.

---

## 1. Zimfarm Architecture

### Components

The Zimfarm ([openzim/zimfarm](https://github.com/openzim/zimfarm)) is a semi-decentralized system with these components:

| Component | Role |
|-----------|------|
| **Backend** | Central database + API. Stores recipes (ZIM metadata), schedules tasks, assigns work to workers. Hosted at `api.farm.openzim.org/v2`. |
| **Frontend** | Web UI at `farm.openzim.org` for creating/editing recipes and monitoring task progress. |
| **Manager** | Lightweight container on each worker node. Declares available resources (CPU, RAM, disk), polls for tasks every 180s, spawns task-worker containers. |
| **Task-Worker** | Spawned per task. Runs the scraper, monitors progress, manages uploads. |
| **Uploader** | Handles ZIM file and log uploads via SFTP/SCP to the receiver. |
| **Receiver** | Jailed SSH server that accepts completed ZIMs and routes them to `download.kiwix.org`. |
| **DNSCache** | Local dnsmasq per worker for stable DNS during long-running tasks. |
| **Scrapers** | Independent tools invoked by task-workers: `mwoffliner` (MediaWiki), `sotoki` (Stack Exchange), `zimit` (generic web crawl), etc. |

### Recipe-Based Scheduling

Each ZIM file has a **recipe** -- metadata defining which scraper to use, what parameters to pass, and how often to rebuild. The backend automatically creates tasks from recipes and assigns them to workers with sufficient declared resources.

### Infrastructure Scale

- ~6 servers running 24/7
- 4,000+ recipes across 100+ languages
- Most ZIMs updated monthly
- Completed ZIMs distributed to `download.kiwix.org` and mirrored worldwide

### GSoC 2025 Reengineering

The zimfarm underwent major modernization in 2025: Flask to FastAPI, Marshmallow to Pydantic, JS to TypeScript, Vue 2 to Vue 3. Over 100 PRs merged. The rewrite addressed fragilities like crashing on special characters, missing field validation, and security improvements.

**Source**: `docs/gsoc-2025-zimfarm-reengineering.md`, `docs/zimfarm-readme.md`

---

## 2. Worker Hardware Requirements

### Default Worker Configuration

The worker setup script (`workers/contrib/zimfarm.sh`) declares these defaults:

| Resource | Default | Notes |
|----------|---------|-------|
| RAM | 2 GB | Sufficient for small SE sites |
| Disk | 10 GB | Sufficient for sites under ~5 GB |
| CPU cores | 3 | Minimum recommended |

### Minimum Requirements (from docs)

- 2 GB RAM, 3 CPU cores
- Docker CE
- Fast bidirectional internet
- Linux/macOS
- Synchronized system clock

### Stack Overflow Build Requirements

The SO build is an extreme outlier. Based on issues #394, #403, #409, and #418:

| Resource | SO Requirement | Rationale |
|----------|---------------|-----------|
| RAM | **80-172 GB** | GNU sort of Posts.xml allocates 90% of available memory. 32 GB was insufficient (OOM killed). Container assigned 80 GB on a 172 GB host. |
| Disk | **500+ GB** | 75 GB output ZIM + SE data dump (~100 GB compressed, ~200 GB extracted XML) + temp sort files + image cache |
| Network | Sustained for **5+ days** | 4.1M images from `i.sstatic.net` at ~10 req/s throttle limit = ~5 days for image download alone |
| CPU | Multi-core, days of runtime | XML parsing, HTML rendering, image optimization, ZIM compression |
| Total build time | **2+ weeks** estimated | Dump download + XML sort + post processing + image download + ZIM finalization |

The SO build runs on dedicated high-memory machines. The worker "kathrin" referenced in issues has 172 GB RAM.

**Source**: `docs/sotoki-issue-394-oom-post-sort.md`, `docs/sotoki-issue-409-throttling.md`, `docs/zimfarm-worker-script.md`

---

## 3. Why the Full SO ZIM Is Stale

The last SO ZIM was `stackoverflow.com_en_all_2023-11.zim` (75 GB, published 2023-12-01). It has not been refreshed in over 2.5 years. This is due to a cascade of technical failures, not a deliberate choice:

### Timeline of Build Failures

| Date | Issue | Problem |
|------|-------|---------|
| 2020-09 | [#174](https://github.com/openzim/sotoki/issues/174) | "stackoverflow's duration is not manageable" -- 3 GB/day build rate, 32 GB OOM kill. Architectural weakness with temp files on filesystem. |
| 2022-10 | Overflow Offline partnership | Stack Overflow provides financial and technical support to Kiwix. Successful builds in 2023 (May and November). |
| 2023-11 | Last successful build | `stackoverflow.com_en_all_2023-11.zim` (75 GB) published. |
| 2024-2025 | Gap | No SO ZIM builds appear to succeed. Sotoki undergoes major version upgrades (v2.x to v3.0). |
| 2026-02-21 | [#394](https://github.com/openzim/sotoki/issues/394) | OOM kill during post sort. `sort --buffer-size 160525965312b` (149.5 GB!) -- bug read host memory (172 GB) instead of container limit (80 GB). |
| 2026-03-02 | [#392](https://github.com/openzim/sotoki/issues/392) | New SO design coming -- will need CSS/template updates. |
| 2026-04-05 | [#403](https://github.com/openzim/sotoki/issues/403) | Image processing "damned slow and then went completely stuck." Stopped at 85k of 4.1M images. |
| 2026-04-21 | [#409](https://github.com/openzim/sotoki/issues/409) | Even with S3 caching, Cloudflare throttling limits to ~10 req/s. ~5 days for all images. Root cause: `i.sstatic.net` returns 429 on HEAD requests. |
| 2026-05-04 | [#418](https://github.com/openzim/sotoki/issues/418) | Build fails on a question with Unicode control characters in the title. libzim throws TitleIndexingError. Fixed by stripping control chars. |

### Root Causes

1. **Scale**: SO has ~24M questions, ~4.1M images. It is 10x larger than the next biggest SE site (math at 6.9 GB). Every pipeline stage that works fine for other sites breaks at SO scale.

2. **Resource demands**: 80+ GB RAM for sorting, hundreds of GB disk, days of sustained network access. These requirements exceed normal zimfarm worker capacity by 40x (RAM) and 50x (disk).

3. **External dependencies**: Cloudflare rate-limiting on `i.sstatic.net` adds ~5 days to every build attempt. LLM-era web scraping has tightened rate limits globally.

4. **Long feedback cycles**: Each build attempt takes days-to-weeks. When it fails at a new point, the fix-and-retry cycle is extremely slow.

5. **Edge cases at scale**: Unicode control characters in titles, stuck URL loops, image processing bugs -- issues that never surface on smaller SE sites.

### Current Status (May 2026)

The team is actively working on it. Issue #418 (TitleIndexingError) was fixed 2026-05-04. Issue #409 (throttling optimization) is still open. The next build attempt may succeed, but each attempt reveals new failure modes.

**Source**: `docs/sotoki-issue-174-comments.md`, `docs/sotoki-issue-394-oom-post-sort.md`, `docs/sotoki-issue-403-comments.md`, `docs/sotoki-issue-409-throttling.md`, `docs/sotoki-issue-418-title-indexing.md`

---

## 4. Current SE ZIM Catalog

### Stack Overflow (English)
| File | Size | Date | Status |
|------|------|------|--------|
| `stackoverflow.com_en_all_2023-05.zim` | 74 GB | 2023-06-06 | Stale |
| `stackoverflow.com_en_all_2023-11.zim` | 75 GB | 2023-12-01 | **Latest** (2.5 years old) |

### Other Large SE Sites (all recently updated)
| Site | Size | Latest Date |
|------|------|-------------|
| math.stackexchange.com | 6.9 GB | 2026-02 |
| tex.stackexchange.com | 4.2 GB | 2026-02 |
| electronics.stackexchange.com | 3.9 GB | 2026-02 |
| superuser.com | 3.7 GB | 2026-02 |
| askubuntu.com | 2.6 GB | 2025-12 |
| blender.stackexchange.com | 2.6 GB | 2025-12 |
| ru.stackoverflow.com | 2.5 GB | 2025-12 |
| diy.stackexchange.com | 1.9 GB | 2025-12 |

### Key Pattern

- **500+ ZIM files** covering the full Stack Exchange ecosystem
- All sites **except english SO** have recent builds (Dec 2025 or Feb 2026)
- Most sites are updated on a roughly quarterly cycle
- The gap is entirely a resource/scale problem unique to english SO

**Source**: `docs/kiwix-se-zim-catalog.md`

---

## 5. Community/Custom ZIM Builds

### No Tag-Filtered SO ZIMs Exist

Extensive searching found zero published tag-filtered Stack Overflow ZIM files. No community member, organization, or project has published a subset ZIM filtered by programming language or framework tags.

### Seekoff (Archived)

The closest prior art is **[Seekoff](https://github.com/Caspia/seekoff)** -- an offline SO reader with tag-based inclusion/exclusion. Key details:

- **Purpose**: Prison education use case (restrict security topics)
- **Architecture**: Elasticsearch + Node.js webapp in Docker containers
- **Tag filtering**: Supports include/exclude lists for SE tags
- **Status**: **Archived December 2022** -- no longer maintained
- **Output format**: Elasticsearch index, NOT ZIM files
- **Limitation**: Separate tool, cannot produce ZIMs, requires Elasticsearch infrastructure

Seekoff validates the use case for tag-filtered SO subsets but provides no reusable ZIM tooling.

### TheStaticTurtle's Setup

A blogger set up a self-hosted full SO ZIM (161 GB, 2021 dump) on Proxmox with NAS storage and a custom browser redirect extension. No filtering -- used the full dataset. Sourced the ZIM from Reddit's DataHoarder community because the official download was stale.

### Kiwix Self-Hosting

Multiple blog posts describe self-hosting Kiwix with Docker for Wikipedia, Arch Wiki, and SE sites. All use pre-built full ZIMs from `download.kiwix.org`. None describe building custom filtered versions.

**Source**: `docs/seekoff-readme.md`, `docs/staticturtle-offline-so-setup.md`, `docs/rickcarlino-offline-so.md`

---

## 6. Sotoki's Tag-Filtering Capability (or Lack Thereof)

### Current CLI Flags for Content Control

Sotoki offers these content-reduction flags:

| Flag | Effect |
|------|--------|
| `--without-images` | Exclude images (reduces size, keeps all questions) |
| `--without-unanswered` | Exclude zero-answer posts |
| `--without-user-profiles` | Skip user profile pages |
| `--without-external-links` | Strip external URLs |
| `--without-names` | Replace usernames with generated ones |
| `--censor-words-list` | Remove posts containing specific words |

### What Does NOT Exist

- No `--include-tags` or `--exclude-tags` flag
- No `--tag-filter` or equivalent
- No way to select a subset of questions by their SE tags
- The `--tags` flag is for ZIM metadata only (how the ZIM appears in the Kiwix catalog)

### Implication for This Spike

Building tag-filtered SO ZIM subsets requires either:
1. **Pre-filtering the XML dump** before passing it to sotoki (filter Posts.xml to only include questions with desired tags, then filter related files)
2. **Modifying sotoki's source code** to add tag-based filtering
3. **Post-processing the ZIM** to remove unwanted content (not well-supported by ZIM tooling)

Option 1 (XML pre-filtering) is the most viable path. The SE data dump XML format includes tags directly in each post's `Tags` attribute.

**Source**: `docs/sotoki-cli-arguments.md`, `docs/sotoki-changelog-v3.md`

---

## 7. Self-Hosting Build Infrastructure Requirements

### For Normal SE Sites (< 7 GB)

A standard zimfarm worker setup is sufficient:
- 4+ GB RAM
- 20+ GB disk
- Docker CE on Linux
- Moderate internet connection
- Build time: hours

### For Full Stack Overflow

Based on the zimfarm's own struggles:
- **RAM**: 80-128 GB minimum (sort alone needs 32+ GB, container had 80 GB)
- **Disk**: 500+ GB fast storage (SSD strongly preferred for sort performance)
- **Network**: Sustained for 5+ days, ideally with multiple IPs to manage Cloudflare throttling
- **CPU**: 4+ cores (not the primary bottleneck)
- **Redis**: Required for sotoki's internal state management
- **S3 cache**: Optional but recommended for image optimization caching
- **Build time**: 2-3 weeks per attempt
- **Failure tolerance**: Expect multiple failed attempts before success

### For Tag-Filtered SO Subsets (Estimated)

If pre-filtering the XML dump reduces SO by ~95% (e.g., a single-ecosystem slice):
- **RAM**: 8-16 GB (sort of ~5% of Posts.xml)
- **Disk**: 50-100 GB (dump + temp + output)
- **Network**: Hours for image download (proportionally fewer images)
- **CPU**: 2-4 cores
- **Build time**: Hours to a day
- **Viability**: High -- resource requirements become comparable to large SE sites like math.stackexchange.com

---

## 8. Requesting Custom ZIM Builds from Kiwix

### zim-requests Repository

[openzim/zim-requests](https://github.com/openzim/zim-requests) is the official channel for requesting new ZIM files. Users file an issue with website URL, title, description, language, and license info. The team creates a zimfarm recipe, and if it works, the ZIM appears in the library within 24-48 hours.

### Limitations

- Designed for adding **new websites** to the catalog, not custom filters of existing ones
- SO already has a recipe -- the problem is it fails to build
- No mechanism to request tag-filtered subsets
- Kiwix offers fee-based custom builds through the Zimit service, but Zimit crawls websites (not suitable for SO -- too large, rate-limited)

### Zimit (youzim.it)

Kiwix's self-service ZIM builder for arbitrary websites. Free tier limited to 1,000 pages or 2 hours of crawling. Completely unsuitable for SO (24M+ questions). Works by web crawling, not data dump processing.

**Source**: `docs/zimit-service.md`

---

## 9. Key Findings for This Spike

1. **No tag-filtered SO ZIMs exist anywhere.** This is a genuine gap in the ecosystem.

2. **Sotoki cannot filter by tag natively.** Pre-filtering the XML dump is the most viable approach.

3. **The full SO ZIM build is a known, actively-worked problem** that has been failing for 2.5 years. The Kiwix team is fixing issues one by one as of May 2026.

4. **Tag-filtered subsets would dramatically reduce build requirements** -- from 80+ GB RAM / 500+ GB disk / weeks of build time to ~16 GB RAM / 100 GB disk / hours. This makes CI/CD builds feasible.

5. **The Overflow Offline partnership** (Oct 2022) between Stack Overflow and Kiwix provided funding and resources but has not solved the scale problem.

6. **Network throttling is a growing concern** -- Cloudflare rate limits have tightened, partly due to LLM-era web scraping. Even image HEAD requests count against the budget.

7. **Seekoff validated the tag-filtering use case** for prison education (exclude security topics) but is archived and doesn't produce ZIMs.

8. **Self-hosted tag-filtered builds are feasible** on modest hardware (comparable to a dev workstation) if the XML pre-filtering step is handled correctly.
