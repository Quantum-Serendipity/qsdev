# Dependency Freshness and Update Workflow Research

## Research Question

Beyond Renovate (already planned), should gdev include dependency freshness checking (`qsdev outdated`) and coordinated update workflows?

## Current Landscape: Per-Ecosystem Outdated Commands

Every package manager already has an outdated command:

| Ecosystem | Command | Output |
|-----------|---------|--------|
| npm | `npm outdated` | Table: current, wanted, latest |
| pnpm | `pnpm outdated` | Same format |
| yarn | `yarn outdated` | Same format |
| pip/uv | `pip list --outdated` / `uv pip list --outdated` | Table with latest version |
| cargo | `cargo outdated` (third-party) | Table with SemVer analysis |
| go | `go list -m -u all` | List with available updates |
| dotnet | `dotnet list package --outdated` | Table per project |
| composer | `composer outdated` | Table with color-coded severity |
| bundler | `bundle outdated` | List with SemVer analysis |
| mix | `mix hex.outdated` | Table |

### The Polyglot Gap

No tool runs ALL of these across a polyglot project. A TypeScript + Python + Docker project requires three separate commands with three different output formats. Renovate handles this for CI but is async (PRs, not interactive).

### Age-Gating Proliferation

The dependency freshness story has evolved rapidly. As of early 2026, age-gating is built into most JavaScript package managers:
- pnpm: `minimumReleaseAge` (since v10.16, Sept 2025)
- Yarn: `npmMinimalAgeGate` (since v4.10.0, Sept 2025)
- Bun: `minimumReleaseAge` (since v1.3, Oct 2025)
- npm: `min-release-age` (since v11.10.0, Feb 2026)
- uv (Python): `--exclude-newer` (relative durations)
- Cargo: registry-side cooldowns (stabilized Cargo 1.94, March 2026)

The config key names are all different, which is exactly the kind of inconsistency gdev already addresses with per-ecosystem templates.

## Analysis: Should gdev Include `qsdev outdated`?

### Arguments For

1. **Unified view**: One command, all ecosystems, one format. "Are any of my dependencies outdated?" answered in 5 seconds instead of running 3 separate commands.
2. **Consulting context**: When picking up a stale client project, `qsdev outdated` immediately shows the maintenance debt.
3. **Security signal**: Outdated dependencies correlate with unpatched vulnerabilities. This complements OSV Scanner (which checks known CVEs) with a broader "freshness" signal.
4. **Natural extension**: gdev already knows which ecosystems are present (detection engine). Running the per-ecosystem outdated commands is straightforward.

### Arguments Against

1. **Renovate already does this**: Renovate's Dependency Dashboard provides an always-current view of outdated dependencies with PR creation. Running a local command duplicates this.
2. **Per-ecosystem commands already exist**: Developers know `npm outdated`. A wrapper adds one more command to remember.
3. **Output aggregation is hard**: Each tool's output format, versioning semantics, and "outdated" definition differ. Normalizing across 27 ecosystems is significant work.
4. **Stale data**: A local `outdated` check is a point-in-time snapshot. Renovate runs continuously.

### Verdict

**Include a thin wrapper, not a full aggregator.**

`qsdev outdated` should:
1. Detect active ecosystems (already built)
2. Run each ecosystem's native outdated command
3. Print results sequentially with ecosystem headers
4. Exit with non-zero if any ecosystem has outdated deps

It should NOT:
- Parse and normalize output formats
- Provide a unified table
- Track versions itself
- Duplicate Renovate's analysis

This is 50 lines of Go code per ecosystem -- detect, exec, print, check exit code. The value is "one command to check everything" without the complexity of "unified dependency analysis platform."

## Coordinated Updates: `qsdev update`

### The Update Dance

When gdev itself is updated, several things may need to change:
1. gdev binary (updated via self-update)
2. Generated configs (devenv.nix, settings.json, etc.) -- need regeneration
3. devenv inputs (Nix packages) -- need `devenv update`
4. Application dependencies (npm, pip, cargo) -- need per-ecosystem update

Today these are 4 separate operations. A coordinated update command would be:

```
$ qsdev update
[1/4] Checking for qsdev updates... v1.2.0 -> v1.3.0 available
[2/4] Regenerating configs for v1.3.0... 3 files updated
[3/4] Updating devenv inputs... 2 inputs updated
[4/4] Application dependencies... skipped (use qsdev update --deps or Renovate)
```

### Analysis

Steps 1-3 are safe and fast -- they update gdev's own managed artifacts. Step 4 (application dependencies) is dangerous for unattended execution and should be left to Renovate or manual developer action.

**Recommendation: Include `qsdev update` for steps 1-3 only.** This is the "keep gdev infrastructure current" command. Application dependency updates remain Renovate's domain.

## Breaking Change Detection

### What Would This Mean?

Detecting when a dependency update introduces breaking changes. For example:
- Major version bump in a direct dependency
- Deprecated API usage in current code
- Incompatible peer dependency requirements

### Analysis

This is **out of scope for gdev**. Breaking change detection requires:
- Semantic analysis of API surfaces
- Per-language type checking
- Changelog parsing

Tools that do this (Renovate's major/minor PR splitting, npm's `npm audit signatures`, cargo's SemVer checking) are ecosystem-specific and well-established. gdev should not reimplement them.

**Recommendation: Exclude.** Renovate's separation of major vs minor PRs is the right approach. gdev adds no value here.

## Summary

| Feature | Recommendation | Rationale |
|---------|---------------|-----------|
| `qsdev outdated` (thin wrapper) | **Include** | One command for all ecosystems, low complexity |
| `qsdev outdated` (unified analysis) | **Exclude** | Too complex, Renovate does this better |
| `qsdev update` (self + configs + devenv) | **Include** | Coordinated infrastructure update, safe |
| `qsdev update --deps` (app deps) | **Exclude** | Renovate's domain, dangerous for unattended |
| Breaking change detection | **Exclude** | Per-ecosystem tools do this, out of scope |

## Depth Checklist

- [x] Underlying mechanism explained -- per-ecosystem outdated commands, age-gating configs, update orchestration
- [x] Key tradeoffs -- unified view vs duplicating Renovate, thin wrapper vs full aggregator
- [x] Compared to alternatives -- Renovate Dashboard, per-ecosystem commands, mise outdated
- [x] Failure modes -- stale local data vs Renovate's continuous monitoring, breaking changes from uncoordinated updates
- [x] Concrete examples -- per-ecosystem command table, qsdev update output mockup, age-gating config names
- [x] Standalone-readable -- yes
