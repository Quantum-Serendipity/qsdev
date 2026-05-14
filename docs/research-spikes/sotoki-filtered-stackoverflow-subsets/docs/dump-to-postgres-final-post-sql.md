---
source: https://raw.githubusercontent.com/Networks-Learning/stackexchange-dump-to-postgres/master/sql/final_post.sql
retrieved: 2026-05-14
---

# Stack Exchange Dump-to-Postgres: final_post.sql

## PostTags Join Table (derived, not in raw dump)
```sql
CREATE TABLE PostTags (
    PostId  int not NULL,
    TagId   int not NULL,
    PRIMARY KEY (PostId, TagId)
);
```

## Lookup/Reference Tables
```sql
CREATE TABLE CloseAsOffTopicReasonTypes (
    Id                      int  PRIMARY KEY,
    IsUniversal             bool NOT NULL,
    MarkdownMini            text NOT NULL,
    CreationDate            timestamp,
    CreationModeratorId     int,
    ApprovalDate            timestamp,
    ApprovalModeratorId     int,
    DeactivationDate        timestamp,
    DeactivationModeratorId int
);

CREATE TABLE PostTypes (
    Id   int  PRIMARY KEY,
    Name text NOT NULL
);

CREATE TABLE FlagTypes (
    Id            int  PRIMARY KEY,
    Name          text NOT NULL,
    Description   text NOT NULL
);

CREATE TABLE PostHistoryTypes (
    Id   int  PRIMARY KEY,
    Name text NOT NULL
);

CREATE TABLE CloseReasonTypes (
    Id          int  PRIMARY KEY,
    Name        text NOT NULL,
    Description text
);

CREATE TABLE VoteTypes (
     Id     int PRIMARY KEY,
     Name   text
);

CREATE TABLE ReviewTaskTypes (
    Id            int  PRIMARY KEY,
    Name          text,
    Description   text
);

CREATE TABLE ReviewTaskResultType (
    Id            int  PRIMARY KEY,
    Name          text,
    Description   text
);

CREATE TABLE PostLinkTypes (
    Id   int  PRIMARY KEY,
    Name text
);
```

Note: PostTags is a DERIVED join table created by the import tool by parsing the Tags string field on Posts. It is NOT present in the raw XML dump.
