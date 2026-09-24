---
name: version-sentinel
description: Use when adding, bumping, or changing a dependency in a manifest (package.json, requirements.txt, pyproject.toml, Cargo.toml, a .csproj, ...). Explains how to check the change with the version-sentinel MCP tools.
---

# Version Sentinel — Workflow

The `version-sentinel` MCP server (`mcp__version-sentinel__*`) gives you
**advisory** checks on dependency versions. It does not block edits or
installs by itself — the package guard hook is the enforcement layer — so
run these checks yourself whenever you change a dependency.

## Before changing a dependency

1. **Look up the latest version** of the package on its upstream registry
   (npmjs.com, pypi.org, crates.io, nuget.org, ...). Use `WebSearch`, or
   `WebFetch` on the registry URL, or context7's `query-docs` tool.
2. **See what the project uses today** with `check_versions` — it scans the
   manifest files and reports the declared dependency versions.

## After changing a dependency

1. Run `detect_drift` to compare lockfile entries against the manifest
   declarations. Resolve any drift (for example, regenerate the lockfile
   with the project's package manager) before committing.
2. Commit the manifest and the lockfile together.

If you intentionally pin an older version (CVE mitigation, compatibility, a
private registry), say so and give the reason in your commit message or PR
description so reviewers see it was deliberate.

## Coverage

`manifest_coverage` lists the ecosystems Version-Sentinel tracks and the ones
it does not. Manifests it does not cover (also listed in
`.version-sentinel/ignore`) must be reviewed manually.

## History

`version_history` reads the event log at `.version-sentinel/events.jsonl`, a
timeline of recorded version changes. An absent log simply means no events
have been recorded yet.
