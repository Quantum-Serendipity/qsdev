<!-- Source: https://github.com/goreleaser/goreleaser/pull/2648 -->
<!-- Retrieved: 2026-05-15 -->

# GoReleaser PR #2648: SBOM Generation Feature

## Overview
This pull request, submitted by wagoodman (Alex Goodman, Anchore/Syft maintainer) and merged by caarlos0 on December 12, 2021, introduced SBOM (Software Bill of Materials) generation capabilities to GoReleaser.

## Purpose
The feature allows users to automatically generate SBOMs for artifacts during the release process. As described in the PR, it was designed to work in conjunction with existing signing functionality, enabling "SBOMs generated can be checksummed and referenced under the `signs` section."

## Configuration
The implementation introduced a new `sboms` configuration section. The basic usage pattern is:
```yaml
sboms:
  -
    id: binary-sbom
    artifacts: binary
```

## Technical Details
- **Size**: Marked as size/XXL (1000+ lines changed)
- **Changes**: Added 197 lines across multiple files, primarily:
  - New `internal/pipe/sbom/sbom.go` module (85.29% test coverage)
  - Updates to pipeline orchestration files
  - Documentation additions
  - Configuration schema updates
- **Coverage Impact**: Overall coverage increased by 0.02% to 84.95%

## Design Decisions

The implementation placed SBOM generation **before signing** in the pipeline sequence, allowing the generated SBOMs themselves to be signed as artifacts. A reviewer noted: "I like this! Many thanks for the PR!"

The feature was designed to be flexible, supporting multiple SBOM configurations with different artifact targets and generation IDs.

## Integration
The merged code was incorporated into milestone v1.2.0 (released late 2021/early 2022), suggesting this was positioned as a significant feature enhancement rather than a patch release.

## Key Insight: Pipeline Ordering
SBOM generation runs BEFORE signing in GoReleaser's pipeline. This means:
1. Artifacts are built
2. SBOMs are generated for those artifacts
3. Checksums are computed (covering both artifacts and SBOMs)
4. Signing happens last (covering checksums, which transitively cover everything)
