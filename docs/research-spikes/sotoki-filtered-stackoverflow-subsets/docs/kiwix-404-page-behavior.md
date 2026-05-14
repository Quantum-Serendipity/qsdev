---
source: https://github.com/kiwix/kiwix-tools/issues/430
retrieved: 2026-05-14
---

# Kiwix-Serve 404 Page Behavior

## Current Behavior (as of issue filing, Jan 2021)

When a URL doesn't exist in a ZIM file, kiwix-serve displays a plain text error:

> "The requested URL '/wikipedia_en_all_maxi/A/Open Suse' was not found on this server."

No additional functionality, no search, no navigation back.

## Enhancement (Issue #430, closed)

User JensKorte proposed adding a search link to the 404 page:
> "Do you want to search for 'Open Suse' in this zim file?"

Search link format: `/search?content=wikipedia_en_all_maxi&pattern=Open+Suse`

## Resolution

PR #465 in libkiwix addressed this. The 404 page now includes a search link.
Issue was labeled: enhancement, good first issue, kiwix-serve, question.
Assignee: soumyankar. Status: Closed.

## Key Implication for Cross-Tag Links

When a link in a ZIM file points to a non-existent entry, the user sees a "not found" 
error page. This is a hard failure - the user hits a dead end. The enhanced version 
at least offers search, but the UX is still poor for frequent occurrences.
