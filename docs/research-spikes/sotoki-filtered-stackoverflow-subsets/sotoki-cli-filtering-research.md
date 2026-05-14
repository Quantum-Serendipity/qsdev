# Sotoki CLI & Tag-Based Filtering Research

## Executive Summary

**Sotoki does not support tag-based filtering.** There is no native capability to limit ZIM output to questions matching specific StackExchange tags. The tool processes the entire SE data dump for a given domain and includes all (non-deleted) posts in the output. An open GitHub issue (#287) requests exactly this feature, but it has received no maintainer response or development activity since July 2023.

Adding tag-based filtering is architecturally feasible and moderately complex. The codebase already parses tags during processing and maintains tag-to-post mappings in Redis. The main work would be adding filter predicates at two points in the pipeline, plus handling the cascade effects on user profiles, tag pages, and statistics.

---

## Complete CLI Flags Reference

Source: `src/sotoki/entrypoint.py` (argparse definitions) and `offliner-definition.json` (zimfarm parameter schema).

### Required Flags

| Flag | Description |
|------|-------------|
| `-d, --domain` | StackExchange domain to scrape (e.g., `stackoverflow.com`, `sports.stackexchange.com`) |
| `--mirror` | URL from which to download compressed XML dumps |
| `--title` | Title for the ZIM file (max 30 chars) |
| `--description` | Description for the ZIM file (max 80 chars) |

### Metadata Flags

| Flag | Description |
|------|-------------|
| `--name` | ZIM name/identifier (constructed from domain if omitted) |
| `--long-description` | Extended ZIM description (max 4000 chars) |
| `--favicon` | URL/path to ZIM illustration |
| `--creator` | Content creator name (default: "Stack Exchange") |
| `-p, --publisher` | Publisher name (default: "openZIM") |
| `--tags` | **ZIM metadata tags** (semicolon-delimited). These are tags applied to the ZIM file itself for cataloging in Kiwix library — NOT content filtering tags. |

### Content Censorship/Optimization Flags

| Flag | Description |
|------|-------------|
| `--without-images` | Exclude in-post images and user icons. Faster. |
| `--without-user-profiles` | Exclude user profile pages. Faster. |
| `--without-external-links` | Strip external link URLs (keep text). Slower. |
| `--without-unanswered` | Exclude posts with zero answers. Faster. |
| `--without-users-links` | Remove "social" website links entirely. Slower. |
| `--without-names` | Replace real usernames with generated ones. |
| `--censor-words-list` | URL/path to word list for content censorship. Very slow. |

### Advanced/Performance Flags

| Flag | Description |
|------|-------------|
| `--output` | Output folder for ZIM file |
| `--threads` | Concurrency threads (default: 1) |
| `--tmp-dir` | Temp folder for build (default: TMPDIR or CWD) |
| `--zim-file` | Custom ZIM filename |
| `--optimization-cache` | S3 URL for image optimization cache |
| `--redis-url` | Redis connection URL |
| `--debug` | Verbose output |
| `--stats-filename` | Progress JSON file path |
| `--prepare-only` | Download and prepare dumps, then exit |
| `--keep` | Don't remove build folder on start |
| `--keep-redis` | Don't flush Redis on exit |
| `--keep-intermediates` | Keep intermediate files during prepare step |
| `--build-in-tmp` | Use tmp-dir directly as workdir |
| `--defrag-redis` | Restart Redis after user cleanup to reclaim memory |
| `--shell` | Drop into IPython shell after init (dev only) |
| `--dev-skip-tags-meta` | Skip tag metadata pass (dev only) |
| `--dev-skip-questions-meta` | Skip questions first-pass (dev only) |
| `--dev-skip-users` | Skip user file reading (dev only) |

---

## Tag Filtering: Does It Exist?

### Answer: No.

After exhaustive review of every source file in sotoki, there is **no mechanism** to filter which StackExchange posts are included in the ZIM based on their tags. Specifically:

1. **No CLI flag** for tag filtering exists. The `--tags` flag is exclusively for ZIM metadata (library cataloging), not content selection.

2. **No configuration file** supports filtering. The `Context` dataclass has no field for include/exclude tag lists.

3. **No environment variable** enables filtering.

4. **No code path** checks post tags against a filter list. The only content-level filter is `--without-unanswered`, which skips zero-answer posts.

5. **The offliner-definition.json** (zimfarm integration schema) confirms the complete parameter set — no tag filtering parameter exists there either.

### The `--tags` Flag Confusion

The `--tags` CLI parameter is a common source of confusion. It accepts a semicolon-delimited list of strings that become **ZIM file metadata** — tags used by Kiwix library for cataloging and discovery. These tags appear in the ZIM file's metadata, not in content selection. The scraper automatically adds tags like `_category:stack_exchange`, `stack_exchange`, `_videos:no`, and `_details:no`. The `--tags` flag lets users add additional catalog labels.

---

## Sotoki's Data Pipeline Architecture

Understanding the pipeline is essential for estimating where tag filtering would be injected.

### Phase 1: Archive Download (`archives.py`)

The `ArchiveManager` downloads a single 7z archive per domain from the configured mirror:
```
{mirror}/{domain}.7z
```
This archive contains the complete SE data dump for that domain. There is no way to download a subset — the dump is all-or-nothing. After extraction, sotoki requires six XML files: `Badges.xml`, `Comments.xml`, `PostLinks.xml`, `Posts.xml`, `Tags.xml`, `Users.xml`. These are merged into `posts_complete.xml` (posts + answers + comments + links).

### Phase 2: Tag Metadata (`tags.py` — TagFinder)

Walks `Tags.xml` via SAX parsing. Each tag row is recorded in Redis with its name, count, excerpt post ID, and wiki post ID. Tags with Count=0 are skipped. This builds the `TagsDatabase` which maps tag names to IDs bidirectionally (using `bidict`).

### Phase 3: Questions Metadata — First Pass (`posts.py` — PostFirstPasser)

Walks `posts_complete.xml` for the first time. For each question:
- Skips deleted posts (`DeletionDate` present)
- Optionally skips unanswered posts (`--without-unanswered`)
- Parses tags from the post (format: `|tag1|tag2|` or `<tag1><tag2>`)
- Records in Redis: question ID in the global `questions` sorted set (scored by votes), question ID in each tag's sorted set (`T:{tag_name}`), and question details (timestamp, owner, accepted status, tag IDs)
- Collects user IDs for later processing

### Phase 4: User Pages (`users.py` — UserGenerator)

Walks `Users.xml`, creates ZIM pages for users who had interactions with recorded posts.

### Phase 5: Questions — Second Pass (`posts.py` — PostGenerator)

Walks `posts_complete.xml` again. This time renders full HTML pages for each question, including comments, answers, and links. Each question gets a ZIM entry at `questions/{id}/{slug}`.

### Phase 6: Tag Pages (`tags.py` — TagGenerator)

For each tag in the database, creates paginated listing pages showing the highest-voted questions for that tag. Also creates an overall tags index page and a `api/tags.json` endpoint.

### Phase 7: Index Pages

Creates the homepage (paginated questions list), user listing pages, and the about page.

### Phase 8: Image Processing

Downloads, optimizes, and stores all referenced images.

---

## GitHub Issues Related to Tag Filtering

### Issue #287: "Support for tag filtering" (OPEN)
- **URL**: https://github.com/openzim/sotoki/issues/287
- **Created**: 2023-07-20 by natamox
- **Status**: Open, no assignees, no comments, no linked PRs
- **Request**: "Because the whole thing is really too big, more than 70 GB. For example, I only want to grab javascript or other tags, how to do it, thank you"
- **Assessment**: This is exactly our use case. The complete lack of response from maintainers (nearly 3 years open) suggests this feature is not on the development roadmap.

### Issue #224: "Case insensitive tag filter?" (CLOSED)
- **URL**: https://github.com/openzim/sotoki/issues/224
- **Created**: 2021-07-05 by kelson42
- **Status**: Closed
- **Context**: This is about the **in-ZIM tag browser UI**, not about filtering during ZIM generation. The user wanted case-insensitive search when browsing tags within an already-generated ZIM file. Not relevant to our question.

### Issue #76: "Put the tag 'stackexchange' to all created ZIM file"
- **URL**: https://github.com/openzim/sotoki/issues/76
- **Context**: About adding ZIM metadata tags automatically. Resulted in the automatic `_category:stack_exchange` tags. Not relevant to content filtering.

---

## Complexity Estimate: Adding Tag Filtering

### What Would Need to Change

Tag filtering would require modifications at multiple points in the pipeline. Here is a layer-by-layer breakdown:

#### 1. CLI & Context (Small — ~30 lines)

Add two new CLI arguments and Context fields:
```python
# In entrypoint.py
censored.add_argument(
    "--include-tags",
    help="Only include posts matching at least one of these tags "
         "(comma-separated). All other posts excluded.",
    type=lambda x: [t.strip() for t in x.split(",")],
    dest="include_tags",
)
censored.add_argument(
    "--exclude-tags",
    help="Exclude posts matching any of these tags (comma-separated).",
    type=lambda x: [t.strip() for t in x.split(",")],
    dest="exclude_tags",
)

# In context.py
include_tags: list[str] = field(default_factory=list)
exclude_tags: list[str] = field(default_factory=list)
```

#### 2. First Pass Filter — PostFirstPasser.processor() (Small — ~10 lines)

After `harmonize_post(item)` parses the tags, add a filter check:
```python
def processor(self, item):
    # ... existing deleted/unanswered checks ...
    harmonize_post(item)
    
    # NEW: tag filtering
    if context.include_tags:
        if not set(item["Tags"]) & set(context.include_tags):
            self.release()
            return
    if context.exclude_tags:
        if set(item["Tags"]) & set(context.exclude_tags):
            self.release()
            return
    
    # ... rest of existing code ...
```

#### 3. Second Pass Filter — PostGenerator.processor() (Small — ~10 lines)

Same filter logic, applied before rendering:
```python
def processor(self, item):
    # ... existing deleted/unanswered checks ...
    harmonize_post(post)
    
    # NEW: tag filtering (same as first pass)
    if context.include_tags:
        if not set(post["Tags"]) & set(context.include_tags):
            self.release()
            return
    if context.exclude_tags:
        if set(post["Tags"]) & set(context.exclude_tags):
            self.release()
            return
    
    # ... rest of existing code ...
```

#### 4. Tag Page Generation — TagGenerator.run() (Medium — ~15 lines)

When `--include-tags` is set, only generate tag pages for included tags (or tags that have remaining posts). When `--exclude-tags` is set, skip generating pages for excluded tags that are now empty. The `TagFinder` in Phase 2 could also be filtered to avoid recording unwanted tags at all, but this is optional since the tag metadata pass is fast.

#### 5. User Page Handling (Medium — ~20 lines)

Users are currently recorded based on which posts reference them. If posts are filtered out, fewer users will be in the set, and this should cascade naturally since `PostFirstPasser` only adds user IDs for posts that pass filters. However, the user count and progress tracking would need adjustment.

#### 6. Statistics & Progress (Small — ~10 lines)

The `total_questions` count (used for progress bars) comes from counting questions in `posts_complete.xml` during archive preparation. With filtering, the actual number of processed questions would be less than the total. Progress tracking would need a post-filtering count or just accept inaccurate progress percentages.

#### 7. Edge Cases (Medium complexity)

- **Answers referencing filtered-out questions**: Not an issue — answers are nested under their parent question in `posts_complete.xml`, so filtering a question removes its answers automatically.
- **Cross-links between filtered and unfiltered posts**: The `PostLinks` (linked/duplicate) handling in `PostsWalker` could reference posts that were filtered out. These links would need graceful handling (skip the link or show a "not included" message).
- **Tag pages with partial data**: If `--include-tags python,django` is used, the `django` tag page would only show questions that also have the `python` tag (since multi-tag questions exist). This might be surprising. Need to decide: does `--include-tags` mean "include any question that has ANY of these tags" or "only these tag pages"?
- **Download size unchanged**: The full dump must still be downloaded and extracted. Filtering only reduces the ZIM output size, not the input size or processing working set (Redis usage during first pass).

### Overall Complexity Assessment

| Component | Lines Changed | Difficulty |
|-----------|--------------|------------|
| CLI/Context | ~30 | Easy |
| First pass filter | ~10 | Easy |
| Second pass filter | ~10 | Easy |
| Tag page generation | ~15 | Easy |
| User cascade | ~20 | Medium |
| Stats/progress | ~10 | Easy |
| Edge cases (links) | ~30 | Medium |
| Testing | ~100-200 | Medium |
| **Total** | **~225-325** | **Medium** |

Estimated effort: **2-4 days** for a developer familiar with the codebase, including testing. The core filter is trivially simple (check tag intersection at two points). The complexity is in the cascade effects and edge cases.

### Alternative: Pre-filter the XML Dump

Instead of modifying sotoki, one could pre-process the `posts_complete.xml` to remove posts not matching desired tags before running sotoki. This is hacky but avoids forking:

1. Extract the 7z dump
2. Parse `Posts.xml`, filter by tags, write filtered version
3. Rebuild `posts_complete.xml` from filtered posts
4. Run sotoki with `--build-in-tmp` pointing to the pre-filtered directory

This approach is fragile (must match sotoki's internal XML format expectations) and still requires downloading the full dump, but avoids code changes to sotoki itself.

---

## Nearest Existing Capabilities

The closest thing sotoki has to content filtering:

1. **`--without-unanswered`**: Excludes posts with zero answers. This is the only post-level content filter. Its implementation pattern (check in `PostFirstPasser.processor()` and `PostGenerator.processor()`) is exactly where tag filtering would be added.

2. **`--domain` selection**: You can choose which SE site to scrape. Smaller sites (e.g., `sports.stackexchange.com`) produce manageable ZIM files. But for Stack Overflow itself (70+ GB), this doesn't help.

3. **`--prepare-only`**: Downloads and prepares dumps without generating ZIM. Could be used to get the raw XML for external pre-processing.

---

## Sources

All source files saved to `docs/` directory:

- `docs/sotoki-readme-full.md` — Project README
- `docs/sotoki-entrypoint-source.md` — CLI argument definitions (entrypoint.py)
- `docs/sotoki-tags-source.md` — Tag processing code (tags.py)
- `docs/sotoki-posts-source.md` — Post processing code analysis (posts.py)
- `docs/sotoki-scraper-source.md` — Main scraper orchestrator analysis (scraper.py)
- `docs/sotoki-context-source.md` — Configuration dataclass analysis (context.py)
- `docs/sotoki-archives-source.md` — Archive download/extraction analysis (archives.py)
- `docs/sotoki-posts-database-source.md` — Redis post storage analysis (utils/database/posts.py)
- `docs/sotoki-offliner-definition.md` — Zimfarm parameter schema
- `docs/sotoki-changelog-analysis.md` — Version history analysis
- `docs/sotoki-issue-287-tag-filtering.md` — Tag filtering feature request (open)
- `docs/sotoki-issue-224-case-insensitive-tags.md` — Case-insensitive tag browser (closed, unrelated)
