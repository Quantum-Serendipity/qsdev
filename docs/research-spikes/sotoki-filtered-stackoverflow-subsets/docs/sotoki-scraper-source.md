# Sotoki scraper.py Source Code
- **Source URL**: https://raw.githubusercontent.com/openzim/sotoki/main/src/sotoki/scraper.py
- **Retrieved**: 2026-05-14

---

## StackExchangeToZim Class — Main Orchestrator

### Data Pipeline (in order):
1. `__init__()` — Site details extraction, language detection, directory setup
2. `run()` — Downloads dumps via ArchiveManager, sets up Redis databases, Creator
3. `start()` — Main execution:
   a. `add_illustrations()` / `add_assets()` — Static assets
   b. `process_tags_metadata()` — Walk Tags.xml, record in Redis (TagFinder)
   c. `process_questions_metadata()` — Walk posts_complete.xml first pass (PostFirstPasser)
   d. `process_indiv_users_pages()` — Walk Users.xml, create user pages (UserGenerator)
   e. `process_questions()` — Walk posts_complete.xml second pass (PostGenerator)
   f. `process_tags()` — Create tag pages (TagGenerator)
   g. `process_pages_lists()` — Create index/list pages
   h. `shared.imager.process_images()` — Image optimization

### Concurrency:
- shared.executor: 3 workers for HTML rendering
- shared.img_executor: 10 workers for image processing

### Filtering:
- Only existing filter: `context.without_unanswered` (skips zero-answer posts)
- Content censorship: without_images, without_external_links, without_user_profiles, etc.
- **NO tag-based content filtering anywhere in the pipeline**
