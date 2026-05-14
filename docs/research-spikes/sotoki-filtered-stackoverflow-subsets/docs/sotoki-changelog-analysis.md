# Sotoki CHANGELOG.md Analysis
- **Source URL**: https://raw.githubusercontent.com/openzim/sotoki/main/CHANGELOG.md
- **Retrieved**: 2026-05-14

---

## Tag-Related Changes

### v3.0.0
- Breaking: replaced multiple `--tag` with single CSV `--tags` for Zimfarm integration (#351)
- Fixed: Posts tags can be split by `|` or `><` characters (#356)
- Added: auto-remove control characters in Post titles and HTML tags (#418)

### v2.1.0
- Conditional `_pictures:no` in ZIM tags when using `--without-images`

## Content Filtering History
- `--without-user-profiles` link rewriting fix (#247) in v3.0.0
- No tag-based content filtering features have ever been added per the changelog
- All filtering is binary (include/exclude entire categories: images, users, unanswered, external links)

## Key Observation
The `--tags` parameter has always been about ZIM metadata tagging, never about content filtering. No version has ever included tag-based content selection.
