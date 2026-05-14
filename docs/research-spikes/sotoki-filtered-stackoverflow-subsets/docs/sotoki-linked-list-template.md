---
source: https://raw.githubusercontent.com/openzim/sotoki/main/src/sotoki/templates/linked_list.html
retrieved: 2026-05-14
note: AI-summarized by WebFetch; key rendering logic extracted
---

# Sotoki linked_list.html Template

Renders the "Linked" questions sidebar on question pages.

## Rendering Logic

For each item in `list` (which is `post.links.linked`):

1. Shows vote score: `{{ item.Id|question_score }}` with conditional styling for accepted answers
2. Creates a link: `{{ to_root }}questions/{{ item.Id }}/{{ item.Name|slugify }}`
3. Displays question title: `{{ item.Name }}`

## Critical Finding for Cross-Tag Link Integrity

The link URL is constructed directly from the linked question's ID and name:
```
{{ to_root }}questions/{{ item.Id }}/{{ item.Name|slugify }}
```

This means the link is rendered BEFORE the rewriter processes it. The link target is
a ZIM-internal path (`questions/{id}/{slug}`). If that question was excluded from the
tag-filtered subset, this link will point to a non-existent ZIM entry.

**Important**: This is a different path than body text links. Body text links go through
the Rewriter's `rewrite_relative_link()` which does `del link.attrs["href"]` when the
target doesn't exist. But sidebar links in the template are rendered directly from
PostLinks data — they bypass the rewriter's missing-target check.

## Duplicate Links

The question.html template only renders `post.links.linked` (LinkTypeId=1) in the sidebar.
The `post.links.duplicate` section was NOT found in the question template — duplicates may
be handled differently (possibly in post_layout.html or via a banner).
