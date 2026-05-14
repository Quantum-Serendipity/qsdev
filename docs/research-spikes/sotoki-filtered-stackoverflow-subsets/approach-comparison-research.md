# Approach Comparison: Tag-Filtered Stack Overflow ZIM Files

## Executive Summary

Three approaches exist for building tag-filtered Stack Overflow ZIM files. After analyzing development effort, runtime cost, maintenance burden, risk profile, and output quality, **Option B (fork sotoki with tag filtering)** is the recommended primary approach, with **Option A (pre-filter XML dump)** as the fallback. Option C (post-process ZIM) is not viable.

The recommendation is strongly influenced by a finding from the issue #287 discussion: the sotoki maintainer (rgaudin) provided a concrete implementation sketch, explicitly endorsed the feature, and the Kiwix founder (kelson42) called it "a pretty good [feature request]" and suggested "one ZIM per mainstream programming language." The project's CONTRIBUTING.md states "Anybody is welcome to improve Sotoki." This combination of maintainer endorsement, implementation guidance, and open contribution policy makes Option B lower-risk than it would be for a typical fork.

---

## Decision Matrix

| Criterion | Option A: Pre-filter XML | Option B: Fork Sotoki | Option C: Post-process ZIM |
|---|---|---|---|
| **Development effort** | Medium (2-3 days) | Medium (2-4 days) | Very High (1-2 weeks) |
| **Lines of code** | ~300-500 (standalone script) | ~225-325 (across 5-6 files) | ~500+ (dump/parse/filter/repack) |
| **Runtime: first build** | 30-60 min filter + normal sotoki run | Normal sotoki run (filter is inline) | 2+ weeks full build + hours repack |
| **Runtime: subsequent builds** | Same (re-filter each dump) | Same (re-run sotoki with flags) | Same (full rebuild required) |
| **Memory requirement** | ~200 MB filter + normal sotoki | Normal sotoki (negligible filter overhead) | 80+ GB for full SO build |
| **Disk requirement** | Full dump + filtered dump + ZIM output (~150 GB) | Full dump + ZIM output (~120 GB) | Full ZIM (75 GB) + extracted (~200 GB) + repacked |
| **Maintenance burden** | Low (standalone, dump format rarely changes) | Medium (must track upstream sotoki) | N/A (not viable) |
| **Upstream contribution** | Not applicable | High potential (maintainers want this) | Not applicable |
| **Risk: format compatibility** | **High** — must produce XML matching sotoki's internal expectations exactly | **Low** — works within sotoki's own parser | High — must reconstruct navigation, tags, links |
| **Risk: cascade correctness** | **Medium** — must independently handle all 5 tiers (posts, answers, comments, votes, users) | **Low** — sotoki's existing pipeline handles cascades naturally | **Very High** — HTML parsing to find tags, URL rewriting for dead links |
| **Risk: edge cases** | Medium — answer ordering, PostLinks, tag wikis, orphaned references | Low — filter at two points; existing code handles rest | Very High — tag-less navigation, broken related links, missing metadata |
| **Output quality** | Good — sotoki generates clean ZIM from filtered input | **Excellent** — native pipeline, all features work correctly | Poor — broken navigation, dead links, missing tag pages |
| **Multi-tag support** | Easy (filter script parameter) | Easy (CLI flag) | Hard (HTML parsing unreliable) |
| **Reproducibility** | High (deterministic script + sotoki) | **Highest** (single tool invocation) | Low (manual steps, fragile pipeline) |
| **Upstreamable** | No (external tool) | **Yes** (maintainers want this feature) | No |

### Scoring Summary

| Criterion (weight) | Option A | Option B | Option C |
|---|---|---|---|
| Development effort (15%) | 7/10 | 8/10 | 2/10 |
| Runtime cost (20%) | 7/10 | 9/10 | 1/10 |
| Maintenance burden (15%) | 8/10 | 6/10 | 1/10 |
| Risk (25%) | 5/10 | 8/10 | 1/10 |
| Output quality (25%) | 7/10 | 9/10 | 3/10 |
| **Weighted total** | **6.7** | **8.2** | **1.6** |

---

## Detailed Analysis

### Option A: Pre-filter the XML Dump

**How it works**: A standalone Python script streams the SE data dump XML files using `xml.sax`, identifies posts matching target tags, and writes a filtered copy of each XML file. The filtered dump is then fed to unmodified sotoki.

**Strengths**:
- No fork of sotoki required; works with any sotoki version including future releases
- Standalone tool can be useful for other purposes (feeding to Seekoff, database import, etc.)
- Dump XML format is stable; SE has used the same schema for over a decade
- Low maintenance burden — only needs updating if SE changes the dump format

