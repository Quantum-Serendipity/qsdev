---
source: https://raw.githubusercontent.com/Networks-Learning/stackexchange-dump-to-postgres/master/sql/optional_post.sql
retrieved: 2026-05-14
---

# Stack Exchange Dump-to-Postgres: optional_post.sql

## Derived Tables (not in raw dump, created by import tool)

```sql
CREATE TABLE UserTagQA (
    UserId      int,
    TagId       int,
    Questions   int,
    Answers     int,
    PRIMARY KEY (UserId, TagId)
);

CREATE TABLE QuestionAnswer (
    QuestionId int,
    AnswerId   int,
    PRIMARY KEY (QuestionId, AnswerId)
);

CREATE TABLE AllPostTags (
    PostId int,
    TagId  int,
    PRIMARY KEY (PostId, TagId)
);
```

## Indexes

```sql
CREATE INDEX usertagqa_questions_idx ON UserTagQA USING btree (Questions);
CREATE INDEX usertagqa_answers_idx ON UserTagQA USING btree (Answers);
CREATE INDEX usertagqa_questions_answers_idx ON UserTagQA USING btree (Questions, Answers);
CREATE INDEX usertagqa_all_qa_posts_idx ON UserTagQA USING btree ((Questions + Answers));
CREATE INDEX posts_id_post_type_id_idx ON Posts USING btree (Id, PostTypeId);
CREATE INDEX posts_id_parent_id_idx ON Posts USING btree (Id, ParentId);
CREATE INDEX posts_id_accepted_answers_id_idx ON Posts USING btree (Id, AcceptedAnswerId);
CREATE INDEX posts_owner_user_id_creation_date_idx ON Posts USING btree (OwnerUserId, CreationDate);
```

Key insight: AllPostTags propagates tags from questions to their answers (both questions AND answers get tag associations). PostTags only maps questions to tags. This is critical for tag-based filtering.
