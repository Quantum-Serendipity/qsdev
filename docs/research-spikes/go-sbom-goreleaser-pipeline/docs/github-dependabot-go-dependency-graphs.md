# Dependabot-based Dependency Graphs for Go

- **Source URL**: https://github.blog/changelog/2025-12-09-dependabot-dgs-for-go/
- **Retrieved**: 2026-05-15

---

## What Changed

Go projects now receive more comprehensive and precise transitive dependency information in their dependency graphs and Software Bill of Materials (SBOMs).

## Why Dynamic Resolution Matters for Go

Go resolves dependency versions dynamically, getting an accurate picture of a project's dependencies cannot rely on static parsing. This fundamental characteristic of Go's dependency system necessitates a different approach than what works for other ecosystems.

## How the New Approach Works

When a commit modifies a project's `go.mod` file, GitHub triggers a specialized Dependabot job that constructs a dependency snapshot and transmits it via the Dependency Submission API. This mirrors the autosubmission process used for other language ecosystems.

## Key Advantages

- Does not consume GitHub Actions minutes
- Supports organization-level configurations for private package registries established for Dependabot

## Migration Notes

No breaking changes or migration requirements are mentioned in the changelog entry.

## Publication Date

December 9, 2025