**Weaknesses**:
- Must independently implement the full cascade correctly (5 tiers of dependent data)
- Requires extra disk space for both the original and filtered dumps
- Format risk: sotoki uses `posts_complete.xml` (a merged file it creates internally from Posts.xml + Comments.xml + PostLinks.xml). Pre-filtering must either produce files that merge correctly or produce the merged format directly. Getting this wrong causes silent data loss or sotoki crashes.
- The `posts_complete.xml` merge format is an internal sotoki detail that could change between versions without notice
- Two separate tools to orchestrate (filter script + sotoki), adding operational complexity

**The `posts_complete.xml` problem**: Sotoki's `ArchiveManager` merges Posts.xml, Comments.xml, and PostLinks.xml into a single `posts_complete.xml` after extraction. The SAX walkers then read this merged file, not the original Posts.xml. A pre-filter must either: (a) filter the individual XML files and let sotoki merge them, requiring the filter to correctly handle Comments.xml and PostLinks.xml based on post ID sets from Posts.xml, or (b) filter after sotoki's merge step using `--prepare-only`, which requires running sotoki's download/extraction phase first. Option (a) is cleaner but adds complexity; option (b) couples the filter to sotoki's temp directory layout.

### Option B: Fork Sotoki and Add Tag Filtering

**How it works**: Fork the sotoki repository and add `--include-tags` and `--exclude-tags` CLI flags. Filter predicates are inserted at two points in the pipeline (`PostFirstPasser.processor()` and `PostGenerator.processor()`). Cascade effects on tag pages, user profiles, and statistics are handled within the existing pipeline.

**Strengths**:
- Maintainer-endorsed: rgaudin (sotoki maintainer) provided a concrete implementation sketch in issue #287
- Kiwix founder (kelson42) explicitly wants this feature, suggesting "one ZIM per mainstream programming language"
- CONTRIBUTING.md says "Anybody is welcome to improve Sotoki" — clear path to upstream PR
- Filter operates within sotoki's existing pipeline, so all cascade effects (tag pages, user profiles, statistics, link removal) are handled by existing code
- Existing `--without-unanswered` filter provides an exact pattern to follow
- rgaudin confirmed that link removal for filtered-out questions is already handled: "We already remove links to questions that are not in the DB"
- Single tool invocation: `sotoki --domain stackoverflow.com --include-tags python,django ...`
- No extra disk space for filtered dump copies

**Weaknesses**:
- Fork maintenance: must stay compatible with upstream sotoki (active development — 15+ PRs merged in 2026 alone)
- If upstream PR is accepted, maintenance burden drops to zero; if rejected, ongoing rebase work
- Must learn sotoki's development tooling (hatch, pre-commit, pyright, ruff, black)
- The answer shortcut URL (`a/{aId}`) is a known gap: rgaudin noted "for missing target this would lead to a dead link" — but this is a minor issue affecting only cross-linked answers

**Implementation complexity per rgaudin's sketch**:
1. CLI param to capture wanted tags, parse to list (~30 lines)
2. In `tags.py`, skip if TagName not in the list (~10 lines)
3. In `posts.py` in both passes, filter by tag list (~20 lines)
4. About template should mention tag restriction (~5 lines)

This is a remarkably small changeset for significant functionality. The maintainer's confidence ("I think that should do it") suggests the pipeline is well-designed for this kind of extension.

### Option C: Post-Process the ZIM File

**How it works**: Build the full Stack Overflow ZIM (75 GB), then use zim-tools to extract entries, identify which belong to desired tags, remove unwanted entries, and repack into a smaller ZIM.

**Why it is not viable**:

1. **Requires building the full SO ZIM first**: This is the exact problem we're trying to avoid. The full SO build requires 80+ GB RAM, 500+ GB disk, 2+ weeks of build time, and has been failing for the Kiwix team for 2.5 years. Even if we obtained the existing stale 75 GB ZIM, it's from November 2023.

2. **ZIM tools do not support content-based filtering**: `zimdump` can extract entries by namespace, URL, or index — but not by content. `zimrecreate` copies all entries with no filter options. There is no "filter by tag" capability in any ZIM tool.

3. **Tags are not in ZIM metadata**: The ZIM file contains rendered HTML pages. Tag information is embedded in the HTML content, not in ZIM entry metadata. Filtering requires parsing every HTML page to extract tags — essentially re-implementing the tag parsing that sotoki already does from the XML source.

4. **Navigation would be broken**: The sotoki maintainer explicitly warned against this approach: "since we only bundle the HTML version of questions without metadata, you'd have to parse the HTML of every entry to find the tags... Also, the tag-less navigation would need to be fixed and the related links would not work either." Tag listing pages, pagination, the homepage, and user profile pages would all reference removed content.

