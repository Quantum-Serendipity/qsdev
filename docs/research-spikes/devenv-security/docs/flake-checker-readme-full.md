<!-- Source: https://raw.githubusercontent.com/DeterminateSystems/flake-checker/main/README.md -->
<!-- Retrieved: 2026-05-12 -->

# Nix Flake Checker - Full README

## Overview
Nix Flake Checker is a utility from Determinate Systems that validates the health of `flake.lock` files in Nix projects. The tool's primary objective is ensuring projects maintain current, supported Nixpkgs versions.

## Installation & Basic Usage

```shell
nix run github:DeterminateSystems/flake-checker

# With explicit path specification
nix run github:DeterminateSystems/flake-checker /path/to/flake.lock
```

## Default Health Checks

Three automatic validations:

1. Git references belong to the supported branches list
2. Nixpkgs dependencies are under 30 days old
3. Dependencies originate from the NixOS GitHub organization

## Supported Branches

Current supported Nixpkgs branches include: `nixos-25.05`, `nixos-25.05-small`, `nixos-25.11`, `nixos-25.11-small`, `nixos-unstable`, `nixos-unstable-small`, `nixpkgs-25.05-darwin`, `nixpkgs-25.11-darwin`, and `nixpkgs-unstable`.

## Configuration Options

Three flags control checker behavior (all enabled by default):

| Flag | Environment Variable | Purpose |
|------|---------------------|---------|
| `--check-outdated` | `NIX_FLAKE_CHECKER_CHECK_OUTDATED` | Validate input recency |
| `--check-owner` | `NIX_FLAKE_CHECKER_CHECK_OWNER` | Verify NixOS ownership |
| `--check-supported` | `NIX_FLAKE_CHECKER_CHECK_SUPPORTED` | Confirm supported refs |

## Policy Conditions via CEL

Custom validation rules use Common Expression Language syntax:

```shell
flake-checker --condition "numDaysOld < 365"
```

Available variables: `gitRef`, `numDaysOld`, `owner`, `supportedRefs`, and `refStatuses`.

Recommended baseline condition: `supportedRefs.contains(gitRef) && numDaysOld < 30 && owner == 'NixOS'`

## GitHub Actions Integration

```yaml
checks:
  steps:
    - uses: actions/checkout@v6
    - name: Check Nix flake Nixpkgs inputs
      uses: DeterminateSystems/flake-checker-action@main
```

The GitHub Actions version returns exit code 0 by default and reports findings via Markdown summaries.

## Telemetry Control
Disable diagnostic reporting with `--no-telemetry` flag or `FLAKE_CHECKER_NO_TELEMETRY=true` environment variable.

## Rust Library Integration
A `parse-flake-lock` crate enables `flake.lock` parsing in Rust projects:

```toml
[dependencies]
parse-flake-lock = { git = "https://github.com/DeterminateSystems/flake-checker", branch = "main" }
```
