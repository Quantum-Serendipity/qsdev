---
source: https://clickhouse.com/docs/getting-started/example-datasets/stackoverflow
retrieved: 2026-05-14
---

# ClickHouse Stack Overflow Dataset - Complete Schema and Sizes

## Posts Table (59.82M rows, 38.07 GB)
- Id (Int32), PostTypeId (Enum8), AcceptedAnswerId (UInt32)
- CreationDate (DateTime64), Score (Int32), ViewCount (UInt32)
- Body (String), OwnerUserId (Int32), OwnerDisplayName (String)
- LastEditorUserId (Int32), LastEditorDisplayName (String)
- LastEditDate (DateTime64), LastActivityDate (DateTime64)
- Title (String), Tags (String), AnswerCount (UInt16)
- CommentCount (UInt8), FavoriteCount (UInt8)
- ContentLicense (LowCardinality String), ParentId (String)
- CommunityOwnedDate (DateTime64), ClosedDate (DateTime64)

## Votes Table (238.98M rows, 2.13 GB)
- Id (UInt32), PostId (Int32), VoteTypeId (UInt8)
- CreationDate (DateTime64), UserId (Int32), BountyAmount (UInt8)

## Comments Table (90.38M rows, 11.14 GB)
- Id (UInt32), PostId (UInt32), Score (UInt16), Text (String)
- CreationDate (DateTime64), UserId (Int32), UserDisplayName (LowCardinality String)

## Users Table (22.48M rows, 1.36 GB)
- Id (Int32), Reputation (LowCardinality String)
- CreationDate (DateTime64), DisplayName (String)
- LastAccessDate (DateTime64), AboutMe (String)
- Views (UInt32), UpVotes (UInt32), DownVotes (UInt32)
- WebsiteUrl (String), Location (LowCardinality String), AccountId (Int32)

## Badges Table (51.29M rows, 797.05 MB)
- Id (UInt32), UserId (Int32), Name (LowCardinality String)
- Date (DateTime64), Class (Enum8), TagBased (Bool)

## PostLinks Table (6.55M rows, 129.70 MB)
- Id (UInt64), CreationDate (DateTime64), PostId (Int32)
- RelatedPostId (Int32), LinkTypeId (Enum8)

## PostHistory Table (160.79M rows, 67.08 GB)
- Id (UInt64), PostHistoryTypeId (UInt8), PostId (Int32)
- RevisionGUID (String), CreationDate (DateTime64), UserId (Int32)
- Text (String), ContentLicense (LowCardinality String)
- Comment (String), UserDisplayName (String)

## Tag Storage Format
Tags are pipe-delimited strings. ClickHouse extracts them with:
arrayJoin(arrayFilter(t -> (t != ''), splitByChar('|', Tags)))

Note: The pipe delimiter suggests ClickHouse pre-processes the angle-bracket format from the raw XML into pipe-delimited during import.

## Total Dataset
~630M rows across all tables, ~120 GB total (ClickHouse compressed format).
