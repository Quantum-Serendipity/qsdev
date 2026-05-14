---
source: https://code.mendhak.com/taking-a-backup-of-stackoverflow/
retrieved: 2026-05-14
---

# Taking a Backup of Stack Overflow

## Overall Size and Format
The complete Stack Overflow data dump is approximately 70 GB when compressed. The schema follows a Microsoft SQL Server database export format, with each table stored as an XML file where individual elements represent database rows.

## Key Tables and Sizes
- **Posts Table**: The largest component at 105 GB uncompressed, containing both questions (PostTypeID=1) and answers (PostTypeID=2). Answers link to questions via the ParentID column.
- **Comments Table**: 28 GB uncompressed, providing supplementary context for posts.

## Data Structure Details
"Each table is stored in the archive as an XML file, each element representing a row in the table." All Stack Exchange network sites follow the same schema, enabling consistent data handling across different Q&A communities.

## Schema Documentation
The official schema documentation is referenced in a meta post, though the article observes "there is surprisingly little official documentation available on how to work with it."

## Community Tools
Scripts exist to convert the XML data dump into formats compatible with PostgreSQL, MySQL, and other databases, developed by community members rather than Stack Exchange Inc.
