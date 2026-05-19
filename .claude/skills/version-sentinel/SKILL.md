---
name: version-sentinel
description: Use when adding, bumping, or changing a dependency in package.json, requirements.txt, pyproject.toml, Cargo.toml, or a .csproj. Triggered automatically by version-sentinel's PreToolUse hook — this skill explains how to satisfy the hook.
---

# Version Sentinel — Workflow

The `version-sentinel` plugin blocks dependency changes until you've verified the package version against its upstream registry. Here's the required flow:

## When you see a BLOCKED message

If a tool call exits 2 with `BLOCKED: version-sentinel`, you must:

1. **Look up the latest version.** Use `WebSearch` first:
   - `npm`:      search `"<pkg> latest version site:npmjs.com"`
   - `pip`/`pyproject`: search `"<pkg> latest version site:pypi.org"`
   - `csproj`:   search `"<pkg> latest version site:nuget.org"`
   - `cargo`:    search `"<pkg> latest version site:crates.io"`

   If WebSearch is unavailable (non-US region), use `WebFetch` on the registry URL directly, or consult context7's `query-docs` tool for the package.

2. **Record the check.** Invoke:

       /vs-record <ecosystem> <pkg> <version-you-intend-to-install> <source-url>

   The source must be an `http(s)://` URL from your search OR prefixed with `intentional:` for deliberate pins.

3. **Retry the original edit or install.** The hook will see the fresh entry and let the tool call through.

## Intentional non-latest pins

If you genuinely intend to install an older version (CVE mitigation, compat, private registry), record with:

    /vs-record <ecosystem> <pkg> <version> "intentional: <brief reason>"

This passes the hook and is flagged as `intentional-pin` (not `DRIFT`) in `/check-versions` output.

## What NOT to do

- Don't fake a source URL you didn't actually see. The skill contract assumes honest reporting; v0.2 will probe the transcript to verify.
- Don't try to bypass the hook with `git commit --no-verify` or similar — the hook runs on `Edit`/`Write`/`Bash`, not on git.
- Don't `unset VS_DISABLE` without the user's awareness; that's an escape hatch for throwaway sessions, not normal flow.

## Audit command

`/check-versions` scans every manifest under the current directory and reports drift. Run it before tagging a release.
