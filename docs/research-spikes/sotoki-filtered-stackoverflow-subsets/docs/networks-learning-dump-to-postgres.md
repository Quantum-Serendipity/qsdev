---
source: https://raw.githubusercontent.com/Networks-Learning/stackexchange-dump-to-postgres/master/README.md
retrieved: 2026-05-14
---

# Stack Exchange Dump to Postgres

## Tables handled:
- Badges
- Posts
- Tags
- Users
- Votes
- PostLinks
- PostHistory
- Comments

## Key notes:
- The Body field in Posts table is NOT populated by default. Must use --with-post-body argument.
- tags.xml was missing from the Sept 2011 data dump
- PostTag and UserTagQA tables depend on Tags.xml
- Actual CREATE TABLE statements in sql/final_post.sql and sql/optional_post.sql files
