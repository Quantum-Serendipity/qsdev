---
source: https://raw.githubusercontent.com/openzim/sotoki/main/src/sotoki/renderer.py
retrieved: 2026-05-14
note: AI-summarized by WebFetch; key method extracted
---

# Sotoki renderer.py - get_question()

## Template Rendering

The `get_question()` method passes the entire post dict to the template:

```python
def get_question(self, post: dict):
    """Single question HTML for ZIM"""
    return self.env.get_template("question.html").render(
        body_class="question-page",
        whereis="questions",
        post=post,
        to_root="../../",
        title=shared.rewriter.rewrite_string(post["Title"]),
        **self.global_context,
    )
```

## Key Finding

The entire `post` dictionary (including `post["links"]["linked"]` and `post["links"]["duplicate"]`) 
is passed directly to the template. No filtering or validation of link targets occurs at this stage.

The template then directly renders sidebar links from `post.links.linked` using the linked_list.html 
partial, constructing URLs from the linked question's ID and name WITHOUT checking whether those 
questions exist in the current ZIM build.

This means: for a tag-filtered build that excludes certain questions, the sidebar links will 
contain valid-looking URLs that point to non-existent ZIM entries.
