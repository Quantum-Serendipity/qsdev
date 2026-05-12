# Package Managers Need to Cool Down — Andrew Nesbitt

- **Source URL**: https://nesbitt.io/2026/03/04/package-managers-need-to-cool-down.html
- **Retrieved**: 2026-05-12

## Core Argument

The article presents dependency cooldowns as a critical security measure. Cooldowns delay package installation until a minimum period has passed since publication, creating a window for community detection of malicious versions before automated tools pull them into projects.

## Key Statistics

William Woodruff's analysis examined "ten supply chain attacks" with a crucial finding: "eight had windows of opportunity under a week." This demonstrates that "even a modest cooldown of seven days would have blocked most of them from reaching end users."

## The Core Problem

Language package managers distribute new versions instantly—"running `npm publish` or `gem push` makes a package installable worldwide in seconds." When dependency bots run during this window, malicious code reaches projects without human review.

## Implementation Landscape

**JavaScript ecosystem (fastest adoption):**
- pnpm: `minimumReleaseAge` (v10.16, September 2025)
- Yarn: `npmMinimalAgeGate` (v4.10.0, September 2025)
- Bun: `minimumReleaseAge` (v1.3, October 2025)
- npm: `min-release-age` (v11.10.0, February 2026)
- Deno: `--minimum-dependency-age`

**Python:**
- uv: `--exclude-newer` with relative durations (v0.9.17, December 2025)
- pip: `--uploaded-prior-to` (v26.0, January 2026)

**Ruby:**
- gem.coop: 48-hour delay via registry-level enforcement

**Rust:**
- Cargo: Registry-side infrastructure stabilized (v1.94, March 2026)

**Pending:**
- Go, Bundler, Composer, Maven, Gradle, Swift Package Manager, Dart pub, Elixir Hex

## Dependency Update Tools

- **Renovate**: Long-standing `minimumReleaseAge` support; Mend Renovate 42 made 3-day minimum default for npm
- **Dependabot**: Cooldown block with `default-days` and semver-level overrides (July 2025)
- **Snyk**: Non-configurable 21-day cooldown
- **npm-check-updates**: `--cooldown` parameter with duration suffixes

## Configuration Fragmentation

Ten different naming conventions exist across tools: `cooldown`, `minimumReleaseAge`, `min-release-age`, `npmMinimalAgeGate`, `exclude-newer`, `stabilityDays`, `uploaded-prior-to`, `min-age`, `cooldown-days`, `minimum-dependency-age`.

## Technical Considerations

**Absolute vs. relative timestamps:** Absolute timestamps enable reproducibility across time; relative durations create sliding security windows. pip and uv support both; most JavaScript tools implement relative durations only.

**Duration parsing complexity:** Systems must handle timezone variations and calendar unit conversions. The article notes: "A few hours of timezone drift can determine whether a package published six days and twenty-two hours ago passes the cooldown check or not."

## System Package Manager Comparison

Traditional system managers (apt, brew) already separate publishing from distribution through mandatory human review. Debian implements automated 2-10 day migration windows across release channels, providing built-in protection language managers now retrofit through cooldowns.
