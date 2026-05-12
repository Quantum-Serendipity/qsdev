<!-- Source: https://discourse.nixos.org/t/what-is-sandboxing-and-what-does-it-entail/15533 -->
<!-- Retrieved: 2026-05-12 -->

# Nix Sandboxing: Technical Overview

## What Sandboxing Does

Sandboxing in Nix creates isolated build environments by constraining build inputs to enhance reproducibility. It ensures "the build step is a pure function without side effects or visibility to the outside world."

The mechanism works by isolating builds from dynamic input sources — preventing dependency on files in global directories like `/usr/bin` where executables may change over time.

## Allowed Input Sources

Sandboxed builds can only access:
- The Nix store
- Temporary build directories (typically under `/tmp`, named like `nix-build-<pname>-<version>.drv-<index>`)
- Paths configured via the `sandbox-paths` option
- Fixed-output derivations (results of `fetch*` commands)
- Linux-specific private versions of `/proc`, `/dev`, `/dev/shm`, `/dev/pts`

## Technical Implementation (Linux Namespaces)

On Linux, builds run with separate process IDs and isolated mount, network, IPC, and UTS namespaces. However, fixed-output derivations notably don't use private network namespaces, allowing internet access.

Regular derivations have no network access whatsoever.

## The sandbox=relaxed Mode

When set to relaxed, two derivation types skip sandboxing entirely:
- Fixed-output derivations
- Derivations with `__noChroot` attribute set to true

## Platform Support & Defaults

**Linux:** Fully supported; enabled by default on NixOS (but NOT enabled by default in upstream Nix itself — NixOS configuration enables it).

**macOS:** Partial support with documented issues and lacking Linux namespace features.

**BSD/Windows:** Limited or no support.

## Configuration Requirements

Sandboxing requires Nix to run as root, utilizing the "build users" feature for actual builds under different user accounts.

## What It Does NOT Protect

- Nix evaluation (the `.nix` file interpretation phase) is NOT sandboxed — it runs with the calling user's privileges
- Fixed-output derivations have full network access
- Trusted users can disable the sandbox entirely via `sandbox = false`
- The sandbox is about build reproducibility, not security isolation per se
