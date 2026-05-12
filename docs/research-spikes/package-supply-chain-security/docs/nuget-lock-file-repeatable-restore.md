# NuGet Lock File: Repeatable Package Restore

- **Source**: https://github.com/NuGet/Home/wiki/Enable-repeatable-package-restore-using-lock-file
- **Retrieved**: 2026-05-12

## Overview

NuGet's lock file feature enables repeatable package restores across different machines and time periods. The default lock file is named `packages.lock.json`.

## Key Purposes

- **Floating versions**: Packages specified as `4.*` can be locked to specific versions
- **Repository changes**: Protects against version availability shifts on NuGet repositories
- **Transitive dependencies**: Documents complete package graphs including indirect dependencies
- **Consistency**: Ensures identical package restoration regardless of location or timing

## File Format

The `packages.lock.json` includes:
- Version field
- Metadata about dependencies organized by target framework
- Package details showing "type" (direct or transitive)
- "requested" versus "resolved" version information
- Content hash (SHA512) for integrity verification
- Complete dependency chains for each package

## Enabling Lock Files

Three methods:

1. **Presence-based**: Simply having `packages.lock.json` in the project
2. **Central management**: Automatic when using centrally managed packages
3. **Property-based**: Setting `RestorePackagesWithLockFile` to `true`

## Configuration Options

| Option | Values | Purpose |
|--------|--------|---------|
| `RestorePackagesWithLockFile` | true/false (default: false) | Enables lock file usage |
| `RestoreLockedMode` | true/false (default: false) | Fails if lock file is out of sync |
| `NuGetLockFilePath` | Custom path | Specifies alternate lock file location |

## CI/CD Integration

The `RestoreLockedMode` setting is critical for CI. When set to `true`, the restore command "will fail if the lock file is out of sync." This prevents unintended package updates during automated builds.

Command line equivalents:
- `--use-lock-file`: Enables lock file with restore
- `--locked-mode`: Enforces lock file synchronization
- `--force-evaluate`: Recomputes dependencies and updates lock file

## Out-of-Sync Conditions

A lock file becomes out-of-sync when:
- Project dependencies differ from lock file entries
- Package references are added, removed, or modified
- Target framework changes occur
- Runtime identifier modifications happen

## Restore Behavior

When `RestoreLockedMode` is disabled (default):
- Restore uses the lock file if synchronized
- Out-of-sync conditions trigger automatic updates with warnings

When enabled:
- Builds fail if synchronization issues exist
- Prevents unexpected dependency changes in CI environments
