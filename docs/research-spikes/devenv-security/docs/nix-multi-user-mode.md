# Nix Multi-User Mode: Security Architecture
- **Source**: https://nix.dev/manual/nix/stable/installation/multi-user
- **Retrieved**: 2026-05-12

## Daemon-Based Privilege Separation

The Nix multi-user system operates through a daemon architecture: "the Nix store and database are owned by some privileged user (usually `root`) and builders are executed under special user accounts (usually named `nixbld1`, `nixbld2`, etc.)." When unprivileged users invoke Nix commands affecting the store, these operations are delegated to the daemon, which handles execution with appropriate permissions.

## Build User Accounts

Build users function as isolated execution contexts. The documentation specifies that "build users are the special UIDs under which builds are performed. They should all be members of the _build users group_ `nixbld`." Importantly, these accounts should have no other group memberships. The typical setup creates 10 build user accounts, though this number can be adjusted based on expected concurrent build volume.

## Trust Model and Binary Cache Restrictions

A critical limitation exists in the trust model: "only root and a set of trusted users specified in `nix.conf` can specify arbitrary binary caches." This means unprivileged users can install packages from custom Nix expressions but cannot directly source pre-built binaries from untrusted locations.

## Unprivileged User Capabilities and Constraints

Unprivileged users operate within defined boundaries. They can request package installations from arbitrary expressions, but the actual build execution occurs under build user accounts controlled by the daemon. They cannot directly manipulate the store, manage the database, or configure binary cache sources.

## Access Control

Socket-level access restrictions provide additional security: "users who are not in the `nix-users` group cannot connect to the Unix domain socket `/nix/var/nix/daemon-socket/socket`, so they cannot perform Nix operations." This prevents unauthorized daemon access entirely.
