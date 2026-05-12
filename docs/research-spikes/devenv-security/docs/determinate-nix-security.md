# Nix Security Overview from Determinate Systems
- **Source**: https://manual.determinate.systems/installation/nix-security.html
- **Retrieved**: 2026-05-12

## Security Model

Determinate Systems follows a multi-user security architecture. As stated: "Nix follows a multi-user security model in which all users can perform package management operations." The system prevents privilege escalation attacks by ensuring users cannot compromise others' packages.

## Daemon Architecture & Trusted Users

The security boundary is maintained through a daemon-based approach where:

- The Nix store is owned by a privileged user (typically root)
- Build operations run under special, unprivileged accounts (nixbld1, nixbld2, etc.)
- Regular users' commands are forwarded to a daemon that executes privileged operations

Important limitation noted: "only root and a set of trusted users specified in nix.conf can specify arbitrary binary caches." This restricts which users can designate package sources.

## Access Control Recommendations

Determinate Systems suggests controlling daemon access via filesystem permissions on /nix/var/nix/daemon-socket, allowing administrators to create user groups with restricted Nix operation privileges.

## Upstream Improvements

The provided documentation does not explicitly detail improvements Determinate Systems has made over upstream Nix -- it primarily documents the standard multi-user security model.
