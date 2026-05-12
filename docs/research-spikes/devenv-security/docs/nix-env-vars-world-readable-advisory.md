<!-- Source: https://discourse.nixos.org/t/security-advisory-environment-variables-accessible-during-a-build-might-be-world-readable/52601 -->
<!-- Retrieved: 2026-05-12 -->

# Security Advisory: World-Readable Environment Variables in NixOS Builds

## Vulnerability Overview

A security issue affects NixOS builds where environment variables containing sensitive information may be exposed through world-readable files. The system creates an `env-vars` file to help debug broken builds, but this file could be accessed by any user if the temporary build directory lacks proper access restrictions.

## Technical Details

The vulnerability occurs specifically during impure builds. As described in the advisory: "If the temporary build directory is world-readable, the generated `env-vars` file is also accessible to everyone." This becomes problematic when sensitive data like credentials or secrets are present in environment variables during interactive sessions, such as when running `nix-shell` commands.

## Patches and Remediation

The issue has been addressed by modifying file permissions. The `env-vars` file is now created with `0600` permissions (readable only by the owner) rather than `0644` (world-readable). Patches were released across multiple versions:

- **Unstable channel**: Two pull requests addressing the issue and a Darwin-specific fix
- **24.05 branch**: A backport ensuring consistency across release versions

## Recommended Actions

Users should consider revoking any secrets that may have been exposed through previously generated `env-vars` files, particularly those created in shared or multi-user environments.
