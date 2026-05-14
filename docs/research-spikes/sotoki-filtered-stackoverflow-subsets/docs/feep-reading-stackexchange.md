---
source: https://search.feep.dev/blog/post/2021-09-04-stackexchange
retrieved: 2026-05-14
---

# Reading All of StackExchange - Feep Blog

## File Structure
The Stack Overflow data dump is distributed as a compressed 16GB file (stackoverflow.com-Posts.7z) containing a single XML file (Posts.xml). The XML structure uses a root <posts> element with individual <row> elements representing posts.

## Post Record Fields
Key attributes found in each <row> element:
- Id: Primary key, used in question URLs
- PostTypeId: Integer referencing post type (1 = question, 2 = answer)
- ParentId: For answers, references the question's Id
- Title: Question title
- Body: HTML content with entity escaping
- Tags: Tag metadata in a specific format
- CreationDate, LastEditDate, LastActivityDate: Timestamps in ISO format
- Score, ViewCount, AnswerCount, CommentCount, FavoriteCount: Engagement metrics
- OwnerUserId, LastEditorUserId: User references

## Tags Field Format (CRITICAL)
Tags are stored as concatenated strings surrounded by angle brackets. For example:
"<c#><floating-point><type-conversion><double><decimal>"

When parsed through XML, the entity escaping (&lt; and &gt;) resolves to literal angle bracket characters.

## Data Relationships
Questions and answers exist as separate rows in the same Posts.xml file. Answers reference their parent question through the ParentId field matching a question's Id.
