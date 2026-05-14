---
source: https://raw.githubusercontent.com/openzim/sotoki/main/src/sotoki/posts.py
retrieved: 2026-05-14
---

# Sotoki Posts.xml Processing

## Processing Pipeline
Uses a two-pass SAX parser approach. FirstPassWalker collects metadata and statistics, PostsWalker processes complete post structures including nested answers and comments.

## Questions vs. Answers
Questions are top-level <post> elements, answers are nested within <answers> blocks. Deleted answers are explicitly skipped: "if 'DeletionDate' in attrs: return". Answers lacking accepted status are still processed but tracked separately.

## Tags Handling (CRITICAL)
Tags are extracted and split using multiple delimiters:
```python
re.split(r"\||><", post["Tags"][1:-1])
```
This accommodates two dump formats:
- pipe-separated: |tag1|tag2|
- angle-bracket-separated: <tag1><tag2>

**No tag-based filtering occurs in this code.**

## Answer-to-Question Linking
Answers are associated through direct nesting in the XML structure. During parsing, answers are appended to a temporary array, then assigned to the post. Answers are sorted by score descending before final processing.

## Redirect Paths
Creates redirect paths mapping individual answer IDs to their parent question, enabling direct answer access while routing to the question page.
