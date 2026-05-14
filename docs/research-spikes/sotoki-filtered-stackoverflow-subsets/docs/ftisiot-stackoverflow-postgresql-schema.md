---
source: https://ftisiot.net/posts/stackoverflow-postgresql/
retrieved: 2026-05-14
---

# Load StackOverflow Data in PostgreSQL - Schema

## Users Table
```sql
CREATE TABLE users(
    id int PRIMARY KEY,
    reputation int,
    CreationDate text,
    DisplayName text,
    LastAccessDate timestamp,
    Location text,
    AboutMe text,
    views int,
    UpVotes int,
    DownVotes int,
    AccountId int
);
```

## Posts Table
```sql
CREATE TABLE posts (
    id int PRIMARY KEY,
    PostTypeId int,
    CreationDate timestamp,
    score int,
    viewcount int,
    body text,
    OwnerUserId int,
    LastActivityDate text,
    Title text,
    Tags text,
    AnswerCount int,
    CommentCount int,
    ContentLicense text
);
```

Note: This is a simplified schema. The article references the official meta post for full attribute documentation. Additional tables (Badges, Comments, PostHistory, PostLinks, Tags, Votes) are also defined but details were not fully extracted.
