---
source: https://raw.githubusercontent.com/openzim/sotoki/main/src/sotoki/entrypoint.py
retrieved: 2026-05-14
type: source-code-extraction
---

# Sotoki CLI Arguments (Full List)

## Required Arguments
- `-d, --domain`: Stack Exchange domain to scrape

## Metadata Arguments
- `--name`: ZIM identifier/filename
- `--title`: ZIM title (max 30 chars, required)
- `--description`: ZIM description (max 80 chars, required)
- `--long-description`: Extended description (max 4000 chars)
- `--favicon`: Icon URL/path
- `--creator`: Content creator name
- `-p, --publisher`: Publisher name
- `--tags`: Semicolon-delimited tag list (ZIM metadata tags, NOT content filtering)

## Content Filtering Arguments
- `--without-images`: Exclude in-post images and user icons
- `--without-user-profiles`: Skip user profile pages
- `--without-external-links`: Strip external URLs (retain link text)
- `--without-unanswered`: Exclude posts with zero answers
- `--without-users-links`: Remove social media links entirely
- `--without-names`: Replace usernames with generated ones
- `--censor-words-list`: Path/URL to word list for removal

## Advanced Arguments
- `--output`: Output folder
- `--threads`: Concurrent thread count
- `--tmp-dir`: Temp folder path
- `--zim-file`: Custom ZIM filename
- `--optimization-cache`: S3 credentials URL
- `--mirror`: XML dump download source (required)
- `--redis-url`: Redis connection string
- `--debug`: Verbose output
- `--stats-filename`: Progress JSON file path
- `--prepare-only`, `--keep`, `--keep-redis`, `--keep-intermediates`, `--build-in-tmp`, `--defrag-redis`, `--shell`: Development/debugging options
- `--dev-skip-tags-meta`, `--dev-skip-questions-meta`, `--dev-skip-users`: Developer skip flags

## KEY FINDING: No Tag-Based Content Filtering

There is NO flag to filter questions by Stack Exchange tags. The `--tags` flag is for ZIM metadata only (how the ZIM is cataloged), not for filtering which questions are included.

The only content reduction options are:
- `--without-images` (reduces size but keeps all questions)
- `--without-unanswered` (removes zero-answer posts)
- `--without-user-profiles` (removes user pages)

To build a tag-filtered subset, you would need to pre-filter the XML dump before feeding it to sotoki, or modify sotoki's source code.
