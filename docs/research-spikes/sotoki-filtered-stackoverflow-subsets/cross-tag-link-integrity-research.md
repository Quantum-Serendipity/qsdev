# Cross-Tag Link Integrity in Tag-Filtered Stack Overflow ZIM Subsets

## Overview

When building a tag-filtered Stack Overflow ZIM file (e.g., python-only content), any link that targets a question excluded from the subset becomes broken. This report analyzes the three link pathways in sotoki's rendering pipeline, quantifies the scope of the problem, documents what happens when links break in Kiwix, and evaluates mitigation strategies.

---

## 1. How Sotoki Renders Cross-Question Links

Sotoki produces three distinct categories of cross-question links, each with different rendering pathways and different broken-link behavior.

### 1a. Sidebar "Linked" Section (from PostLinks.xml)

**Source**: `PostLinks.xml` entries with `LinkTypeId=1` (linked) are parsed in `posts.py`'s `PostsWalker.startElement()`:

```python
if name == "link":
    pipe = {"1": "linked", "3": "duplicate"}.get(attrs["LinkTypeId"])
    if pipe:
        self.post["links"][pipe].append(
            {"Id": int(attrs["RelatedPostId"]), "Name": attrs["PostName"]}
        )
```

**Rendering**: The `linked_list.html` template renders these directly into sidebar HTML:

```jinja2
<a href="{{ to_root }}questions/{{ item.Id }}/{{ item.Name|slugify }}">{{ item.Name }}</a>
```

**Critical finding**: These links are constructed directly from PostLinks data and **bypass the rewriter's missing-target check entirely**. The URL is assembled from the linked question's ID and slugified name. If that question was excluded from the tag-filtered subset, this produces a link to a non-existent ZIM entry.

**Broken-link behavior**: The link renders normally in the HTML. When clicked, the user hits Kiwix's "not found" error page (see Section 2).

### 1b. Inline Body Links (from question/answer HTML)

**Source**: Links embedded in question and answer body text (e.g., "see also [this question](/questions/12345/...)").

**Rendering**: Body HTML passes through the `Rewriter.rewrite_links()` method in `utils/html.py`. The rewriter:

1. Detects internal SO links via domain regex matching
2. Normalizes the URL path
3. Looks up the question title in `shared.postsdatabase.get_question_title_desc()`
4. If the lookup **succeeds**: rewrites the URL to the ZIM-internal path
5. If the lookup **fails**: executes `del link.attrs["href"]` — removing the href entirely

**Broken-link behavior**: The link text remains visible but becomes unclickable plain text. This is a **graceful degradation** — the user sees the text but cannot navigate to the missing page. No 404 error occurs.

**Key code path** (from `rewrite_relative_link()`):

```python
qid_m = self.qid_re.match(uri_path)
if qid_m:
    qid = qid_m.groupdict().get("post_id")
    title = shared.postsdatabase.get_question_title_desc(int(qid))["title"]
    if not title:
        del link.attrs["href"]    # <-- graceful removal
    else:
        link["href"] = rebuild_uri(...)
```

### 1c. Duplicate Markers (from PostLinks.xml, LinkTypeId=3)

**Source**: `PostLinks.xml` entries with `LinkTypeId=3` (duplicate) are stored in `post["links"]["duplicate"]`.

**Rendering**: The `question.html` template only renders `post.links.linked` in the sidebar. The duplicate data (`post.links.duplicate`) is **not rendered in the sidebar**. It is passed in the post dict but its rendering location was not found in the examined templates (`question.html`, `post_layout.html`, `linked_list.html`). It may be rendered in the question header area as a "marked as duplicate" banner, but this was not confirmed in the source inspection.

**Impact**: Lower concern — duplicate markers are less common than linked posts, and if they render as a banner pointing to the canonical question, the same broken-link issue applies but affects fewer pages.

---

## 2. What Happens with Broken Links in Kiwix

### kiwix-serve (web-based reader)

When a user clicks a link pointing to a non-existent entry in a ZIM file, kiwix-serve returns:

> **"The requested URL '/path/to/missing/page' was not found on this server."**

