---
source: https://clickhouse.com/docs/getting-started/example-datasets/stackoverflow
retrieved: 2026-05-14
---

# ClickHouse Stack Overflow Dataset - PostLinks Statistics

## PostLinks Table Schema
- `Id` (UInt64)
- `CreationDate` (DateTime64)
- `PostId` (Int32)
- `RelatedPostId` (Int32)
- `LinkTypeId` (Enum8: 'Linked' = 1, 'Duplicate' = 3)

## Row Counts (as of April 2024)
- **PostLinks: 6.55 million rows**
- Posts: 59.82 million rows
- Votes: 238.98 million rows
- Comments: 90.38 million rows
- Users: 22.48 million rows
- Badges: 51.29 million rows
- PostHistory: 160.79 million rows

## Key Derived Statistics
- ~24.3 million questions (PostTypeId=1, estimated from total posts)
- 6.55M PostLinks across ~24.3M questions = ~27% of questions have at least one PostLink
  (upper bound; many questions have multiple links, so actual % with any link is lower)
- PostLinks are bidirectional in sidebar display, so effective link density is higher
