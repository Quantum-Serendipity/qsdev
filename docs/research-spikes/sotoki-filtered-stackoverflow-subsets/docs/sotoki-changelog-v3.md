---
source: https://raw.githubusercontent.com/openzim/sotoki/main/CHANGELOG.md
retrieved: 2026-05-14
type: changelog
---

# Sotoki Changelog (v3.x)

## Version 3.0.2 (2025-12-22)
- Fixed header extraction for non-StackOverflow and non-StackExchange domains

## Version 3.0.1 (2025-12-18)
- Fixed scraper looping over failing URLs forever
- Corrected header content extraction for StackOverflow domains
- Added CSS file rewriting functionality

## Version 3.0.0 (2025-12-12)
- Upgraded to Python 3.14 and Debian bookworm
- **Breaking change**: Replace usage of multiple `--tag` with single CSV `--tags` for Zimfarm integration
  - NOTE: This is about ZIM METADATA tags, not content filtering by SO question tags
- Enhanced image download logic to better respect upstream servers
- Fixed post tag processing, SVG support, image handling
- Various display fixes for comments, user info, vote counts

## Key Insight
The `--tags` flag change was purely a CLI interface refactor for zimfarm integration, not the addition of content filtering. There remains NO way to filter which SO questions are included based on their Stack Exchange tags.