This is a plain-text error page. Since late 2021 (issue #430, PR #465 in libkiwix), the error page includes a search link offering to search for the missing content within the ZIM. Before that enhancement, it was a dead end.

### kiwix-desktop / kiwix-android

Desktop and mobile Kiwix apps use a `zim://` custom URL scheme for internal links. When a target entry doesn't exist, the behavior varies:
- Some implementations show `about:blank`
- Others show a "Not Found" error page
- The enhanced 404 with search link applies to kiwix-serve; native apps may or may not have equivalent functionality

### User Experience Impact

A broken sidebar link creates a **hard failure**: the user clicks, expects a related question, and gets an error page. This is significantly worse UX than no link at all. The inline body link degradation (href removal) is much less disruptive because the text remains readable.

---

## 3. Estimated Scope of the Problem

### PostLinks Volume

- **6.55 million** PostLinks rows in the SO data dump (April 2024)
- Of ~24.3 million questions total on Stack Overflow
- PostLinks include both Linked (type 1) and Duplicate (type 3) entries
- Links are bidirectional in the sidebar display: if question A links to B, both A and B show the link

### Inline Body Links Volume

- **11.9 million** external links shared on Stack Overflow (per ICSE 2022 study)
- **82.5%** of link-sharing activities share external resources, implying **~17.5% are internal SO-to-SO links**
- Estimated **~2.5 million** inline internal link instances in body text
- Note: this is independent of PostLinks.xml — body links can exist without PostLinks entries and vice versa

### Cross-Tag Link Percentage (Estimated)

No published study directly measures what fraction of PostLinks cross tag boundaries. However, we can reason from the structure:

**Lower bound estimate (30-40% cross-tag)**:
- Questions on SO have 1-5 tags. The median is ~3 tags per question.
- "Linked" relationships arise when someone posts a link to another question in an answer/comment. These are topically related but often adjacent (e.g., a Python question links to a Linux question about file permissions, or a React question links to a JavaScript question about closures).
- Duplicate links (type 3) are almost always same-tag because duplicates address the same topic.
- For Linked (type 1), approximately 30-50% of links likely cross primary-tag boundaries based on tag network co-occurrence patterns.

**Upper bound reasoning (50-60% for narrow filters)**:
- A tag like `python` (~2.2M questions) is so broad that many linked questions share the python tag.
- But a narrow tag like `flask` (~50K questions) links heavily to `python`, `html`, `javascript`, and `sql` questions that may not carry the `flask` tag.
- The narrower the tag filter, the higher the percentage of broken links.

**Working estimate**: For a broad tag like `python`, expect **30-40% of sidebar links to be broken** (pointing to questions not tagged `python`). For a narrow tag like `flask`, expect **50-70% broken**. For multi-tag filters like `python+django`, the rate drops somewhat but remains significant.

### Per-Question Impact

- Not all questions have PostLinks. Rough estimate: ~15-20% of questions have at least one sidebar link.
- Of those, 30-70% of their sidebar links (depending on tag breadth) would be broken.
- Inline body links are handled gracefully (href removed), so they add no UX damage.
- **Net impact**: ~5-15% of questions in a tag-filtered subset would display at least one broken sidebar link.

---

## 4. Mitigation Strategies

### Strategy A: Transitive Closure — Include Linked Questions

**Approach**: When building a tag-filtered subset, include not only questions matching the tag filter but also all questions linked from matching questions (1-hop transitive closure).

**Pros**:
- Zero broken sidebar links (for 1-hop)
- Preserves the full "linked questions" UX
- Linked questions are usually topically adjacent, so they add value

**Cons**:
- **Significant size increase**: If 15-20% of included questions have sidebar links, and 30-40% of those links are cross-tag, including linked questions could add 5-10% more questions. But those included questions also have their OWN links, creating a second hop. Without a depth limit, this snowballs.
- **1-hop is tractable, 2+ hops is not**: Limiting to 1-hop keeps size manageable but leaves second-order links broken.
- **Complicates filtering pipeline**: Must process PostLinks.xml in addition to Posts.xml during pre-filtering, requiring an additional pass.
- **Inconsistency**: Included-by-link questions won't appear in tag-based browsing/search, creating a confusing discoverability gap.

**Recommendation**: Not recommended as the primary strategy due to size unpredictability and the snowball effect. Could work as an optional enhancement flag.

### Strategy B: Remove Broken Sidebar Links During Build

**Approach**: Modify sotoki (or a pre-processing step) to validate sidebar links against the set of included questions, dropping any links that point to excluded questions.

**Pros**:
- Clean UX — no broken links, no error pages
- No size increase
- Straightforward to implement: build a set of included question IDs, filter `post["links"]["linked"]` before template rendering

**Cons**:
- Some questions lose their entire "Linked" sidebar (especially narrow-tag subsets)
- Loses cross-topic context that can be valuable

**Implementation**: ~20 lines of code in `PostGenerator.processor()` or `PostsWalker`:
```python
included_ids = set(...)  # populated during first pass
post["links"]["linked"] = [
    link for link in post["links"]["linked"]
    if link["Id"] in included_ids
]
```

**Recommendation**: **Best primary strategy.** Clean, predictable, minimal implementation cost. The lost links are a minor information loss compared to the UX damage of broken links.

### Strategy C: Rewrite Broken Links to a Notice Page

**Approach**: Instead of removing broken sidebar links, rewrite them to point to a static "This question is not included in this subset" page within the ZIM, optionally showing the question title and suggesting the user search for it online.

**Pros**:
- User sees that related content exists but wasn't included
- Preserves information about the link relationship
- Educates the user about the subset's limitations
- Single static page, minimal size overhead

**Cons**:
- More complex implementation than Strategy B
- Still a somewhat jarring UX (click, get a notice instead of content)
- Adds a page that serves no direct knowledge purpose

**Implementation**: Add one static HTML page to the ZIM (`_not_included.html`), rewrite broken sidebar link hrefs to point to it with a query parameter for the title.

**Recommendation**: Good enhancement on top of Strategy B, but adds complexity. Worth considering for v2.

### Strategy D: Accept Broken Links

**Approach**: Do nothing. Let sidebar links break and rely on Kiwix's 404+search page.

**Pros**:
- Zero implementation effort
- The enhanced 404 page (with search) partially mitigates the dead-end problem

**Cons**:
- Poor UX — 30-70% of sidebar links lead to error pages depending on tag filter breadth
- Undermines user trust in the subset's quality
- 5-15% of questions affected

**Recommendation**: Not acceptable for a production-quality tool. The error rate is too high for sidebar links, which are a prominent UI element.

### Strategy E: Hybrid — Filter Links + 1-Hop for High-Value Links

**Approach**: Remove most broken sidebar links (Strategy B), but include the target questions for a subset of high-value links: specifically, duplicate targets (LinkTypeId=3), since these represent canonical answers that the user should always be able to reach.

**Pros**:
- Duplicate chains remain navigable (critical for UX — "this was marked as duplicate of X" must lead somewhere)
- Most broken links are cleaned up
- Size increase is modest (duplicate links are a minority of PostLinks)

**Cons**:
- More complex than pure Strategy B
- Included-by-duplicate questions still won't appear in tag-based browsing

**Recommendation**: **Best overall strategy.** Combines the clean UX of Strategy B with the critical integrity of duplicate chain preservation.

---

## 5. Recommended Approach

**Primary**: Strategy B + E hybrid.

1. **Filter broken sidebar links**: During the build, remove any `post.links.linked` entries where the target question ID is not in the included set. This eliminates broken Linked sidebar entries.

2. **Include duplicate targets**: When a question in the subset is marked as a duplicate (LinkTypeId=3), include the target question in the subset even if it doesn't match the tag filter. This preserves duplicate chain navigation.

3. **Inline body links already handled**: Sotoki's existing rewriter gracefully degrades inline body links (removes href when target doesn't exist in the database). For a tag-filtered build, this behavior works correctly as-is — excluded questions won't be in the posts database, so their inline links will be silently de-linked.

4. **Future enhancement**: Consider Strategy C (notice page) as a polish item if users report wanting to know about excluded linked content.

**Implementation points in sotoki**:
- PostLinks sidebar filtering: ~20 lines in `posts.py`
- Duplicate target inclusion: ~30 lines in the pre-filtering pass (collect duplicate target IDs, merge into inclusion set)
- Inline body links: no changes needed (rewriter already handles missing targets)
- Total: ~50 lines of new code, well within the estimated 225-325 line tag-filtering implementation

---

## 6. Open Questions

- **Exact cross-tag percentage**: The 30-40% estimate for broad tags is derived from structural reasoning, not measured data. Running a SEDE query or analyzing the data dump would provide a precise number. However, the mitigation strategy (filter broken links) works regardless of the exact percentage.
- **Duplicate rendering**: Where exactly `post.links.duplicate` renders in sotoki's templates was not fully confirmed. If it renders as a header banner with a link to the canonical question, the same broken-link issue applies and duplicate target inclusion (Strategy E) becomes essential.
- **Second-order broken links on included-by-duplicate questions**: If a duplicate target question is included by Strategy E, its own sidebar links may also be broken. This is acceptable — the important thing is that the duplicate chain is navigable, not that every tangential link from the canonical question works.