5. **URL rewriting nightmare**: Every internal link in every remaining HTML page would need checking and rewriting. Cross-references between questions, user profile links, tag page links, and navigation elements would all potentially point to removed content.

**Verdict**: Option C is eliminated. The sotoki maintainer's own assessment was "You're better off implementing this ticket [Option B], way less work and outcome is clear and solid."

---

## Recommendation: Option B (Fork Sotoki)

### Rationale

1. **Lowest risk**: Filtering within sotoki's pipeline means cascade effects are handled by existing, tested code. The `--without-unanswered` filter is an exact precedent for the implementation pattern.

2. **Highest quality output**: Tag pages, user profiles, statistics, navigation, and link handling all work correctly because they're generated from the filtered dataset by sotoki's own rendering pipeline.

3. **Maintainer alignment**: This is the rare case where a fork has a clear path to becoming an upstream contribution. Both the project founder and the primary maintainer want this feature. A well-implemented PR has a strong chance of acceptance, which would eliminate all maintenance burden.

4. **Operational simplicity**: One tool, one invocation, one output. No intermediate files, no multi-step orchestration.

5. **The implementation is small**: ~100 lines of actual filtering logic (per the maintainer's sketch), plus ~100-200 lines of tests. This is not a major fork divergence — it's a feature addition that the maintainer estimated as "quite easy."

### Why Not Option A?

Option A is solid engineering but has a subtle, significant risk: the `posts_complete.xml` coupling. Sotoki's internal merge of Posts.xml + Comments.xml + PostLinks.xml creates an intermediate format that the pre-filter must either avoid (by filtering the source files separately) or replicate. Either way, the pre-filter becomes coupled to sotoki's internal behavior without being part of its codebase. When sotoki changes its merge logic (as it did in the v2-to-v3 rewrite), the pre-filter breaks silently.

Option A also duplicates work: the pre-filter must implement the same 5-tier cascade logic that sotoki's pipeline already handles naturally when filtering is done inline.

---

## Implementation Sketch: Option B

### Step 1: Fork and Set Up Development Environment

```bash
gh repo fork openzim/sotoki --clone
cd sotoki
hatch shell
hatch run pre-commit install
```

### Step 2: Add CLI Arguments (`src/sotoki/entrypoint.py`)

Add to the `censored` argument group (where `--without-unanswered` lives):

```python
censored.add_argument(
    "--include-tags",
    help="Only include posts with at least one of these tags (comma-separated). "
         "Produces a tag-filtered subset ZIM.",
    type=lambda x: [t.strip().lower() for t in x.split(",")],
    dest="include_tags",
    default=[],
)
censored.add_argument(
    "--exclude-tags",
    help="Exclude posts matching any of these tags (comma-separated).",
    type=lambda x: [t.strip().lower() for t in x.split(",")],
    dest="exclude_tags",
    default=[],
)
```

### Step 3: Add Context Fields (`src/sotoki/context.py`)

```python
include_tags: list[str] = field(default_factory=list)
exclude_tags: list[str] = field(default_factory=list)
```

### Step 4: Filter in Tag Metadata Pass (`src/sotoki/tags.py` — TagFinder)

When `include_tags` is set, skip recording tags not in the include list. This prevents tag pages from being generated for irrelevant tags:

```python
# In TagFinder.processor() or equivalent
if context.include_tags and tag_name.lower() not in context.include_tags:
    return  # Skip this tag entirely
if context.exclude_tags and tag_name.lower() in context.exclude_tags:
    return
```

Note: Also retain tags that co-occur on matching questions (e.g., if filtering for `python`, also retain `django` because python questions carry both tags). The cleaner approach per rgaudin is to skip tag recording here and let the post passes determine which tags actually appear.

### Step 5: Filter in First Pass (`src/sotoki/posts.py` — PostFirstPasser.processor())

After `harmonize_post(item)` parses tags:

```python
post_tags = set(t.lower() for t in item["Tags"])

if context.include_tags:
    if not post_tags & set(context.include_tags):
        self.release()
        return

if context.exclude_tags:
    if post_tags & set(context.exclude_tags):
        self.release()
        return
```

### Step 6: Filter in Second Pass (`src/sotoki/posts.py` — PostGenerator.processor())

Same filter logic, applied before HTML rendering. This is the gate that prevents filtered-out questions from becoming ZIM entries.

### Step 7: Update About Page Template

Add a notice when tag filtering is active:

```
This ZIM contains a filtered subset of Stack Overflow, limited to questions
tagged: {', '.join(context.include_tags)}
```

### Step 8: Add Tests

- Test include-tags filters correctly (matching posts included, non-matching excluded)
- Test exclude-tags filters correctly
- Test combined include + exclude
- Test tag co-occurrence (question with `python` and `django` is included when filtering for `python`)
- Test cascade: answers to filtered-out questions are excluded
- Test tag pages: only tags from included posts appear

### Step 9: Update CHANGELOG.md

Add entry under `[Unreleased]` per contribution guidelines.

### Step 10: Submit PR to Upstream

Reference issue #287. Include the implementation sketch from rgaudin's comment to show alignment with maintainer vision.

---

## Fallback Plan: Option A (Pre-filter XML)

If Option B fails (PR rejected, sotoki internals too complex, or upstream changes make the fork impractical), fall back to Option A.

### When to Trigger Fallback

- Sotoki's codebase has changed significantly from what our research analyzed (e.g., major v4 rewrite)
- Development environment cannot be set up (dependency issues, platform incompatibility)
- After 3+ days of implementation effort with no working prototype
- PR submitted but explicitly rejected by maintainers with no path forward

### Fallback Implementation Sketch

**Architecture**: Standalone Python script using `xml.sax` for streaming XML parsing.

```
se-tag-filter.py --tags python,django --input-dir ./dump/ --output-dir ./filtered/
```

**Pass 1: Collect IDs (Posts.xml)**
- Stream Posts.xml with SAX parser
- For PostTypeId=1 (questions): check Tags field for target tags
- Record matching question IDs in a set
- Record OwnerUserId and LastEditorUserId in a user ID set

**Pass 2: Collect Answer IDs (Posts.xml)**
- Stream Posts.xml again
- For PostTypeId=2 (answers): if ParentId is in question set, record answer ID and user IDs
- For PostTypeId=4,5 (tag wikis): if associated tag is in target set, record post ID

**Pass 3: Filter All Files**
- Posts.xml: write rows where Id is in question set OR answer set OR tag wiki set
- Comments.xml: write rows where PostId is in combined post set
- PostLinks.xml: write rows where PostId OR RelatedPostId is in question set
- Users.xml: write rows where Id is in user set
- Badges.xml: write rows where UserId is in user set
- Tags.xml: write rows for tags that appear on any included question
- Votes.xml: write rows where PostId is in combined post set (OPTIONAL — sotoki doesn't use Votes.xml)
- PostHistory.xml: write rows where PostId is in combined post set (OPTIONAL — sotoki doesn't use PostHistory.xml)

**Memory budget**: ~200 MB for popular tags (storing ID sets as Python integers in sets).

**Key implementation detail**: Output XML must preserve the exact format sotoki expects. Use SAX ContentHandler to pass through the XML declaration, root element, and row elements verbatim, only filtering which `<row>` elements are written. This avoids format mismatches.

**Sotoki invocation with `--prepare-only`**: Use sotoki's `--prepare-only` flag to download and extract the dump, then run the filter on the extracted files, then run sotoki again with `--keep` and `--build-in-tmp` pointing to the filtered directory. This avoids re-downloading the dump.

**Critical note**: Verify whether sotoki reads Posts.xml directly or only `posts_complete.xml` (the merged file). If the latter, the filter must run after sotoki's merge step, or filter the three component files separately so the merge produces correct output.

### Fallback Risk Mitigations

| Risk | Mitigation |
|---|---|
| posts_complete.xml format mismatch | Use `--prepare-only` to let sotoki create the merged file, then filter the merged file |
| Missing answers (answer before parent in XML) | Two-pass approach on Posts.xml eliminates ordering dependency |
| Orphaned comments/votes | Filter by post ID set membership — if post excluded, its comments excluded |
| Tag page generation for absent tags | Filter Tags.xml to only include tags present on filtered questions |
| Progress bar inaccuracy | Accept inaccurate progress or patch sotoki's count after filtering |

---

## Sources

Research reports consulted:
- `sotoki-cli-filtering-research.md` — Comprehensive sotoki CLI and pipeline analysis
- `se-data-dump-structure-research.md` — XML schema, tag architecture, cascade analysis
- `kiwix-build-infrastructure-research.md` — Zimfarm, SO build failures, infrastructure requirements

Web sources saved to `docs/`:
- `docs/sotoki-issue-287-full-discussion.md` — Full issue #287 discussion with maintainer implementation sketch
- `docs/sotoki-contributing-md.md` — Sotoki contribution guidelines
- `docs/zim-tools-readme.md` — ZIM tools overview
- `docs/zimdump-capabilities.md` — zimdump extraction/filtering capabilities
- `docs/zimrecreate-capabilities.md` — zimrecreate functionality (no filtering support)
