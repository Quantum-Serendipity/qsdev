---
source: https://raw.githubusercontent.com/openzim/sotoki/main/src/sotoki/utils/html.py
retrieved: 2026-05-14
note: Complete source code retrieved via WebFetch
---

# Sotoki HTML Rewriter - Full Source (utils/html.py)

Complete source code of the Rewriter class that handles all URL rewriting for ZIM output.

See sotoki-html-rewriter-source.md for the analyzed summary. This file contains the verbatim code.

Key finding: When a link target (question, user, tag) cannot be resolved in the database,
the code does `del link.attrs["href"]` — removing the link target entirely while preserving
the link text as unclickable text in the HTML output.

Critical methods:
- `rewrite_relative_link()` - handles all internal SE links, removes href when target missing
- `rewrite_links()` - entry point, classifies links as relative/external
- `rewrite_external_link()` - adds external-link class, optionally removes href

Full code preserved in conversation context for analysis.
