---
source: https://raw.githubusercontent.com/openzim/sotoki/main/src/sotoki/utils/html.py
retrieved: 2026-05-14
note: Content was AI-summarized by WebFetch; key mechanisms extracted but not verbatim source
---

# Sotoki HTML Rewriter (utils/html.py)

## Key Mechanisms

### 1. Internal Link Rewriting Strategy

The `Rewriter` class uses regex patterns to identify and transform internal Stack Exchange paths:

- **Question links**: Converts formats like `q/{id}`, `questions/{id}/{slug}`, or `questions/{id}/slug/{aid}` into normalized `questions/{id}/{slug}` structure
- **Answer links**: Transforms `a/{id}` patterns while preserving fragment anchors
- **User profiles**: Rewrites `users/{id}/slug` paths using retrieved user names
- **Tags**: Converts tag IDs to tag names via `questions/tagged/{name}`

### 2. Missing Target Handling (CRITICAL)

When a link target doesn't exist in the ZIM:

**The code removes the `href` attribute entirely: `del link.attrs["href"]`**. This occurs when:
- Question title retrieval fails
- User data is unavailable
- Tag name cannot be resolved
- The path isn't in supported routes

### 3. Transformation Process

The `rewrite_relative_link()` method:

1. Parses the URI path and normalizes folder-walking prefixes
2. Matches against specific regex patterns for content types
3. Retrieves metadata (titles, names) from shared databases
4. Reconstructs paths using `rebuild_uri()` with normalized slugs
5. Marks unsupported paths as external (non-offlined content)

### 4. Rewriter Filters for Jinja2

The renderer registers these as Jinja2 filters:
- `rewrote` - rewrites full HTML content
- `rewrote_comment` - rewrites comment HTML
- `rewrote_string` - rewrites plain text strings
