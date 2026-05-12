# Bazel Lockfile Documentation

- **Source**: https://bazel.build/external/lockfile
- **Retrieved**: 2026-05-12

## Overview

Bazel's lockfile feature records specific dependency versions and module resolution results to enable reproducible builds. The lockfile is automatically generated and updated during the build process.

## Lockfile Location and Generation

The primary lockfile is created at the workspace root as `MODULE.bazel.lock`. It captures dependencies included in the current build invocation.

## Lockfile Modes

The `--lockfile_mode` flag controls behavior:

- **update (default)**: Uses lockfile information to skip downloads of known registry files and avoid re-evaluating extensions whose results are still up-to-date
- **refresh**: Similar to update, but refreshes mutable information like yanked versions approximately hourly
- **error**: Fails if information is missing or outdated; never modifies the lockfile or performs network requests during resolution
- **off**: The lockfile is neither checked nor updated

## Lockfile Contents

Two primary sections:

### 1. Registry File Hashes
Contains checksums of all remote registry files accessed during module resolution, enabling reproducible results.

### 2. Module Extensions
Maps extension identifiers to their inputs and outputs. Each entry includes:
- `bzlTransitiveDigest`: Hash of the extension implementation and transitively loaded files
- `usagesDigest`: Hash of extension usages in the dependency graph
- `generatedRepoSpecs`: Repositories created by the extension

## CI/CD Enforcement

Use `--lockfile_mode=error` in CI to fail builds if the lockfile is out of date or missing required information. This prevents network requests during resolution and ensures reproducibility.

## Best Practices

1. Regularly update via `bazel mod deps --lockfile_mode=update`
2. Commit the lockfile to version control for team consistency
3. Use `bazelisk` with a `.bazelversion` file to ensure matching Bazel versions

## Merge Conflict Resolution

Bazel provides a custom git merge driver. Simple conflicts in `registryFileHashes` and `selectedYankedVersions` can be resolved by keeping entries from both sides. Other conflicts should be resolved by resetting the lockfile and running `bazel mod deps`.
