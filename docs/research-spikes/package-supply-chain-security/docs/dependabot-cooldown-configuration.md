# Dependabot Cooldown Configuration — GitHub Docs

- **Source URL**: https://docs.github.com/en/code-security/reference/supply-chain-security/dependabot-options-reference
- **Retrieved**: 2026-05-12

## Overview

The `cooldown` option enables users to delay dependency updates for a configurable number of days. It "allows updates to be delayed for a configurable number of days" and is exclusive to version updates — not security updates.

## Configuration Parameters

### Primary Settings

**`default-days`**
- Optional parameter establishing a baseline cooldown period
- Applies to dependencies without specific rules
- Used when `semver-*-days` parameters aren't defined

**`semver-major-days`**
- Optional cooldown for major version updates
- Applies only to package managers supporting semantic versioning
- Takes precedence over `default-days` for major releases

**`semver-minor-days`**
- Optional cooldown for minor version updates
- Takes precedence over `default-days` for minor releases

**`semver-patch-days`**
- Optional cooldown for patch version updates
- Takes precedence over `default-days` for patch releases

### Inclusion/Exclusion Lists

**`include`**
- Specifies which dependencies receive cooldown protection
- Supports wildcard matching with `*`
- Maximum 150 items allowed

**`exclude`**
- Specifies dependencies exempt from cooldown
- Supports wildcard matching with `*`
- Maximum 150 items allowed
- Takes precedence over `include` list

## Behavior & Workflow

1. Dependabot checks for updates per the defined `schedule.interval`
2. Cooldown settings are evaluated
3. If a new release falls within cooldown, that version update is skipped
4. Dependencies past their cooldown period are updated to latest version
5. Standard `versioning-strategy` settings apply after cooldown ends

## SemVer Support by Package Manager

The following package managers support semantic versioning for granular cooldown control:

Bundler, Bun, Cargo, Composer, Dotnet SDK, Elm, Gomod, Gradle, Hex, Julia, Maven, NPM/Yarn, NuGet, OpenTofu, Pip, Pub, Swift, UV

Package managers **without** SemVer support (Bazel, Docker, Docker Compose, GitHub Actions, Gitsubmodule, Helm, Terraform, Devcontainers) can only use `default-days`.

## Key Constraints

- Cooldown is **version-update only** — security updates bypass cooldown entirely
- The `exclude` list always takes precedence; overlapping dependencies are excluded from cooldown
- Cooldown days must be between 1 and 90
- If SemVer parameters are undefined, `default-days` becomes the fallback

## Configuration Example

```yaml
version: 2
updates:
  - package-ecosystem: "npm"
    directory: "/"
    schedule:
      interval: "daily"
    cooldown:
      default-days: 5
      semver-major-days: 30
      semver-minor-days: 10
      semver-patch-days: 2
      include: ["*"]
      exclude: ["express", "lodash*"]
```

In this configuration, most dependencies wait 5 days before updating, major versions wait 30 days, while express and lodash-related packages update immediately.
