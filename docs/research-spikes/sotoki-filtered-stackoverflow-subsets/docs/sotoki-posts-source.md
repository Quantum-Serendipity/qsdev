# Sotoki posts.py Source Code
- **Source URL**: https://raw.githubusercontent.com/openzim/sotoki/main/src/sotoki/posts.py
- **Retrieved**: 2026-05-14

---

(See sotoki-entrypoint-source.md for retrieval context. Full source code was retrieved via WebFetch and analyzed inline. Key findings documented in the research report.)

## Key Classes and Their Roles

### harmonize_post(post)
- Normalizes post data: sets has_accepted, OwnerName, CreationTimestamp
- Splits Tags field using regex: `re.split(r"\||><", post["Tags"][1:-1])`
- Tags come as either `|tag1|tag2|` or `<tag1><tag2>` format from SE dumps

### FirstPassWalker / PostFirstPasser
- SAX parser for posts_complete.xml (first pass)
- Counts answers per question, collects user IDs
- Filters: skips deleted posts (DeletionDate), optionally skips unanswered (context.without_unanswered)
- Calls shared.postsdatabase.record_question(post) for each valid post
- **NO tag-based filtering exists here**

### PostsWalker / PostGenerator
- SAX parser for posts_complete.xml (second pass)
- Full post processing: comments, answers, links
- Same filters: deleted posts, optionally unanswered
- Creates ZIM pages for each question
- **NO tag-based filtering exists here**

### generate_questions_page()
- Creates paginated question index pages
- Reads from Redis sorted set (questions_key)
