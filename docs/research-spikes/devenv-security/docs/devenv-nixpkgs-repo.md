<!-- Source: https://github.com/cachix/devenv-nixpkgs -->
<!-- Retrieved: 2026-05-12 -->

# devenv-nixpkgs Repository README

## Overview

The devenv-nixpkgs project provides "Battle-tested nixpkgs using devenv's extensive testing infrastructure." The repository maintains a curated version of nixpkgs optimized for use with the devenv development environment tool.

## Rolling Release Branch

Currently, the only supported release is the rolling branch, which "is based on nixpkgs-unstable plus any patches that improve the integrations and services offered by devenv."

## Usage

Users can integrate this into their `devenv.yaml` configuration with:

```yaml
inputs:
  nixpkgs:
    url: github:cachix/devenv-nixpkgs/rolling
    flake: false
```

## Patch Management

Patches are organized into two categories within `patches/default.nix`:

**Upstream patches**: These come from nixpkgs pull requests or unreleased fixes. The documentation advises against using `fetchpatch` for unmerged PRs due to potential force-pushes, but permits it for merged commits with immutable content.

**Local patches**: These are fixes not yet submitted to upstream nixpkgs. Contributors create patches using `git format-patch` and add them to the patches configuration.

## Overlays

For package-level modifications that don't require source patches, the repository uses `overlays/default.nix`. This approach is described as "more resilient to upstream changes than source patches."

## Test Results

The latest test summary shows:
- Total test jobs: 284
- Successful: 266
- Failed: 18
- Overall success rate: 93%

Platform-specific results range from 90.1% to 97.1% success rates across aarch64-linux, x86_64-linux, aarch64-darwin, and x86_64-darwin.

## CI and Deployment Workflow

Weekly automation (Mondays at 9:00 UTC) performs: nixpkgs updates, patch validation, test suite execution across all platforms, README updates with results, and automatic release PR creation to promote main to rolling branch.
