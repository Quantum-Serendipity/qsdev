---
source: https://stackoverflow.blog/2010/08/01/tag-folksonomy-and-tag-synonyms/
retrieved: 2026-05-14
---

# Tag Folksonomy and Tag Synonyms

## How Tag Synonyms Work
When a synonym is defined, it creates an automatic remapping system. Any question tagged with a synonym (e.g., [js] or [java-script]) is automatically and silently remapped to the canonical tag (e.g., [javascript]) behind the scenes.

## Effect on Existing Posts
The remapping occurs transparently -- users won't see the synonym tag they originally used. Posts automatically display the canonical tag version.

## Hierarchy
One-to-many hierarchy: one primary/canonical tag as destination, multiple variant tags pointing to it.

## Key Requirement
Synonyms must already exist as actual tags on at least one question before being established as official synonyms. Prevents moderators from predicting every possible synonym.

## Tag Synonyms vs. Tag Merges
- Tag merging was purely a moderator function (more permanent administrative action)
- Tag synonyms are community-driven: users propose and vote on relationships
- Community votes on synonym proposals through a dedicated interface visible to higher-reputation users

## Important Implication for Data Dumps
Tag synonyms are NOT included in the data dump XML files. The Tags field in Posts.xml already contains the canonical (post-synonym-resolution) tag names. This means filtering by tag in the dump should capture all posts, even those originally tagged with a synonym.
