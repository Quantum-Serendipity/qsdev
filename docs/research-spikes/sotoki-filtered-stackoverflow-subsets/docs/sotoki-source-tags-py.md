---
source: https://raw.githubusercontent.com/openzim/sotoki/main/src/sotoki/tags.py
retrieved: 2026-05-14
---

# Sotoki Tags.xml Processing

## Fields Read from Tags.xml
- Id
- TagName
- Count
- ExcerptPostId
- WikiPostId

## Tag Processing
TagFinder class processes each tag:
1. Filters unused tags: checks if Count == 0 and skips them
2. Converts Count from string to integer
3. Records valid tags via shared.tagsdatabase.record_tag(tag)

## Wiki and Excerpt Handling
Separate generators handle tag descriptions:
- TagExcerptRecorder reads from posts_excerpt.xml, stores excerpt content
- TagDescriptionRecorder reads from posts_wiki.xml, stores description content
Both use shared.tagsdatabase.tags_details_ids to match post IDs with tag names.

## No Tag Subsetting
Code doesn't filter tags by count threshold beyond zero. Pagination limits apply during page generation: NB_PAGINATED_QUESTIONS_PER_TAG controls question pagination per tag.
