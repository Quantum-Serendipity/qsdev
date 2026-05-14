# Release Automation Tool Comparison

- **Source**: Multiple web searches and articles
- **Retrieval Date**: 2026-05-14

## Tools Compared

### git-cliff
- Generates changelogs from conventional commits
- Handles version bump calculation
- Rust binary, no runtime dependencies
- Focused scope: changelog + version bump only
- Already included in gdev plan (Phase 16)
- https://git-cliff.org/

### semantic-release
- Fully automated version management and package publishing
- Parses git history for feat/fix/breaking changes
- Auto-bumps version, generates changelog, publishes to npm/pypi/etc
- Node.js ecosystem, requires npm/Node runtime
- Heavy plugin system
- 22k+ GitHub stars
- Changelogs produced "rarely meet end users' expectations" (per comparison article)
- https://github.com/semantic-release/semantic-release

### changesets
- "Intentional" release management
- Developers add changeset files during PR (like a CHANGELOG entry)
- Bot comments on PRs missing changesets
- Designed for monorepos (Turborepo, pnpm workspaces)
- Forces quality through explicit change documentation
- "The most powerful part is the pause—especially in Pull Requests"
- https://github.com/changesets/changesets

### release-please (Google)
- GitHub Actions-based release automation
- Creates release PRs automatically from conventional commits
- Multi-language support
- Less opinionated than semantic-release

## Analysis for gdev

gdev already includes:
- **commitlint** — enforces conventional commits (Phase 16)
- **git-cliff** — changelog generation (Phase 16)

The gap between git-cliff and full release automation (semantic-release/changesets) is:
1. **Package publishing** — out of scope for gdev (ecosystem-specific CI step)
2. **Version bumping in source files** — git-cliff already handles this
3. **Release PR creation** — GitHub Actions concern, not gdev concern

**Recommendation**: Keep rejected. git-cliff + commitlint cover the valuable parts (changelog, version discipline). Full release automation (publishing, multi-package coordination) is CI/CD territory.
