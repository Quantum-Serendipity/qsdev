# Source: https://github.com/KSEGIT/Version-Sentinel/blob/main/CHANGELOG.md
# Retrieved: 2026-05-12

# Changelog

All notable changes to version-sentinel.

## [0.2.1](https://github.com/KSEGIT/Version-Sentinel/compare/version-sentinel-v0.2.0...version-sentinel-v0.2.1) (2026-04-22)

### Bug Fixes

* **marketplace:** use relative source to avoid SSH clone ([b922e84](https://github.com/KSEGIT/Version-Sentinel/commit/b922e84a1dcb09cb8145d653361689c6daf6f363))

## [0.2.0](https://github.com/KSEGIT/Version-Sentinel/compare/version-sentinel-v0.1.0...version-sentinel-v0.2.0) (2026-04-22)

### Features

- add GitHub Actions workflow for running tests
- add initial project structure and configuration files
- check-sidecar.sh — exit-2 block with stderr for Claude
- /check-versions audit + registries.sh
- /vs-record slash command + shell backend
- detect-install-cmd.sh (Bash install commands)
- detect-manifest-edit.sh (Edit|Write|MultiEdit)
- wire hooks.json (Edit|Write|MultiEdit + Bash matchers)
- sidecar read/write with dedupe + auto-gitignore
- Cargo.toml parser (path/git deps skipped)
- csproj/fsproj/vbproj PackageReference parser
- install-command parser (npm/pip/cargo/dotnet)
- npm package.json parser (all 4 dep sections)
- path->ecosystem dispatch + manifest-set diff
- pip requirements.txt parser
- pyproject.toml parser (PEP 621 + Poetry + uv)
- plugin hardening, release automation, CI matrix
- plugin.json + marketplace.json + LICENSE
- version-sentinel SKILL.md workflow guide

### Bug Fixes

- address CodeRabbit review feedback
- macOS bash 3.2 compat + shellcheck severity
- guard sidecar write against jq failure + test robustness
- run.sh integration smoke path (cwd-relative after cd)

## [Unreleased]

### Added
- userConfig block in plugin.json for disable (bool) and window_hours (number)
- SessionStart hook running prereq-check.sh
- PostToolUse:Bash hook running auto-record.sh
- bin/vs-record thin wrapper
- agents/version-reviewer.md
- CONTRIBUTING.md
- release-please workflow
- changelog-check workflow
- 3-OS CI matrix

## [0.1.0] — 2026-04-17

### Added
- PreToolUse hooks for Edit, Write, MultiEdit, and Bash tools
- Manifest parsers for 5 ecosystem file families
- Install-command parsers for npm/pnpm/yarn/bun, pip/pip3, poetry, uv, cargo, dotnet
- /vs-record and /check-versions slash commands
- Sidecar state with auto-gitignore and last-write-wins dedupe
- VS_DISABLE=1 escape hatch and .version-sentinel/ignore pattern file
- Fail-open philosophy
- Bash test harness with 14+ tests

### Known limitations
- PackageReference child Version element not parsed (attribute only)
- pyproject.toml range specifiers reduced to lower bound
- Offline use: /check-versions requires network; hook blocks still work
- v0.1 trusts model to search before /vs-record; v0.2 will probe transcript
