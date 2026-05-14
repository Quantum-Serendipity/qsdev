# Sotoki PostsDatabase Source Code
- **Source URL**: https://raw.githubusercontent.com/openzim/sotoki/main/src/sotoki/utils/database/posts.py
- **Retrieved**: 2026-05-14

---

## PostsDatabase Class — Redis Storage for Posts

### Storage Patterns:
- `questions` sorted set: PostId ordered by Score (for homepage)
- `T:{tag}` sorted set: PostId ordered by Score per tag (for tag pages)
- `Q:{id}` compressed JSON: CreationDate, OwnerName, has_accepted, nb_answers, tag_ids
- `QD:{id}` compressed JSON: Title, Excerpt (can reach 9GB for full SO)

### Key Method: record_question(post)
This is where tag-to-post association happens:
```python
for tag in post.get("Tags", []):
    shared.database.pipe.zadd(
        shared.tagsdatabase.tag_key(tag),
        mapping={post["Id"]: post["Score"]},
        nx=True,
    )
```
Each tag gets its own Redis sorted set with all associated question IDs.

### Significance for Tag Filtering:
The tag-post relationship is already tracked in Redis during the first pass. A tag filter would need to intercept at the `record_question()` level (first pass) and at the `PostGenerator.processor()` level (second pass) to skip posts whose tags don't match the filter criteria.
