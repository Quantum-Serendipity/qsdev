---
source: https://raw.githubusercontent.com/openzim/sotoki/main/src/sotoki/posts.py
retrieved: 2026-05-14
note: Content was AI-summarized by WebFetch; key link-handling code extracted
---

# Sotoki posts.py - PostLinks Handling

## PostLinks Processing

In `PostsWalker.startElement()`, links from PostLinks.xml are stored as part of the post object:

```python
if name == "link":
    pipe = {"1": "linked", "3": "duplicate"}.get(attrs["LinkTypeId"])
    if pipe:
        self.post["links"][pipe].append(
            {"Id": int(attrs["RelatedPostId"]), "Name": attrs["PostName"]}
        )
```

Links are categorized as:
- LinkTypeId "1" -> "linked" (related posts)
- LinkTypeId "3" -> "duplicate" (duplicate markers)

Each link stores the RelatedPostId and PostName (title).

## ZIM Entry Creation

The `PostGenerator.processor()` method creates entries:

```python
shared.creator.add_item_for(
    path=path,
    title=shared.rewriter.rewrite_string(post.get("Title")),
    content=post_page,
    mimetype="text/html",
    is_front=True,
)
```

## Redirect Entries for Answers

```python
shared.creator.add_redirect(
    path=f'a/{answer["Id"]}',
    target_path=path,
)
```

## Key Insight

The actual HTML rendering of link sections (related questions, duplicate banners) is delegated to:
- `shared.renderer.get_question(post)` - which uses Jinja2 templates
- The rewriter handles URL transformation within body text
