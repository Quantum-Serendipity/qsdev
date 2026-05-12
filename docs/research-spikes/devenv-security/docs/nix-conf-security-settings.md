# nix.conf Security-Related Settings
- **Source**: https://nix.dev/manual/nix/2.19/command-ref/conf-file
- **Retrieved**: 2026-05-12

## sandbox
**Type:** Boolean/String
**Default:** `true` (Linux), `false` (other platforms)
**Security Function:** Isolates builds from the filesystem, allowing access only to Nix store dependencies, temporary directories, and configured paths. Prevents undeclared dependencies and runs builds in private namespaces on Linux.

## restrict-eval
**Type:** Boolean
**Default:** `false`
**Security Function:** When enabled, prevents evaluator access to files outside `builtins.nixPath` or URIs outside `allowed-uris`, ensuring expressions cannot access arbitrary system resources.

## allowed-uris
**Type:** List
**Default:** Empty
**Security Function:** Specifies URI prefixes permitted in restricted evaluation mode, controlling network access for functions like `fetchGit`.

## trusted-substituters
**Type:** List (store URLs)
**Default:** Empty
**Security Function:** Lists substituters that unprivileged users can enable, restricting which binary caches regular users can access through the daemon.

## trusted-public-keys
**Type:** Whitespace-separated list
**Default:** `cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY=`
**Security Function:** "At least one of the following condition must be met for Nix to accept copying a store object from another Nix store" - signed with trusted keys, disabled signature requirement, or output-addressed paths.

## require-sigs
**Type:** Boolean
**Default:** `true`
**Security Function:** Mandates cryptographic signatures on non-content-addressed paths added to store, preventing use of unsigned substitutes unless explicitly disabled.

## trusted-users
**Type:** Whitespace-separated usernames
**Default:** `root`
**Security Function:** Grants elevated daemon permissions including ability to specify substituters and import unsigned NARs. "Adding a user to `trusted-users` is essentially equivalent to giving that user root access."

## allowed-users
**Type:** Whitespace-separated usernames
**Default:** `*`
**Security Function:** Controls which users can connect to the Nix daemon, with trusted users always permitted regardless of this setting.

## sandbox-paths
**Type:** List
**Default:** Empty
**Security Function:** Specifies filesystem paths bind-mounted into sandboxes, controlling what build environments can access beyond standard isolation.

## sandbox-fallback
**Type:** Boolean
**Default:** `true`
**Security Function:** Determines whether to disable sandboxing when kernel lacks support, choosing between reduced isolation or build failure.

## filter-syscalls
**Type:** Boolean
**Default:** `true`
**Security Function:** Prevents dangerous system calls like setuid/setgid file creation and ACL/extended attribute manipulation unless explicitly disabled.

## secret-key-files
**Type:** Whitespace-separated file paths
**Default:** Empty
**Security Function:** Contains private keys for signing locally-built paths, enabling distribution to other users via corresponding public keys in their configurations.
