# Nix Build Sandboxing: Comprehensive Overview
- **Source**: https://discourse.nixos.org/t/what-is-sandboxing-and-what-does-it-entail/15533
- **Retrieved**: 2026-05-12

## What Sandboxing Is

In the Nix context, sandboxing isolates each build process in a restricted environment to enhance reproducibility. "Sandbox allows you to build derivations in an empty file system, without access to the internet, and on a perfectly empty environment."

The core purpose is preventing builds from depending on dynamically changing external sources, particularly files in system directories like `/usr/bin` that may reference different software versions over time.

## Technical Implementation

Sandboxing uses Linux namespaces to create isolation:

- **Separate process IDs** for build jobs
- **Distinct namespaces** for mount, network, IPC, and UTS operations
- **Private versions** of `/proc`, `/dev`, `/dev/shm`, and `/dev/pts`

These constraints ensure builds function as pure operations without side effects or external dependencies.

## Permitted Input Sources

Sandboxed builds can only access files from:

- The Nix store
- Temporary build directories (typically located in `/tmp`, named like `nix-build-<pname>-<version>.drv-<index>`)
- Paths specified in the `sandbox-paths` configuration option
- Results from fetch commands (fixed-output derivations)

**Note:** Fixed-output derivations bypass network isolation to enable downloading.

## Configuration Options

The `sandbox` setting accepts three values:

- **`true`**: Enables sandboxing
- **`false`**: Disables sandboxing
- **`relaxed`**: Exempts fixed-output derivations and builds with `__noChroot` attributes

## Platform Support and Default Behavior

- **Linux**: Fully supported; enabled by default in NixOS, though disabled by default in raw Nix
- **macOS**: Partial support with documented issues
- **Windows**: Not supported

## Important Constraints

Sandboxing requires Nix to run as root, utilizing the "build users" feature to execute actual builds under different users.
